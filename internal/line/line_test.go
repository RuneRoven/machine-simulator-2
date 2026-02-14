package line

import (
	"testing"

	"machine-simulator-2/internal/config"
)

func TestBufferFIFO(t *testing.T) {
	b := NewBuffer("buf1", "line1", "m1", "m2", 3)

	p1 := NewPart("o1", "op1")
	p2 := NewPart("o2", "op2")
	p3 := NewPart("o3", "op3")

	if !b.Add(p1) || !b.Add(p2) || !b.Add(p3) {
		t.Fatal("should be able to add 3 parts")
	}

	if !b.IsFull() {
		t.Error("buffer should be full")
	}

	p4 := NewPart("o4", "op4")
	if b.Add(p4) {
		t.Error("should not be able to add to full buffer")
	}

	got := b.Remove()
	if got.OrderID != "o1" {
		t.Errorf("expected FIFO order, got %s", got.OrderID)
	}

	got = b.Remove()
	if got.OrderID != "o2" {
		t.Errorf("expected FIFO order, got %s", got.OrderID)
	}

	if b.Count() != 1 {
		t.Errorf("expected count 1, got %d", b.Count())
	}
}

func TestBufferEmpty(t *testing.T) {
	b := NewBuffer("buf1", "line1", "m1", "m2", 5)

	if !b.IsEmpty() {
		t.Error("new buffer should be empty")
	}

	got := b.Remove()
	if got != nil {
		t.Error("remove from empty buffer should return nil")
	}
}

func TestBufferClear(t *testing.T) {
	b := NewBuffer("buf1", "line1", "m1", "m2", 5)
	b.Add(NewPart("o1", "op1"))
	b.Add(NewPart("o2", "op2"))

	cleared := b.Clear()
	if cleared != 2 {
		t.Errorf("expected 2 cleared, got %d", cleared)
	}
	if !b.IsEmpty() {
		t.Error("buffer should be empty after clear")
	}
}

func TestBufferPeek(t *testing.T) {
	b := NewBuffer("buf1", "line1", "m1", "m2", 5)
	b.Add(NewPart("o1", "op1"))

	p := b.Peek()
	if p == nil || p.OrderID != "o1" {
		t.Error("peek should return front part")
	}
	if b.Count() != 1 {
		t.Error("peek should not remove the part")
	}
}

func TestBufferThroughput(t *testing.T) {
	b := NewBuffer("buf1", "line1", "m1", "m2", 5)
	b.Add(NewPart("o1", "op1"))
	b.Add(NewPart("o2", "op2"))
	b.Remove()
	b.Remove()

	if b.TotalThroughput() != 2 {
		t.Errorf("expected throughput 2, got %d", b.TotalThroughput())
	}
}

func TestLineInstantiation(t *testing.T) {
	liCfg := config.LineInstanceConfig{
		InstanceID:   "test-line-1",
		TemplateName: "test-line",
		Domain:       "test",
		Machines: []config.MachineInstanceConfig{
			{ID: "test-line-1-stage-1", Type: "type_a", Stage: 1},
			{ID: "test-line-1-stage-2", Type: "type_b", Stage: 2},
			{ID: "test-line-1-stage-3", Type: "type_c", Stage: 3},
		},
		Buffers: []config.BufferInstanceConfig{
			{ID: "buf-1-2", UpstreamMachineID: "test-line-1-stage-1", DownstreamMachineID: "test-line-1-stage-2", Capacity: 10},
			{ID: "buf-2-3", UpstreamMachineID: "test-line-1-stage-2", DownstreamMachineID: "test-line-1-stage-3", Capacity: 15},
		},
		Recipes: []config.RecipeConfig{
			{ProductID: "P1", Description: "Product 1"},
			{ProductID: "P2", Description: "Product 2"},
		},
	}

	l := InstantiateFromConfig(liCfg)

	if l.InstanceID != "test-line-1" {
		t.Errorf("expected instance ID test-line-1, got %s", l.InstanceID)
	}
	if len(l.MachineIDs) != 3 {
		t.Errorf("expected 3 machines, got %d", len(l.MachineIDs))
	}
	if len(l.Buffers) != 2 {
		t.Errorf("expected 2 buffers, got %d", len(l.Buffers))
	}
	if len(l.Recipes) != 2 {
		t.Errorf("expected 2 recipes, got %d", len(l.Recipes))
	}

	// Test buffer lookup
	buf := l.GetDownstreamBuffer("test-line-1-stage-1")
	if buf == nil {
		t.Fatal("expected downstream buffer for stage 1")
	}
	if buf.MaxCapacity != 10 {
		t.Errorf("expected capacity 10, got %d", buf.MaxCapacity)
	}

	upBuf := l.GetUpstreamBuffer("test-line-1-stage-2")
	if upBuf == nil {
		t.Fatal("expected upstream buffer for stage 2")
	}

	if !l.IsFirstMachine("test-line-1-stage-1") {
		t.Error("stage 1 should be first")
	}
	if !l.IsLastMachine("test-line-1-stage-3") {
		t.Error("stage 3 should be last")
	}
}
