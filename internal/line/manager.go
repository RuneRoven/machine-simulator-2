package line

import (
	"sort"
	"sync"

	"machine-simulator-2/internal/event"
	"machine-simulator-2/internal/machine"
)

// Manager coordinates all active production lines.
type Manager struct {
	mu sync.RWMutex

	lines         map[string]*Line           // keyed by instance ID
	flows         map[string]*flow           // keyed by instance ID
	machineToLine map[string]string          // machine ID -> line instance ID
	machines      map[string]machine.Machine // machine ID -> machine
	bus           *event.Bus
	recipes       RecipeSource
}

// NewManager creates a new line manager.
func NewManager(bus *event.Bus) *Manager {
	return &Manager{
		lines:         make(map[string]*Line),
		flows:         make(map[string]*flow),
		machineToLine: make(map[string]string),
		machines:      make(map[string]machine.Machine),
		bus:           bus,
	}
}

// AddLine registers a line and its machine references.
func (mgr *Manager) AddLine(l *Line, machines map[string]machine.Machine) {
	mgr.mu.Lock()
	defer mgr.mu.Unlock()

	mgr.lines[l.InstanceID] = l
	mgr.flows[l.InstanceID] = newFlow(l, machines)
	for _, id := range l.MachineIDs {
		mgr.machineToLine[id] = l.InstanceID
		if m, ok := machines[id]; ok {
			mgr.machines[id] = m
		}
	}
}

// GetLine returns a line by instance ID.
func (mgr *Manager) GetLine(instanceID string) *Line {
	mgr.mu.RLock()
	defer mgr.mu.RUnlock()
	return mgr.lines[instanceID]
}

// GetAllLines returns all lines, sorted by instance ID.
func (mgr *Manager) GetAllLines() []*Line {
	mgr.mu.RLock()
	defer mgr.mu.RUnlock()
	result := make([]*Line, 0, len(mgr.lines))
	for _, l := range mgr.lines {
		result = append(result, l)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].InstanceID < result[j].InstanceID
	})
	return result
}

// GetLineForMachine returns the line instance ID for a machine.
func (mgr *Manager) GetLineForMachine(machineID string) string {
	mgr.mu.RLock()
	defer mgr.mu.RUnlock()
	return mgr.machineToLine[machineID]
}

// GetBuffer returns a buffer by ID across all lines.
func (mgr *Manager) GetBuffer(bufferID string) *Buffer {
	mgr.mu.RLock()
	defer mgr.mu.RUnlock()
	for _, l := range mgr.lines {
		if b, ok := l.Buffers[bufferID]; ok {
			return b
		}
	}
	return nil
}

// GetAllBuffers returns all buffers across all lines, sorted by ID.
func (mgr *Manager) GetAllBuffers() []*Buffer {
	mgr.mu.RLock()
	defer mgr.mu.RUnlock()
	var result []*Buffer
	for _, l := range mgr.lines {
		for _, b := range l.Buffers {
			result = append(result, b)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].ID < result[j].ID
	})
	return result
}

func (mgr *Manager) getMachine(id string) machine.Machine {
	mgr.mu.RLock()
	defer mgr.mu.RUnlock()
	return mgr.machines[id]
}
