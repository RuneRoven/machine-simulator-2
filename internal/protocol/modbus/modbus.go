package modbus

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	modbusLib "github.com/simonvetter/modbus"
	"github.com/rs/zerolog/log"

	"machine-simulator-2/internal/config"
	"machine-simulator-2/internal/machine"
	"machine-simulator-2/internal/protocol"
)

// RegisterEntry maps a tag name to Modbus holding registers.
type RegisterEntry struct {
	TagName     string `json:"tag_name"`
	DisplayName string `json:"display_name"`
	Address     uint16 `json:"address"`
	Count       uint16 `json:"count"`
	DataType    string `json:"data_type"`
	UnitID      uint8  `json:"unit_id"`
	MachineID   string `json:"machine_id"`
}

// Adapter exposes machines via a single Modbus TCP gateway.
type Adapter struct {
	mu sync.RWMutex

	cfg        config.ModbusAdapterConfig
	srv        *modbusLib.ModbusServer
	machines   map[string]*machineSlot
	nextUnitID uint8
	registers  map[uint8]map[uint16]float64
	registerMap []RegisterEntry
	ctx        context.Context
	cancel     context.CancelFunc
}

type machineSlot struct {
	machine machine.Machine
	unitID  uint8
	entries []RegisterEntry
}

// New creates a new Modbus TCP adapter.
func New(cfg config.ModbusAdapterConfig) *Adapter {
	return &Adapter{
		cfg:        cfg,
		machines:   make(map[string]*machineSlot),
		nextUnitID: 1,
		registers:  make(map[uint8]map[uint16]float64),
	}
}

func (a *Adapter) Name() string { return "modbus" }

func (a *Adapter) Start(ctx context.Context) error {
	a.mu.Lock()
	a.ctx, a.cancel = context.WithCancel(ctx)
	a.mu.Unlock()

	srv, err := modbusLib.NewServer(&modbusLib.ServerConfiguration{
		URL:        fmt.Sprintf("tcp://0.0.0.0:%d", a.cfg.Port),
		MaxClients: uint(a.cfg.MaxClients),
	}, a)
	if err != nil {
		return fmt.Errorf("failed to create Modbus server: %w", err)
	}

	a.mu.Lock()
	a.srv = srv
	a.mu.Unlock()

	if err := srv.Start(); err != nil {
		return fmt.Errorf("failed to start Modbus server: %w", err)
	}

	go a.updateLoop()

	log.Info().Int("port", a.cfg.Port).Msg("Modbus TCP adapter started")
	return nil
}

func (a *Adapter) Stop() error {
	a.mu.Lock()
	if a.cancel != nil {
		a.cancel()
	}
	srv := a.srv
	a.mu.Unlock()

	if srv != nil {
		srv.Stop()
	}

	log.Info().Msg("Modbus TCP adapter stopped")
	return nil
}

func (a *Adapter) RegisterMachine(m machine.Machine, cfg protocol.AdapterMachineConfig) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	unitID := uint8(cfg.UnitID)
	if unitID == 0 {
		unitID = a.nextUnitID
		a.nextUnitID++
	}

	a.registers[unitID] = make(map[uint16]float64)

	var addr uint16
	var entries []RegisterEntry

	for _, td := range machine.AllTagDefinitions(m) {
		entry := RegisterEntry{
			TagName:     td.Name,
			DisplayName: td.DisplayName,
			Address:     addr,
			Count:       2,
			DataType:    td.DataType,
			UnitID:      unitID,
			MachineID:   m.ID(),
		}
		entries = append(entries, entry)
		addr += 2
	}

	slot := &machineSlot{
		machine: m,
		unitID:  unitID,
		entries: entries,
	}

	a.machines[m.ID()] = slot
	a.registerMap = append(a.registerMap, entries...)

	log.Info().Str("machine", m.ID()).Uint8("unit_id", unitID).Int("registers", len(entries)*2).Msg("Modbus: registered machine")
	return nil
}

func (a *Adapter) DeregisterMachine(machineID string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	slot, ok := a.machines[machineID]
	if !ok {
		return fmt.Errorf("machine %q not registered", machineID)
	}

	delete(a.machines, machineID)
	delete(a.registers, slot.unitID)

	var newMap []RegisterEntry
	for _, entry := range a.registerMap {
		if entry.MachineID != machineID {
			newMap = append(newMap, entry)
		}
	}
	a.registerMap = newMap

	log.Info().Str("machine", machineID).Msg("Modbus: deregistered machine")
	return nil
}

// GetRegisterMap returns the full register map for all machines.
func (a *Adapter) GetRegisterMap() []RegisterEntry {
	a.mu.RLock()
	defer a.mu.RUnlock()
	result := make([]RegisterEntry, len(a.registerMap))
	copy(result, a.registerMap)
	return result
}

func (a *Adapter) updateLoop() {
	ticker := time.NewTicker(time.Duration(a.cfg.UpdateIntervalMS) * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-a.ctx.Done():
			return
		case <-ticker.C:
			a.updateAllRegisters()
		}
	}
}

func (a *Adapter) updateAllRegisters() {
	a.mu.Lock()
	defer a.mu.Unlock()

	for _, slot := range a.machines {
		tags := slot.machine.GetTags()
		regs := a.registers[slot.unitID]

		for _, entry := range slot.entries {
			val, ok := tags[entry.TagName]
			if !ok {
				continue
			}
			fval := toFloat64(val)
			// Store as two register values (high and low word of float32)
			f32 := float32(fval)
			bits := math.Float32bits(f32)
			regs[entry.Address] = float64(bits >> 16)
			regs[entry.Address+1] = float64(bits & 0xFFFF)
		}
	}
}

// HandleHoldingRegisters implements the modbus RequestHandler interface.
func (a *Adapter) HandleHoldingRegisters(req *modbusLib.HoldingRegistersRequest) ([]uint16, error) {
	if req.IsWrite {
		return nil, modbusLib.ErrIllegalFunction
	}

	a.mu.RLock()
	defer a.mu.RUnlock()

	regs, ok := a.registers[req.UnitId]
	if !ok {
		return nil, modbusLib.ErrIllegalDataAddress
	}

	result := make([]uint16, req.Quantity)
	for i := uint16(0); i < req.Quantity; i++ {
		addr := req.Addr + i
		if val, ok := regs[addr]; ok {
			result[i] = uint16(val)
		}
	}

	return result, nil
}

// HandleInputRegisters implements the modbus RequestHandler interface.
func (a *Adapter) HandleInputRegisters(req *modbusLib.InputRegistersRequest) ([]uint16, error) {
	return nil, modbusLib.ErrIllegalFunction
}

// HandleCoils implements the modbus RequestHandler interface.
func (a *Adapter) HandleCoils(req *modbusLib.CoilsRequest) ([]bool, error) {
	return nil, modbusLib.ErrIllegalFunction
}

// HandleDiscreteInputs implements the modbus RequestHandler interface.
func (a *Adapter) HandleDiscreteInputs(req *modbusLib.DiscreteInputsRequest) ([]bool, error) {
	return nil, modbusLib.ErrIllegalFunction
}

func toFloat64(v interface{}) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case float32:
		return float64(val)
	case int:
		return float64(val)
	case int64:
		return float64(val)
	case int32:
		return float64(val)
	case bool:
		if val {
			return 1.0
		}
		return 0.0
	default:
		return 0.0
	}
}
