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

	// deadlockStrikes tracks consecutive checks where a line had no progressing machine.
	// Recovery only triggers after deadlockThreshold consecutive strikes.
	deadlockStrikes map[string]int
}

const deadlockThreshold = 10

// NewManager creates a new line manager.
func NewManager(bus *event.Bus) *Manager {
	return &Manager{
		lines:           make(map[string]*Line),
		machineToLine:   make(map[string]string),
		machines:        make(map[string]machine.Machine),
		bus:             bus,
		deadlockStrikes: make(map[string]int),
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

	// If the downstream machine is starved, feed it immediately.
	// This prevents the race where downstream sets starvedByBuffer=true
	// on an empty buffer, then upstream fills it and gets blocked —
	// without this, nobody clears the starved flag.
	if added {
		if downM := mgr.getMachine(downstream.DownstreamMachineID); downM != nil {
			if downM.IsStarvedByBuffer() {
				pulled := downstream.Remove()
				if pulled != nil {
					downM.SetStarvedByBuffer(false)
					// Unblock upstream if buffer is no longer full
					if !downstream.IsFull() {
						if upM := mgr.getMachine(machineID); upM != nil {
							upM.SetBlockedByBuffer(false)
						}
					}
				}
			}
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
// A line is deadlocked when no machine can make progress: every machine is
// either IDLE, STOPPED, FAULT, MAINTENANCE, or RUNNING but blocked/starved.
// Returns true if a deadlock was recovered.
func (mgr *Manager) DetectAndRecoverDeadlock(lineID string) bool {
	mgr.mu.RLock()
	l := mgr.lines[lineID]
	mgr.mu.RUnlock()
	if l == nil {
		return false
	}

	// A machine is "progressing" if it's RUNNING and neither blocked nor starved.
	// If no machine is progressing, the line is deadlocked — but only if at
	// least one machine has an active order (otherwise the line is just idle
	// between orders, which is normal).
	anyProgressing := false
	anyHasOrder := false
	for _, id := range l.MachineIDs {
		m := mgr.getMachine(id)
		if m == nil {
			continue
		}
		orderID, _ := m.GetCurrentOrder()
		if orderID != "" {
			anyHasOrder = true
		}
		state := m.GetState()
		if state == machine.StateRunning && !m.IsBlockedByBuffer() && !m.IsStarvedByBuffer() {
			anyProgressing = true
			break
		}
	}

	if anyProgressing || !anyHasOrder {
		mgr.mu.Lock()
		delete(mgr.deadlockStrikes, lineID)
		mgr.mu.Unlock()
		return false
	}

	// Increment strike counter; only recover after sustained deadlock
	mgr.mu.Lock()
	mgr.deadlockStrikes[lineID]++
	strikes := mgr.deadlockStrikes[lineID]
	mgr.mu.Unlock()

	if strikes < deadlockThreshold {
		return false
	}

	// Sustained deadlock confirmed — recover by clearing all buffer state
	mgr.mu.Lock()
	delete(mgr.deadlockStrikes, lineID)
	mgr.mu.Unlock()
	for _, b := range l.Buffers {
		b.Clear()
	}
	for _, id := range l.MachineIDs {
		if m := mgr.getMachine(id); m != nil {
			m.SetBlockedByBuffer(false)
			m.SetStarvedByBuffer(false)
			state := m.GetState()
			if state == machine.StateStopped || state == machine.StateFault || state == machine.StateMaintenance {
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
