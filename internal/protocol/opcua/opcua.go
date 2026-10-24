package opcua

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gopcua/opcua/id"
	"github.com/gopcua/opcua/server"
	"github.com/gopcua/opcua/ua"
	"github.com/rs/zerolog/log"

	"machine-simulator-2/internal/config"
	"machine-simulator-2/internal/machine"
	"machine-simulator-2/internal/protocol"
)

// Adapter exposes machines via OPC-UA servers.
type Adapter struct {
	mu sync.RWMutex

	cfg      config.OPCUAAdapterConfig
	servers  map[string]*machineServer // machine ID -> server
	nextPort int
	ctx      context.Context
	cancel   context.CancelFunc
}

type machineServer struct {
	srv     *server.Server
	machine machine.Machine
	port    int
	ns      *server.NodeNameSpace
	nodes   map[string]*server.Node
	cancel  context.CancelFunc
	started bool
}

// New creates a new OPC-UA adapter.
func New(cfg config.OPCUAAdapterConfig) *Adapter {
	return &Adapter{
		cfg:      cfg,
		servers:  make(map[string]*machineServer),
		nextPort: cfg.BasePort,
	}
}

func (a *Adapter) Name() string { return "opcua" }

func (a *Adapter) Start(ctx context.Context) error {
	a.mu.Lock()
	a.ctx, a.cancel = context.WithCancel(ctx)

	// Start all registered machine servers
	for id, ms := range a.servers {
		if ms.started {
			continue
		}
		msCtx, msCancel := context.WithCancel(a.ctx)
		ms.cancel = msCancel
		// Start listens and returns. It must finish before the update loop
		// writes values: the library creates its subscription service in
		// Start and panics on change notifications before that.
		if err := ms.srv.Start(msCtx); err != nil {
			log.Error().Err(err).Str("machine", id).Int("port", ms.port).Msg("OPC-UA server failed to start")
			continue
		}
		ms.started = true
	}
	a.mu.Unlock()

	log.Info().Int("base_port", a.cfg.BasePort).Int("machines", len(a.servers)).Msg("OPC-UA adapter started")

	// Start update loop
	go a.updateLoop()

	return nil
}

func (a *Adapter) Stop() error {
	a.mu.Lock()
	if a.cancel != nil {
		a.cancel()
	}
	servers := make([]*machineServer, 0, len(a.servers))
	for _, s := range a.servers {
		servers = append(servers, s)
	}
	a.mu.Unlock()

	for _, s := range servers {
		if s.cancel != nil {
			s.cancel()
		}
		s.srv.Close()
	}

	log.Info().Msg("OPC-UA adapter stopped")
	return nil
}

func (a *Adapter) RegisterMachine(m machine.Machine, cfg protocol.AdapterMachineConfig) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	port := cfg.Port
	if port == 0 {
		port = a.nextPort
		a.nextPort++
	}

	srv := server.New(
		server.EndPoint(a.cfg.Hostname, port),
		server.ServerName(fmt.Sprintf("Machine_%s", m.ID())),
	)

	ns := server.NewNodeNameSpace(srv, fmt.Sprintf("urn:machine-simulator:%s", m.ID()))

	objectsFolderID := ua.NewNumericNodeID(ns.ID(), id.ObjectsFolder)

	// Create nodes for each tag definition with explicit numeric node IDs
	nodes := make(map[string]*server.Node)
	for _, td := range machine.AllTagDefinitions(m) {
		nodeID := ua.NewNumericNodeID(ns.ID(), uint32(td.NodeID))

		attrs := server.Attributes{
			ua.AttributeIDBrowseName:              server.DataValueFromValue(&ua.QualifiedName{NamespaceIndex: ns.ID(), Name: td.Name}),
			ua.AttributeIDDisplayName:              server.DataValueFromValue(ua.NewLocalizedText(td.DisplayName)),
			ua.AttributeIDDescription:              server.DataValueFromValue(ua.NewLocalizedText(td.Name)),
			ua.AttributeIDNodeClass:                server.DataValueFromValue(int32(ua.NodeClassVariable)),
			ua.AttributeIDDataType:                 server.DataValueFromValue(dataTypeNodeID(td.DataType)),
			ua.AttributeIDValueRank:                server.DataValueFromValue(int32(-1)), // Scalar
			ua.AttributeIDAccessLevel:              server.DataValueFromValue(byte(ua.AccessLevelTypeCurrentRead | ua.AccessLevelTypeCurrentWrite)),
			ua.AttributeIDUserAccessLevel:          server.DataValueFromValue(byte(ua.AccessLevelTypeCurrentRead | ua.AccessLevelTypeCurrentWrite)),
			ua.AttributeIDMinimumSamplingInterval:  server.DataValueFromValue(float64(0)),
			ua.AttributeIDHistorizing:              server.DataValueFromValue(false),
		}

		refs := server.References{
			{
				ReferenceTypeID: ua.NewNumericNodeID(0, id.HasComponent),
				IsForward:       false,
				NodeID:          &ua.ExpandedNodeID{NodeID: objectsFolderID},
			},
			{
				ReferenceTypeID: ua.NewNumericNodeID(0, id.HasTypeDefinition),
				IsForward:       true,
				NodeID:          &ua.ExpandedNodeID{NodeID: ua.NewNumericNodeID(0, id.BaseDataVariableType)},
			},
		}

		node := server.NewNode(nodeID, attrs, refs, nil)
		ns.AddNode(node)

		nodes[td.Name] = node
	}

	ms := &machineServer{
		srv:     srv,
		machine: m,
		port:    port,
		ns:      ns,
		nodes:   nodes,
	}

	a.servers[m.ID()] = ms

	log.Info().Str("machine", m.ID()).Int("port", port).Msg("OPC-UA: registered machine")
	return nil
}

func (a *Adapter) DeregisterMachine(machineID string) error {
	a.mu.Lock()
	ms, ok := a.servers[machineID]
	if !ok {
		a.mu.Unlock()
		return fmt.Errorf("machine %q not registered", machineID)
	}
	delete(a.servers, machineID)
	a.mu.Unlock()

	if ms.cancel != nil {
		ms.cancel()
	}
	ms.srv.Close()

	log.Info().Str("machine", machineID).Msg("OPC-UA: deregistered machine")
	return nil
}

func (a *Adapter) updateLoop() {
	a.mu.RLock()
	servers := make([]*machineServer, 0, len(a.servers))
	for _, s := range a.servers {
		if s.started {
			servers = append(servers, s)
		}
	}
	a.mu.RUnlock()

	n := len(servers)
	if n == 0 {
		return
	}

	// Stagger updates: spread machines evenly across the update interval
	// so OPC-UA subscription notifications don't all fire at the same instant.
	interval := time.Duration(a.cfg.UpdateRateMS) * time.Millisecond
	stagger := interval / time.Duration(n)

	for i, ms := range servers {
		offset := stagger * time.Duration(i)
		go a.machineUpdateLoop(ms, interval, offset)
	}
}

// machineUpdateLoop runs a per-machine update ticker, offset from time zero
// so that machines publish their OPC-UA tags at staggered times.
func (a *Adapter) machineUpdateLoop(ms *machineServer, interval, offset time.Duration) {
	// Wait for the initial offset before starting the ticker
	select {
	case <-a.ctx.Done():
		return
	case <-time.After(offset):
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Do the first update immediately after the offset
	a.updateMachineTags(ms)

	for {
		select {
		case <-a.ctx.Done():
			return
		case <-ticker.C:
			a.updateMachineTags(ms)
		}
	}
}

func (a *Adapter) updateMachineTags(ms *machineServer) {
	now := time.Now()
	tags := ms.machine.GetTags()
	for name, node := range ms.nodes {
		if val, ok := tags[name]; ok {
			dv := &ua.DataValue{
				EncodingMask:    ua.DataValueValue | ua.DataValueStatusCode | ua.DataValueSourceTimestamp | ua.DataValueServerTimestamp,
				Value:           ua.MustVariant(toOPCUAType(val)),
				Status:          ua.StatusOK,
				SourceTimestamp: now,
				ServerTimestamp: now,
			}
			ms.ns.SetAttribute(node.ID(), ua.AttributeIDValue, dv)
		}
	}
}

// GetPort returns the port assigned to a machine's OPC-UA server.
func (a *Adapter) GetPort(machineID string) int {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if ms, ok := a.servers[machineID]; ok {
		return ms.port
	}
	return 0
}

// toOPCUAType converts Go values to types supported by ua.MustVariant.
// ua.MustVariant doesn't support plain int/uint - only sized types.
func toOPCUAType(val interface{}) interface{} {
	switch v := val.(type) {
	case int:
		return int64(v)
	case uint:
		return uint64(v)
	case float32:
		return float64(v)
	case fmt.Stringer:
		return v.String()
	default:
		return val
	}
}

// dataTypeNodeID maps data type strings to OPC-UA type NodeIDs.
func dataTypeNodeID(dataType string) *ua.NodeID {
	switch dataType {
	case "String", "DateTime":
		return ua.NewNumericNodeID(0, id.String)
	case "Double":
		return ua.NewNumericNodeID(0, id.Double)
	case "Float":
		return ua.NewNumericNodeID(0, id.Float)
	case "Int32":
		return ua.NewNumericNodeID(0, id.Int32)
	case "Int64":
		return ua.NewNumericNodeID(0, id.Int64)
	case "Boolean":
		return ua.NewNumericNodeID(0, id.Boolean)
	default:
		return ua.NewNumericNodeID(0, id.String)
	}
}
