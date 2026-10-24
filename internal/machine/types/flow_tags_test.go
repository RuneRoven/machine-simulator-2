package types

import (
	"testing"

	"machine-simulator-2/internal/config"
	"machine-simulator-2/internal/machine"
)

type gauge struct{ count, capacity int }

func (g gauge) Count() int    { return g.count }
func (g gauge) Capacity() int { return g.capacity }

func TestEveryTypeExposesFlowTags(t *testing.T) {
	for _, typeName := range machine.ListTypes() {
		t.Run(typeName, func(t *testing.T) {
			m, err := machine.Create(typeName, config.MachineInstanceConfig{ID: "m1", Type: typeName, CycleTimeMS: 1000})
			if err != nil {
				t.Fatal(err)
			}
			defs := machine.AllTagDefinitions(m)
			seen := make(map[int]string)
			for _, d := range defs {
				if prev, dup := seen[d.NodeID]; dup {
					t.Fatalf("NodeID %d used by %s and %s", d.NodeID, prev, d.Name)
				}
				seen[d.NodeID] = d.Name
			}
			for id, name := range map[int]string{1: "state", 6: "blocked_by_buffer", 1001: "starved", 1005: "out_buffer_capacity"} {
				if seen[id] != name {
					t.Errorf("NodeID %d = %q, want %q", id, seen[id], name)
				}
			}
			// Flow tags come last so Modbus registers of existing tags keep their address.
			if last := defs[len(defs)-1]; last.NodeID != 1005 {
				t.Errorf("last definition = %s (%d), want out_buffer_capacity", last.Name, last.NodeID)
			}

			fb, ok := m.(interface {
				SetFlowBuffers(in, out machine.BufferGauge)
			})
			if !ok {
				t.Fatal("machine does not accept flow buffers")
			}
			fb.SetFlowBuffers(gauge{count: 2, capacity: 5}, nil)
			tags := m.GetTags()
			for _, d := range defs {
				if _, ok := tags[d.Name]; !ok {
					t.Errorf("tag %s defined but has no value", d.Name)
				}
			}
			if tags["in_buffer_level"] != int32(2) || tags["in_buffer_capacity"] != int32(5) || tags["out_buffer_capacity"] != int32(0) {
				t.Errorf("buffer tags = %v/%v/%v, want 2/5/0", tags["in_buffer_level"], tags["in_buffer_capacity"], tags["out_buffer_capacity"])
			}
		})
	}
}
