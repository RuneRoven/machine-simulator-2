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

	lines          map[string]*Line            // keyed by instance ID
	machineToLine  map[string]string           // machine ID -> line instance ID
	machines       map[string]machine.Machine  // machine ID -> machine
	bus            *event.Bus
}

// NewManager creates a new line manager.
func NewManager(bus *event.Bus) *Manager {
	return &Manager{
		lines:         make(map[string]*Line),
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

// HandlePartComplete is called when a machine completes a part.
// It routes the good part to the downstream buffer.
// Returns true if the part was added to a buffer (or is the last stage).
func (mgr *Manager) HandlePartComplete(machineID string, good bool) bool {
	mgr.mu.RLock()
	lineID, ok := mgr.machineToLine[machineID]
	if !ok {
		mgr.mu.RUnlock()
		return false
	}
	l := mgr.lines[lineID]
	mgr.mu.RUnlock()

	if !good {
		return true // scrap — don't route
	}

	// If last machine, nothing to route
	if l.IsLastMachine(machineID) {
		return true
	}

	// Add to downstream buffer
	downstream := l.GetDownstreamBuffer(machineID)
	if downstream == nil {
		return false
	}

	orderID, opID := "", ""
	if m := mgr.getMachine(machineID); m != nil {
		orderID, opID = m.GetCurrentOrder()
	}

	part := NewPart(orderID, opID)
	part.IsGood = true

	added := downstream.Add(part)
	if added && mgr.bus != nil {
		mgr.bus.Publish(event.Event{
			Type:   event.LinePartMoved,
			LineID: lineID,
			Data: map[string]interface{}{
				"from_machine": machineID,
				"buffer_id":    downstream.ID,
				"buffer_count": downstream.Count(),
			},
		})

		if downstream.IsFull() {
			mgr.bus.Publish(event.Event{
				Type:   event.BufferFull,
				LineID: lineID,
				Data:   downstream.ID,
			})
		}
	}

	// Block upstream if buffer full
	if downstream.IsFull() {
		if m := mgr.getMachine(machineID); m != nil {
			m.SetBlockedByBuffer(true)
		}
	}

	return added
}

// TryPullFromBuffer attempts to pull a part from the upstream buffer of a machine.
func (mgr *Manager) TryPullFromBuffer(machineID string) *Part {
	mgr.mu.RLock()
	lineID := mgr.machineToLine[machineID]
	l := mgr.lines[lineID]
	mgr.mu.RUnlock()

	if l == nil || l.IsFirstMachine(machineID) {
		return nil
	}

	upstream := l.GetUpstreamBuffer(machineID)
	if upstream == nil {
		return nil
	}

	part := upstream.Remove()
	if part != nil {
		// Unblock the upstream machine if it was blocked
		if !upstream.IsFull() {
			if upM := mgr.getMachine(upstream.UpstreamMachineID); upM != nil {
				upM.SetBlockedByBuffer(false)
			}
		}

		if upstream.IsEmpty() && mgr.bus != nil {
			mgr.bus.Publish(event.Event{
				Type:   event.BufferEmpty,
				LineID: lineID,
				Data:   upstream.ID,
			})
		}
	}

	return part
}

// IsDownstreamBlocked returns true if the downstream buffer of a machine is full.
func (mgr *Manager) IsDownstreamBlocked(machineID string) bool {
	mgr.mu.RLock()
	lineID := mgr.machineToLine[machineID]
	l := mgr.lines[lineID]
	mgr.mu.RUnlock()

	if l == nil || l.IsLastMachine(machineID) {
		return false
	}

	downstream := l.GetDownstreamBuffer(machineID)
	return downstream != nil && downstream.IsFull()
}

// DetectAndRecoverDeadlock checks a line for deadlock and recovers if found.
// Returns true if a deadlock was recovered.
func (mgr *Manager) DetectAndRecoverDeadlock(lineID string) bool {
	mgr.mu.RLock()
	l := mgr.lines[lineID]
	mgr.mu.RUnlock()
	if l == nil {
		return false
	}

	// Detect two deadlock patterns:
	// 1. Classic: all machines stopped + parts stuck in buffers
	// 2. Buffer flag: machines are RUNNING/IDLE but all blocked/starved with no progress
	allStopped := true
	allStuck := true // true if every machine is either blocked, starved, or idle
	for _, id := range l.MachineIDs {
		m := mgr.getMachine(id)
		if m == nil {
			continue
		}
		state := m.GetState()

		if state != machine.StateStopped {
			allStopped = false
		}

		// A machine is "stuck" if it's idle, or running but blocked/starved
		if state == machine.StateRunning && !m.IsBlockedByBuffer() && !m.IsStarvedByBuffer() {
			allStuck = false
		}
	}

	hasPartsInBuffers := false
	for _, b := range l.Buffers {
		if !b.IsEmpty() {
			hasPartsInBuffers = true
			break
		}
	}

	// Pattern 1: all stopped + parts in buffers
	// Pattern 2: all stuck (blocked/starved/idle) — progress is impossible
	isDeadlocked := (allStopped && hasPartsInBuffers) || allStuck
	if !isDeadlocked {
		return false
	}

	// Deadlock detected — recover
	for _, b := range l.Buffers {
		b.Clear()
	}
	for _, id := range l.MachineIDs {
		if m := mgr.getMachine(id); m != nil {
			m.SetBlockedByBuffer(false)
			m.SetStarvedByBuffer(false)
			if m.GetState() == machine.StateStopped {
				m.SendCommand(machine.CommandReset)
			}
		}
	}

	if mgr.bus != nil {
		mgr.bus.Publish(event.Event{
			Type:   event.LineDeadlockRecovered,
			LineID: lineID,
		})
	}

	return true
}

func (mgr *Manager) getMachine(id string) machine.Machine {
	mgr.mu.RLock()
	defer mgr.mu.RUnlock()
	return mgr.machines[id]
}
