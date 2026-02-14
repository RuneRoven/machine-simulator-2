package opcua

import (
	"context"
	"fmt"
	"sync"
	"time"

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
		ms.started = true
		go func(s *machineServer, machineID string, c context.Context) {
			if err := s.srv.Start(c); err != nil && c.Err() == nil {
				log.Error().Err(err).Str("machine", machineID).Msg("OPC-UA server error")
			}
		}(ms, id, msCtx)
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
	srv.AddNamespace(ns)

	// Create nodes for each tag definition with explicit numeric node IDs
	nodes := make(map[string]*server.Node)
	for _, td := range m.GetTagDefinitions() {
		var initVal interface{}
		switch td.DataType {
		case "Int32":
			initVal = int32(0)
		case "Int64":
			initVal = int64(0)
		case "Float":
			initVal = float32(0)
		case "Double":
			initVal = float64(0)
		case "Boolean":
			initVal = false
		case "String":
			initVal = ""
		default:
			initVal = float64(0)
		}

		nodeID := ua.NewNumericNodeID(ns.ID(), uint32(td.NodeID))
		node := server.NewVariableNode(nodeID, td.Name, initVal)
		ns.AddNode(node)

		// Set display name
		node.SetDisplayName(td.DisplayName, "")

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
	ticker := time.NewTicker(time.Duration(a.cfg.UpdateRateMS) * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-a.ctx.Done():
			return
		case <-ticker.C:
			a.updateAllTags()
		}
	}
}

func (a *Adapter) updateAllTags() {
	a.mu.RLock()
	servers := make([]*machineServer, 0, len(a.servers))
	for _, s := range a.servers {
		servers = append(servers, s)
	}
	a.mu.RUnlock()

	for _, ms := range servers {
		tags := ms.machine.GetTags()
		for name, node := range ms.nodes {
			if val, ok := tags[name]; ok {
				dv := server.DataValueFromValue(val)
				node.SetAttribute(ua.AttributeIDValue, dv)
			}
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
