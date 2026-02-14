package line

import (
	"fmt"

	"machine-simulator-2/internal/config"
)

// InstantiateFromConfig creates a Line from a LineInstanceConfig.
func InstantiateFromConfig(li config.LineInstanceConfig) *Line {
	line := &Line{
		InstanceID:   li.InstanceID,
		TemplateName: li.TemplateName,
		Domain:       li.Domain,
		MachineIDs:   make([]string, 0, len(li.Machines)),
		Buffers:      make(map[string]*Buffer),
	}

	for _, m := range li.Machines {
		line.MachineIDs = append(line.MachineIDs, m.ID)
	}

	for _, b := range li.Buffers {
		buf := NewBuffer(b.ID, li.InstanceID, b.UpstreamMachineID, b.DownstreamMachineID, b.Capacity)
		line.Buffers[b.ID] = buf
	}

	for _, r := range li.Recipes {
		line.Recipes = append(line.Recipes, RecipeRef{
			ProductID:   r.ProductID,
			Description: r.Description,
		})
	}

	return line
}

// GetBufferBetween returns the buffer between two machine IDs, or nil.
func (l *Line) GetBufferBetween(upstreamID, downstreamID string) *Buffer {
	for _, b := range l.Buffers {
		if b.UpstreamMachineID == upstreamID && b.DownstreamMachineID == downstreamID {
			return b
		}
	}
	return nil
}

// GetDownstreamBuffer returns the buffer downstream of a machine.
func (l *Line) GetDownstreamBuffer(machineID string) *Buffer {
	for i, id := range l.MachineIDs {
		if id == machineID && i < len(l.MachineIDs)-1 {
			nextID := l.MachineIDs[i+1]
			return l.GetBufferBetween(machineID, nextID)
		}
	}
	return nil
}

// GetUpstreamBuffer returns the buffer upstream of a machine.
func (l *Line) GetUpstreamBuffer(machineID string) *Buffer {
	for i, id := range l.MachineIDs {
		if id == machineID && i > 0 {
			prevID := l.MachineIDs[i-1]
			return l.GetBufferBetween(prevID, machineID)
		}
	}
	return nil
}

// IsFirstMachine returns true if the given machine is the first in the line.
func (l *Line) IsFirstMachine(machineID string) bool {
	return len(l.MachineIDs) > 0 && l.MachineIDs[0] == machineID
}

// IsLastMachine returns true if the given machine is the last in the line.
func (l *Line) IsLastMachine(machineID string) bool {
	return len(l.MachineIDs) > 0 && l.MachineIDs[len(l.MachineIDs)-1] == machineID
}

// StageIndex returns the 0-based stage index for a machine.
func (l *Line) StageIndex(machineID string) (int, error) {
	for i, id := range l.MachineIDs {
		if id == machineID {
			return i, nil
		}
	}
	return -1, fmt.Errorf("machine %q not in line %q", machineID, l.InstanceID)
}
