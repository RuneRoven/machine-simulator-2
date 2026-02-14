package mes

import (
	"testing"

	"machine-simulator-2/internal/config"
	"machine-simulator-2/internal/event"
	"machine-simulator-2/internal/line"
	"machine-simulator-2/internal/machine"
	"machine-simulator-2/internal/recipe"
	"machine-simulator-2/internal/storage"
)

type mockEngine struct {
	machines map[string]machine.Machine
}

func (e *mockEngine) GetMachine(id string) machine.Machine {
	return e.machines[id]
}

func setupMESTest(t *testing.T) (*MES, *storage.OrderRepository, *storage.OperationRepository) {
	t.Helper()
	db, err := storage.NewDB(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	orders := storage.NewOrderRepository(db)
	ops := storage.NewOperationRepository(db)
	bus := event.New()
	cat := recipe.NewCatalog()

	cycleTime := 100
	cat.PopulateFromConfig(map[string]*config.LineTemplateConfig{
		"test-line": {
			Recipes: []config.RecipeConfig{
				{
					ProductID:       "PROD-A",
					PlannedQtyRange: [2]int{10, 50},
					Overrides: map[int]config.RecipeStageOverrideConfig{
						1: {CycleTimeMS: &cycleTime},
					},
				},
				{ProductID: "PROD-B", PlannedQtyRange: [2]int{5, 20}},
			},
		},
	})

	lineMgr := line.NewManager(bus)
	eng := &mockEngine{machines: make(map[string]machine.Machine)}

	// Create machines
	m1 := machine.NewBaseMachine(config.MachineInstanceConfig{
		ID: "line-1-stage-1", Type: "test", Stage: 1, CycleTimeMS: 500, LineInstanceID: "line-1",
	})
	m2 := machine.NewBaseMachine(config.MachineInstanceConfig{
		ID: "line-1-stage-2", Type: "test", Stage: 2, CycleTimeMS: 500, LineInstanceID: "line-1",
	})
	eng.machines["line-1-stage-1"] = m1
	eng.machines["line-1-stage-2"] = m2

	// Set up line with buffer so HandlePartComplete works
	liCfg := config.LineInstanceConfig{
		InstanceID:   "line-1",
		TemplateName: "test-line",
		Machines: []config.MachineInstanceConfig{
			{ID: "line-1-stage-1", Stage: 1},
			{ID: "line-1-stage-2", Stage: 2},
		},
		Buffers: []config.BufferInstanceConfig{
			{ID: "line-1-buffer-1-2", UpstreamMachineID: "line-1-stage-1", DownstreamMachineID: "line-1-stage-2", Capacity: 10},
		},
	}
	l := line.InstantiateFromConfig(liCfg)
	lineMgr.AddLine(l, map[string]machine.Machine{
		"line-1-stage-1": m1,
		"line-1-stage-2": m2,
	})

	mesCfg := Config{
		DispatchIntervalMS: 100,
		DispatchStrategy:   "round-robin",
		TimeScale:          1.0,
	}

	mes := New(mesCfg, orders, ops, bus, cat, lineMgr, eng)
	mes.RegisterLine(liCfg)

	return mes, orders, ops
}

func TestDispatchOrder(t *testing.T) {
	mes, orders, ops := setupMESTest(t)

	order := &storage.Order{
		ID: "order-1", ProductID: "PROD-A", LineInstanceID: "line-1",
		LineTemplate: "test-line", Status: "RELEASED", PlannedQty: 50,
	}
	orders.Create(order)

	mes.dispatchOrder(order)

	// Check operations were created
	opsList, err := ops.ListByOrderID("order-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(opsList) != 2 {
		t.Fatalf("expected 2 operations, got %d", len(opsList))
	}

	// Check order is now IN_PROGRESS
	got, _ := orders.GetByID("order-1")
	if got.Status != "IN_PROGRESS" {
		t.Errorf("expected IN_PROGRESS, got %s", got.Status)
	}
}

func TestCancelOrder(t *testing.T) {
	mes, orders, ops := setupMESTest(t)

	order := &storage.Order{
		ID: "order-1", ProductID: "PROD-A", LineInstanceID: "line-1",
		LineTemplate: "test-line", Status: "RELEASED", PlannedQty: 50,
	}
	orders.Create(order)
	mes.dispatchOrder(order)

	err := mes.CancelOrder("order-1")
	if err != nil {
		t.Fatal(err)
	}

	got, _ := orders.GetByID("order-1")
	if got.Status != "CANCELLED" {
		t.Errorf("expected CANCELLED, got %s", got.Status)
	}

	opsList, _ := ops.ListByOrderID("order-1")
	for _, op := range opsList {
		if op.Status != "CANCELLED" {
			t.Errorf("operation %s should be CANCELLED, got %s", op.ID, op.Status)
		}
	}
}

func TestSelectLineInstanceRoundRobin(t *testing.T) {
	mes, _, _ := setupMESTest(t)

	// Register a second instance
	mes.RegisterLine(config.LineInstanceConfig{
		InstanceID:   "line-2",
		TemplateName: "test-line",
		Machines: []config.MachineInstanceConfig{
			{ID: "line-2-stage-1", Stage: 1},
		},
	})

	first := mes.SelectLineInstance("test-line")
	second := mes.SelectLineInstance("test-line")

	if first == second {
		t.Error("round-robin should alternate between instances")
	}
}

func TestHandleCycleComplete(t *testing.T) {
	mes, orders, ops := setupMESTest(t)

	order := &storage.Order{
		ID: "order-1", ProductID: "PROD-A", LineInstanceID: "line-1",
		LineTemplate: "test-line", Status: "RELEASED", PlannedQty: 2,
	}
	orders.Create(order)
	mes.dispatchOrder(order)

	// Stage-1 produces a part → goes to buffer, stage-2 gets started
	mes.handleCycleComplete("line-1-stage-1", true)

	// Stage-1 operation should have good_qty=1
	opsList, _ := ops.ListByOrderID("order-1")
	if opsList[0].GoodQty != 1 {
		t.Errorf("expected stage-1 op good_qty 1, got %d", opsList[0].GoodQty)
	}

	// Order quantities only update from last machine (stage-2), so still 0
	got, _ := orders.GetByID("order-1")
	if got.GoodQty != 0 {
		t.Errorf("expected order good_qty 0 (stage-1 doesn't update order), got %d", got.GoodQty)
	}

	// Stage-2 processes the part → order good_qty increments
	mes.handleCycleComplete("line-1-stage-2", true)

	got, _ = orders.GetByID("order-1")
	if got.GoodQty != 1 {
		t.Errorf("expected order good_qty 1 after stage-2, got %d", got.GoodQty)
	}

	// Second part flows through both stages
	mes.handleCycleComplete("line-1-stage-1", true)
	mes.handleCycleComplete("line-1-stage-2", true)

	got, _ = orders.GetByID("order-1")
	if got.Status != "CLOSED" {
		t.Errorf("expected CLOSED after meeting planned_qty, got %s", got.Status)
	}
}
