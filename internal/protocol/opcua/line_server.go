package opcua

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gopcua/opcua/id"
	"github.com/gopcua/opcua/server"
	"github.com/gopcua/opcua/ua"
	"github.com/rs/zerolog/log"

	"machine-simulator-2/internal/config"
	"machine-simulator-2/internal/line"
)

// LineJobs is what a line server needs from the line manager.
type LineJobs interface {
	LoadJob(lineID string, spec line.JobSpec) (line.JobStatus, error)
	StartJob(lineID string) (line.JobStatus, error)
	InterruptJob(lineID string) (line.JobStatus, error)
	FinishJob(lineID string) (line.JobStatus, error)
	Job(lineID string) (line.JobStatus, error)
	Ledger(lineID string) (line.Ledger, error)
	AddMaterial(lineID, material string, qty float64) (line.JobStatus, error)
}

// Job commands written to the job_command node.
const (
	CommandLoad      int32 = 1
	CommandStart     int32 = 2
	CommandInterrupt int32 = 3
	CommandFinish    int32 = 4
)

type lineNode struct {
	id       uint32
	name     string
	dataType string
	writable bool
}

// Line server nodes. 1–5 are setpoints a client writes; the server never
// overwrites them, so a read-back after a write sees the written value.
// The rest show the job as the line actually runs it.
var lineNodes = []lineNode{
	{1, "job_order_id", "String", true},
	{2, "job_operation_id", "String", true},
	{3, "job_product_id", "String", true},
	{4, "job_target_qty", "Int32", true},
	{5, "job_command", "Int32", true},
	{6, "job_state", "String", false},
	{7, "job_command_result", "String", false},
	{8, "job_good", "Int32", false},
	{9, "job_scrap", "Int32", false},
	{10, "job_started", "Int32", false},
	{11, "parts_in_process", "Int32", false},
	{12, "wip_scrapped", "Int32", false},
	{13, "active_order_id", "String", false},
	{14, "active_operation_id", "String", false},
	{15, "active_product_id", "String", false},
	{16, "active_target_qty", "Int32", false},
	// Components of the next load: material id and quantity per part, up to four.
	{17, "job_component_1_id", "String", true},
	{18, "job_component_1_per_part", "Double", true},
	{19, "job_component_2_id", "String", true},
	{20, "job_component_2_per_part", "Double", true},
	{21, "job_component_3_id", "String", true},
	{22, "job_component_3_per_part", "Double", true},
	{23, "job_component_4_id", "String", true},
	{24, "job_component_4_per_part", "Double", true},
	// Staging into the line-side store: write id and quantity, then
	// material_command = 1. Every write of the command adds once.
	{25, "material_add_id", "String", true},
	{26, "material_add_qty", "Double", true},
	{27, "material_command", "Int32", true},
	{28, "material_missing", "String", false},
	{29, "component_1_level", "Double", false},
	{30, "component_2_level", "Double", false},
	{31, "component_3_level", "Double", false},
	{32, "component_4_level", "Double", false},
}

const (
	nodeJobCommand      uint32 = 5
	nodeMaterialCommand uint32 = 27
	maxComponents              = 4
)

// CommandAddMaterial is material_command's only command.
const CommandAddMaterial int32 = 1

// jobCommand is a job_command write with the setpoints as they were at that
// moment, so later setpoint writes cannot change a queued load.
type jobCommand struct {
	cmd  int32
	spec line.JobSpec
	err  error // set when the setpoints were unusable

	// material is set for a material_command write.
	material bool
	matID    string
	matQty   float64
}

// writeHookNamespace wraps the line's namespace so client writes are seen in
// the request goroutine, in order, including writes of an unchanged value.
type writeHookNamespace struct {
	*server.NodeNameSpace
	nodeMu  *sync.Mutex // the line server's node lock
	onWrite func(nodeID uint32, v interface{})
}

// SetAttribute applies a client write and reports accepted value writes.
func (w *writeHookNamespace) SetAttribute(id *ua.NodeID, attr ua.AttributeID, val *ua.DataValue) ua.StatusCode {
	w.nodeMu.Lock()
	st := w.NodeNameSpace.SetAttribute(id, attr, val)
	w.nodeMu.Unlock()
	if st == ua.StatusOK && attr == ua.AttributeIDValue && val != nil && val.Value != nil {
		w.onWrite(id.IntID(), val.Value.Value())
	}
	return st
}

// LineServers runs one OPC UA server per line instance: the "line PLC" an
// MES or UMH loads jobs onto.
type LineServers struct {
	cfg  config.OPCUAAdapterConfig
	jobs LineJobs

	mu      sync.Mutex
	servers []*lineServer
}

type lineServer struct {
	lineID string
	port   int
	srv    *server.Server
	ns     *writeHookNamespace
	nodes  map[uint32]*server.Node

	// nodeMu serialises our own node updates (refresh and command results);
	// the library does not synchronise node values.
	nodeMu sync.Mutex

	mu        sync.Mutex // guards setpoints and result
	setpoints map[uint32]interface{}
	result    string

	commands chan jobCommand

	// started is set once the server runs; before that the library has no
	// subscription service and change notifications would panic.
	started atomic.Bool
}

// NewLineServers creates line servers on cfg.LineBasePort and up.
func NewLineServers(cfg config.OPCUAAdapterConfig, jobs LineJobs) *LineServers {
	return &LineServers{cfg: cfg, jobs: jobs}
}

// Register creates the server for a line. Call before Start.
func (ls *LineServers) Register(lineID string) int {
	ls.mu.Lock()
	defer ls.mu.Unlock()

	port := ls.cfg.LineBasePort + len(ls.servers)
	srv := server.New(
		server.EndPoint(ls.cfg.Hostname, port),
		server.ServerName("Line_"+lineID),
	)
	base := server.NewNodeNameSpace(srv, "urn:machine-simulator:line:"+lineID)
	ns := &writeHookNamespace{NodeNameSpace: base}
	// NewNodeNameSpace registered base; swap in the wrapper at the same index
	// so the Write service calls our SetAttribute. Namespaces() returns the
	// server's own slice.
	srv.Namespaces()[base.ID()] = ns

	objects := ua.NewNumericNodeID(ns.ID(), id.ObjectsFolder)
	nodes := make(map[uint32]*server.Node, len(lineNodes))
	for _, ln := range lineNodes {
		access := byte(ua.AccessLevelTypeCurrentRead)
		if ln.writable {
			access |= byte(ua.AccessLevelTypeCurrentWrite)
		}
		nodeID := ua.NewNumericNodeID(ns.ID(), ln.id)
		n := server.NewNode(nodeID, server.Attributes{
			ua.AttributeIDBrowseName:              server.DataValueFromValue(&ua.QualifiedName{NamespaceIndex: ns.ID(), Name: ln.name}),
			ua.AttributeIDDisplayName:             server.DataValueFromValue(ua.NewLocalizedText(ln.name)),
			ua.AttributeIDDescription:             server.DataValueFromValue(ua.NewLocalizedText(ln.name)),
			ua.AttributeIDNodeClass:               server.DataValueFromValue(int32(ua.NodeClassVariable)),
			ua.AttributeIDDataType:                server.DataValueFromValue(dataTypeNodeID(ln.dataType)),
			ua.AttributeIDValueRank:               server.DataValueFromValue(int32(-1)),
			ua.AttributeIDAccessLevel:             server.DataValueFromValue(access),
			ua.AttributeIDUserAccessLevel:         server.DataValueFromValue(access),
			ua.AttributeIDMinimumSamplingInterval: server.DataValueFromValue(float64(0)),
			ua.AttributeIDHistorizing:             server.DataValueFromValue(false),
		}, server.References{
			{ReferenceTypeID: ua.NewNumericNodeID(0, id.HasComponent), IsForward: false, NodeID: &ua.ExpandedNodeID{NodeID: objects}},
			{ReferenceTypeID: ua.NewNumericNodeID(0, id.HasTypeDefinition), IsForward: true, NodeID: &ua.ExpandedNodeID{NodeID: ua.NewNumericNodeID(0, id.BaseDataVariableType)}},
		}, nil)
		ns.AddNode(n)
		nodes[ln.id] = n
	}

	s := &lineServer{
		lineID: lineID, port: port, srv: srv, ns: ns, nodes: nodes,
		setpoints: make(map[uint32]interface{}),
		result:    "OK",
		// Commands queue while one is handled; 64 covers any realistic burst.
		commands: make(chan jobCommand, 64),
	}
	ns.nodeMu = &s.nodeMu
	ns.onWrite = s.onClientWrite
	for _, ln := range lineNodes {
		switch ln.dataType {
		case "String":
			s.set(ln.id, "")
		case "Double":
			s.set(ln.id, float64(0))
		default:
			s.set(ln.id, int32(0))
		}
	}
	s.refresh(ls.jobs)
	ls.servers = append(ls.servers, s)
	log.Info().Str("line", lineID).Int("port", port).Msg("OPC-UA: registered line server")
	return port
}

// Port returns the port of a line's server, or 0.
func (ls *LineServers) Port(lineID string) int {
	ls.mu.Lock()
	defer ls.mu.Unlock()
	for _, s := range ls.servers {
		if s.lineID == lineID {
			return s.port
		}
	}
	return 0
}

// Start runs the servers, the write handlers and the status refresh until ctx
// is cancelled.
func (ls *LineServers) Start(ctx context.Context) {
	ls.mu.Lock()
	servers := append([]*lineServer(nil), ls.servers...)
	ls.mu.Unlock()

	interval := time.Duration(ls.cfg.UpdateRateMS) * time.Millisecond
	if interval <= 0 {
		interval = 100 * time.Millisecond
	}
	for _, s := range servers {
		// Start listens and returns; serving continues in the library's goroutines.
		if err := s.srv.Start(ctx); err != nil {
			log.Error().Err(err).Str("line", s.lineID).Int("port", s.port).Msg("OPC-UA line server failed to start")
			continue
		}
		s.started.Store(true)
		go s.handleWrites(ctx, ls.jobs)
		go s.refreshLoop(ctx, ls.jobs, interval)
	}
	go func() {
		<-ctx.Done()
		for _, s := range servers {
			s.srv.Close()
		}
	}()
}

// set updates a node value without the client access check and notifies
// subscribers.
func (s *lineServer) set(nodeID uint32, v interface{}) {
	s.nodeMu.Lock()
	defer s.nodeMu.Unlock()
	n := s.nodes[nodeID]
	now := time.Now()
	_ = n.SetAttribute(ua.AttributeIDValue, &ua.DataValue{
		EncodingMask:    ua.DataValueValue | ua.DataValueStatusCode | ua.DataValueSourceTimestamp | ua.DataValueServerTimestamp,
		Value:           ua.MustVariant(v),
		Status:          ua.StatusOK,
		SourceTimestamp: now,
		ServerTimestamp: now,
	})
	if s.started.Load() {
		s.ns.ChangeNotification(n.ID())
	}
}

// setpoint returns the last value a client wrote to a writable node.
func (s *lineServer) setpoint(nodeID uint32) interface{} {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.setpoints[nodeID]
}

// onClientWrite runs in the server's request goroutine for every accepted
// value write.
func (s *lineServer) onClientWrite(nodeID uint32, v interface{}) {
	s.mu.Lock()
	s.setpoints[nodeID] = v
	if nodeID == nodeMaterialCommand {
		jc := jobCommand{material: true, matID: toString(s.setpoints[25])}
		cmd, ok := toInt(v)
		q, qok := toFloat(s.setpoints[26])
		switch {
		case !ok || int32(cmd) != CommandAddMaterial:
			jc.err = fmt.Errorf("%w: material_command must be 1 (add)", line.ErrInvalidJob)
		case !qok:
			jc.err = fmt.Errorf("%w: material_add_qty must be a number", line.ErrInvalidJob)
		}
		jc.matQty = q
		s.mu.Unlock()
		s.enqueue(jc)
		return
	}
	if nodeID != nodeJobCommand {
		s.mu.Unlock()
		return
	}
	jc := jobCommand{}
	cmd, ok := toInt(v)
	if !ok {
		jc.err = fmt.Errorf("%w: job_command must be an integer", line.ErrInvalidJob)
	}
	jc.cmd = int32(cmd)
	if jc.cmd == CommandLoad && jc.err == nil {
		target, ok := toInt(s.setpoints[4])
		if s.setpoints[4] == nil {
			target, ok = 0, true
		}
		if !ok {
			jc.err = fmt.Errorf("%w: job_target_qty must be an integer", line.ErrInvalidJob)
		}
		jc.spec = line.JobSpec{
			OrderID:     toString(s.setpoints[1]),
			OperationID: toString(s.setpoints[2]),
			ProductID:   toString(s.setpoints[3]),
			TargetQty:   target,
		}
		for k := 0; k < maxComponents && jc.err == nil; k++ {
			mat := toString(s.setpoints[uint32(17+2*k)])
			if mat == "" {
				continue
			}
			per, ok := toFloat(s.setpoints[uint32(18+2*k)])
			if !ok {
				jc.err = fmt.Errorf("%w: job_component_%d_per_part must be a number", line.ErrInvalidJob, k+1)
			}
			jc.spec.Components = append(jc.spec.Components, line.Component{Material: mat, PerPart: per})
		}
	}
	s.mu.Unlock()
	s.enqueue(jc)
}

func (s *lineServer) enqueue(jc jobCommand) {
	select {
	case s.commands <- jc:
	default:
		s.setResult("error: command queue full, write again")
	}
}

func (s *lineServer) refreshLoop(ctx context.Context, jobs LineJobs, interval time.Duration) {
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			s.refresh(jobs)
		}
	}
}

// refresh writes the read-only nodes from the line's job and ledger.
func (s *lineServer) refresh(jobs LineJobs) {
	js, err := jobs.Job(s.lineID)
	if err != nil {
		return
	}
	led, _ := jobs.Ledger(s.lineID)
	s.mu.Lock()
	result := s.result
	s.mu.Unlock()

	s.set(6, string(js.State))
	s.set(7, result)
	s.set(8, int32(js.Good))
	s.set(9, int32(js.Scrap))
	s.set(10, int32(js.Started))
	s.set(11, int32(js.PartsInProcess))
	s.set(12, int32(led.WIPScrapped))
	s.set(13, js.OrderID)
	s.set(14, js.OperationID)
	s.set(15, js.ProductID)
	s.set(16, int32(js.TargetQty))
	s.set(28, js.MaterialMissing)
	for k := 0; k < maxComponents; k++ {
		level := 0.0
		if k < len(js.Components) {
			level = js.Material[js.Components[k].Material]
		}
		s.set(uint32(29+k), level)
	}
}

// handleWrites runs the queued job commands in write order.
func (s *lineServer) handleWrites(ctx context.Context, jobs LineJobs) {
	for {
		select {
		case <-ctx.Done():
			return
		case jc := <-s.commands:
			err := jc.err
			if err == nil {
				err = s.run(jobs, jc)
			}
			s.setResult(resultText(err))
			s.refresh(jobs)
		}
	}
}

func (s *lineServer) run(jobs LineJobs, jc jobCommand) error {
	if jc.material {
		_, err := jobs.AddMaterial(s.lineID, jc.matID, jc.matQty)
		return err
	}
	switch jc.cmd {
	case CommandLoad:
		_, err := jobs.LoadJob(s.lineID, jc.spec)
		return err
	case CommandStart:
		_, err := jobs.StartJob(s.lineID)
		return err
	case CommandInterrupt:
		_, err := jobs.InterruptJob(s.lineID)
		return err
	case CommandFinish:
		_, err := jobs.FinishJob(s.lineID)
		return err
	case 0:
		return nil
	default:
		return fmt.Errorf("%w: unknown command %d (1 load, 2 start, 3 interrupt, 4 finish)", line.ErrInvalidJob, jc.cmd)
	}
}

func (s *lineServer) resultNow() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.result
}

func (s *lineServer) setResult(r string) {
	s.mu.Lock()
	s.result = r
	s.mu.Unlock()
	s.set(7, r)
}

func resultText(err error) string {
	switch {
	case err == nil:
		return "OK"
	case errors.Is(err, line.ErrInvalidTransition), errors.Is(err, line.ErrUnknownProduct), errors.Is(err, line.ErrInvalidJob):
		return "rejected: " + err.Error()
	default:
		return "error: " + err.Error()
	}
}

func toInt(v interface{}) (int, bool) {
	switch x := v.(type) {
	case int32:
		return int(x), true
	case int64:
		return int(x), true
	case int16:
		return int(x), true
	case uint16:
		return int(x), true
	case uint32:
		return int(x), true
	case float64:
		return int(x), x == float64(int(x))
	case float32:
		return int(x), x == float32(int(x))
	case string:
		i, err := strconv.Atoi(x)
		return i, err == nil
	default:
		return 0, false
	}
}

func toFloat(v interface{}) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case float32:
		return float64(x), true
	case nil:
		return 0, false
	}
	if i, ok := toInt(v); ok {
		return float64(i), true
	}
	return 0, false
}

func toString(v interface{}) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	default:
		return fmt.Sprint(x)
	}
}
