package simulation

import (
	"context"
	"testing"
	"time"

	"machine-simulator-2/internal/config"
	"machine-simulator-2/internal/event"
	"machine-simulator-2/internal/machine"
)

func testMachineConfig(id string) config.MachineInstanceConfig {
	return config.MachineInstanceConfig{
		ID:                   id,
		Type:                 "test",
		Role:                 "Test",
		LineInstanceID:       "line-1",
		Stage:                1,
		CycleTimeMS:          50,
		ScrapRate:            0.0,
		BreakdownProbability: 0.0,
		AutoRestoreTimeMS:    500,
	}
}

func TestEngineRegisterAndStart(t *testing.T) {
	bus := event.New()
	eng := New(bus, 1.0, 0.1)

	m := machine.NewBaseMachine(testMachineConfig("m1"))
	eng.RegisterMachine(m)

	if eng.MachineCount() != 1 {
		t.Fatalf("expected 1 machine, got %d", eng.MachineCount())
	}

	ctx, cancel := context.WithCancel(context.Background())
	eng.Start(ctx)

	eng.SendCommand("m1", machine.CommandStart)
	time.Sleep(200 * time.Millisecond)

	if m.GetState() != machine.StateRunning {
		t.Errorf("expected RUNNING, got %s", m.GetState())
	}

	cancel()
	eng.Stop()
}

func TestEngineSetTimeScale(t *testing.T) {
	bus := event.New()
	eng := New(bus, 1.0, 0.1)

	eng.SetTimeScale(10.0)
	if eng.GetTimeScale() != 10.0 {
		t.Errorf("expected time scale 10.0, got %f", eng.GetTimeScale())
	}
}

func TestEngineSetRandomFactor(t *testing.T) {
	bus := event.New()
	eng := New(bus, 1.0, 0.1)

	eng.SetRandomFactor(0.25)
	if eng.GetRandomFactor() != 0.25 {
		t.Errorf("expected random factor 0.25, got %f", eng.GetRandomFactor())
	}
}

func TestEngineCycleCompleteEvent(t *testing.T) {
	bus := event.New()
	eng := New(bus, 1.0, 0.0) // zero random for predictability

	received := make(chan event.Event, 10)
	bus.Subscribe(event.MachineCycleComplete, func(e event.Event) {
		received <- e
	})

	m := machine.NewBaseMachine(testMachineConfig("m1"))
	eng.RegisterMachine(m)

	ctx, cancel := context.WithCancel(context.Background())
	eng.Start(ctx)
	eng.SendCommand("m1", machine.CommandStart)
	m.AssignOrder("o1", "op1", "p1")

	select {
	case e := <-received:
		if e.MachineID != "m1" {
			t.Errorf("expected machine ID m1, got %s", e.MachineID)
		}
	case <-time.After(500 * time.Millisecond):
		t.Error("timeout waiting for cycle complete event")
	}

	cancel()
	eng.Stop()
}

func TestEngineGetMachine(t *testing.T) {
	bus := event.New()
	eng := New(bus, 1.0, 0.1)

	m := machine.NewBaseMachine(testMachineConfig("m1"))
	eng.RegisterMachine(m)

	got := eng.GetMachine("m1")
	if got == nil {
		t.Fatal("expected machine, got nil")
	}

	got2 := eng.GetMachine("nonexistent")
	if got2 != nil {
		t.Error("expected nil for nonexistent machine")
	}
}
