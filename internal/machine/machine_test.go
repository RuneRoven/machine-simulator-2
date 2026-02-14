package machine

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"machine-simulator-2/internal/config"
)

func testConfig() config.MachineInstanceConfig {
	return config.MachineInstanceConfig{
		ID:                   "test-machine-1",
		Type:                 "test_type",
		Role:                 "Test Role",
		LineInstanceID:       "test-line-1",
		Stage:                1,
		CycleTimeMS:          100,
		ScrapRate:            0.0,
		BreakdownProbability: 0.0,
		AutoRestoreTimeMS:    500,
	}
}

func TestStateTransitions(t *testing.T) {
	m := NewBaseMachine(testConfig())

	if m.GetState() != StateIdle {
		t.Fatalf("expected IDLE, got %s", m.GetState())
	}

	// IDLE -> RUNNING
	m.processCommand(CommandStart)
	if m.GetState() != StateRunning {
		t.Fatalf("expected RUNNING, got %s", m.GetState())
	}

	// RUNNING -> STOPPED
	m.processCommand(CommandStop)
	if m.GetState() != StateStopped {
		t.Fatalf("expected STOPPED, got %s", m.GetState())
	}

	// STOPPED -> IDLE via RESET
	m.processCommand(CommandReset)
	if m.GetState() != StateIdle {
		t.Fatalf("expected IDLE after reset, got %s", m.GetState())
	}
}

func TestInvalidTransition(t *testing.T) {
	m := NewBaseMachine(testConfig())

	// IDLE -> STOP should not work (no direct IDLE -> STOPPED)
	m.processCommand(CommandStop)
	if m.GetState() != StateIdle {
		t.Fatalf("expected IDLE (invalid transition), got %s", m.GetState())
	}
}

func TestFaultAndReset(t *testing.T) {
	m := NewBaseMachine(testConfig())
	m.processCommand(CommandStart)
	m.processCommand(CommandFault)
	if m.GetState() != StateFault {
		t.Fatalf("expected FAULT, got %s", m.GetState())
	}

	// START from FAULT should not work
	m.processCommand(CommandStart)
	if m.GetState() != StateFault {
		t.Fatalf("expected still FAULT, got %s", m.GetState())
	}

	// RESET from FAULT -> IDLE
	m.processCommand(CommandReset)
	if m.GetState() != StateIdle {
		t.Fatalf("expected IDLE, got %s", m.GetState())
	}
}

func TestCycleCompletion(t *testing.T) {
	cfg := testConfig()
	cfg.CycleTimeMS = 50 // 50ms cycle
	m := NewBaseMachine(cfg)

	var completed atomic.Int64
	m.SetOnPartComplete(func(id string, good bool) {
		completed.Add(1)
	})

	ctx, cancel := context.WithCancel(context.Background())
	go m.Start(ctx)

	m.SendCommand(CommandStart)
	m.AssignOrder("order-1", "op-1", "product-1")

	time.Sleep(250 * time.Millisecond)
	cancel()

	if completed.Load() == 0 {
		t.Error("expected at least one cycle completion")
	}
	data := m.GetBaseData()
	if data.GoodCount == 0 {
		t.Error("expected good_count > 0")
	}
}

func TestScrapRate(t *testing.T) {
	cfg := testConfig()
	cfg.CycleTimeMS = 10
	cfg.ScrapRate = 1.0 // 100% scrap
	m := NewBaseMachine(cfg)

	var goodCount, scrapCount atomic.Int64
	m.SetOnPartComplete(func(id string, good bool) {
		if good {
			goodCount.Add(1)
		} else {
			scrapCount.Add(1)
		}
	})

	ctx, cancel := context.WithCancel(context.Background())
	go m.Start(ctx)
	m.SendCommand(CommandStart)
	m.AssignOrder("o1", "op1", "p1")
	time.Sleep(200 * time.Millisecond)
	cancel()

	if goodCount.Load() != 0 {
		t.Errorf("expected 0 good parts with 100%% scrap, got %d", goodCount.Load())
	}
	if scrapCount.Load() == 0 {
		t.Error("expected scrap parts with 100% scrap rate")
	}
}

func TestRecipeOverrideApplyRevert(t *testing.T) {
	m := NewBaseMachine(testConfig())

	originalSettings := m.GetSettings()

	cycleTime := 200
	scrapRate := 0.5
	m.ApplyRecipeOverrides(RecipeOverrides{
		CycleTimeMS: &cycleTime,
		ScrapRate:   &scrapRate,
	})

	settings := m.GetSettings()
	if settings.CycleTimeMS != 200 {
		t.Errorf("expected cycle_time_ms 200, got %d", settings.CycleTimeMS)
	}
	if settings.ScrapRate != 0.5 {
		t.Errorf("expected scrap_rate 0.5, got %f", settings.ScrapRate)
	}

	// ClearOrder should revert
	m.ClearOrder()
	reverted := m.GetSettings()
	if reverted.CycleTimeMS != originalSettings.CycleTimeMS {
		t.Errorf("expected reverted cycle_time_ms %d, got %d", originalSettings.CycleTimeMS, reverted.CycleTimeMS)
	}
	if reverted.ScrapRate != originalSettings.ScrapRate {
		t.Errorf("expected reverted scrap_rate %f, got %f", originalSettings.ScrapRate, reverted.ScrapRate)
	}
}

func TestBlockedByBuffer(t *testing.T) {
	cfg := testConfig()
	cfg.CycleTimeMS = 50
	m := NewBaseMachine(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	go m.Start(ctx)
	m.SendCommand(CommandStart)
	m.AssignOrder("o1", "op1", "p1")

	// Let some cycles complete first
	time.Sleep(300 * time.Millisecond)
	preBlockCount := m.GetBaseData().CycleCount

	// Block the machine
	m.SetBlockedByBuffer(true)
	if !m.IsBlockedByBuffer() {
		t.Error("expected blocked")
	}

	time.Sleep(300 * time.Millisecond)
	blockedCount := m.GetBaseData().CycleCount

	if blockedCount != preBlockCount {
		t.Errorf("expected no cycles while blocked, got delta=%d", blockedCount-preBlockCount)
	}

	// Unblock
	m.SetBlockedByBuffer(false)
	time.Sleep(300 * time.Millisecond)
	cancel()

	finalCount := m.GetBaseData().CycleCount
	if finalCount <= blockedCount {
		t.Error("expected cycles after unblocking")
	}
}

func TestRegistryRegisterAndCreate(t *testing.T) {
	// Register a test type
	Register("test_machine_type", func(cfg config.MachineInstanceConfig) Machine {
		return NewBaseMachine(cfg)
	})

	if !IsRegistered("test_machine_type") {
		t.Error("expected test_machine_type to be registered")
	}

	m, err := Create("test_machine_type", testConfig())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Type() != "test_type" { // from config, not registry key
		t.Errorf("unexpected type: %s", m.Type())
	}

	// Unknown type
	_, err = Create("nonexistent_type", testConfig())
	if err == nil {
		t.Error("expected error for unknown type")
	}
}

func TestListTypes(t *testing.T) {
	types := ListTypes()
	if len(types) == 0 {
		t.Error("expected at least one registered type")
	}
}
