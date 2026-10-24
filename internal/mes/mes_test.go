package mes

import (
	"context"
	"sync"
	"testing"
	"time"

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
	m, orders, ops, _, _ := setupMESLine(t)
	return m, orders, ops
}

func setupMESLine(t *testing.T) (*MES, *storage.OrderRepository, *storage.OperationRepository, *line.Manager, *event.Bus) {
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

	liCfg := config.LineInstanceConfig{
		InstanceID:   "line-1",
		TemplateName: "test-line",
		Machines: []config.MachineInstanceConfig{
			{ID: "line-1-stage-1", Stage: 1, CycleTimeMS: 500},
			{ID: "line-1-stage-2", Stage: 2, CycleTimeMS: 500},
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

	lineMgr.SubscribeCycleEvents()
	mes := New(mesCfg, orders, ops, bus, cat, lineMgr, eng)
	mes.RegisterLine(liCfg)

	return mes, orders, ops, lineMgr, bus
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

func newOrder(t *testing.T, orders *storage.OrderRepository, id string, qty, priority int, created time.Time) *storage.Order {
	t.Helper()
	o := &storage.Order{
		ID: id, ProductID: "PROD-A", LineInstanceID: "line-1", LineTemplate: "test-line",
		Status: "RELEASED", PlannedQty: qty, Priority: priority, CreatedAt: created,
	}
	if err := orders.Create(o); err != nil {
		t.Fatal(err)
	}
	return o
}

func status(t *testing.T, orders *storage.OrderRepository, id string) string {
	t.Helper()
	o, err := orders.GetByID(id)
	if err != nil {
		t.Fatal(err)
	}
	return o.Status
}

// cycle publishes a machine cycle the way the engine does.
func cycle(bus *event.Bus, machineID string, good bool) {
	bus.Publish(event.Event{Type: event.MachineCycleComplete, MachineID: machineID, Data: good})
}

func TestHandleCycleComplete(t *testing.T) {
	mes, orders, ops, _, bus := setupMESLine(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	mes.Start(ctx)

	order := newOrder(t, orders, "order-1", 2, 0, time.Now())
	mes.tryDispatch(order.LineInstanceID)

	cycle(bus, "line-1-stage-1", true)
	opsList, _ := ops.ListByOrderID("order-1")
	if opsList[0].GoodQty != 1 {
		t.Errorf("stage-1 op good_qty = %d, want 1", opsList[0].GoodQty)
	}
	if got, _ := orders.GetByID("order-1"); got.GoodQty != 0 {
		t.Errorf("order good_qty = %d after stage 1, want 0", got.GoodQty)
	}

	cycle(bus, "line-1-stage-2", true)
	if got, _ := orders.GetByID("order-1"); got.GoodQty != 1 {
		t.Errorf("order good_qty = %d after stage 2, want 1", got.GoodQty)
	}

	cycle(bus, "line-1-stage-1", true)
	cycle(bus, "line-1-stage-2", true)
	if got := status(t, orders, "order-1"); got != "CLOSED" {
		t.Errorf("status = %s after planned_qty reached, want CLOSED", got)
	}
}

func TestBusyLineKeepsOrderQueued(t *testing.T) {
	mes, orders, _, lineMgr, bus := setupMESLine(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	mes.Start(ctx)

	now := time.Now()
	newOrder(t, orders, "first", 1, 0, now)
	newOrder(t, orders, "second", 1, 0, now.Add(time.Second))
	mes.tryDispatch("line-1")
	mes.tryDispatch("line-1")

	if got := status(t, orders, "first"); got != "IN_PROGRESS" {
		t.Fatalf("first = %s, want IN_PROGRESS", got)
	}
	if got := status(t, orders, "second"); got != "RELEASED" {
		t.Fatalf("second = %s while line busy, want RELEASED", got)
	}

	cycle(bus, "line-1-stage-1", true)
	cycle(bus, "line-1-stage-2", true)
	if got := status(t, orders, "first"); got != "CLOSED" {
		t.Fatalf("first = %s, want CLOSED", got)
	}
	if got := status(t, orders, "second"); got != "IN_PROGRESS" {
		t.Fatalf("second = %s after first closed, want IN_PROGRESS", got)
	}
	if js, _ := lineMgr.Job("line-1"); js.OrderID != "second" {
		t.Fatalf("line job order = %q, want second", js.OrderID)
	}
}

func TestQueueOrderPriorityThenAge(t *testing.T) {
	mes, orders, _, lineMgr, _ := setupMESLine(t)
	now := time.Now()
	newOrder(t, orders, "old-low", 5, 0, now)
	newOrder(t, orders, "new-high", 5, 9, now.Add(time.Minute))
	newOrder(t, orders, "older-high", 5, 9, now.Add(time.Second))
	mes.tryDispatch("line-1")
	if js, _ := lineMgr.Job("line-1"); js.OrderID != "older-high" {
		t.Fatalf("dispatched %q, want older-high", js.OrderID)
	}
}

func TestNoDoubleDispatch(t *testing.T) {
	mes, orders, ops, _, _ := setupMESLine(t)
	newOrder(t, orders, "order-1", 5, 0, time.Now())

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			mes.tryDispatch("line-1")
		}()
	}
	wg.Wait()

	opsList, _ := ops.ListByOrderID("order-1")
	if len(opsList) != 2 {
		t.Fatalf("operations = %d, want 2 (one dispatch)", len(opsList))
	}
}

func TestRestoreKeepsNewestInProgress(t *testing.T) {
	mes, orders, _, lineMgr, _ := setupMESLine(t)
	now := time.Now()
	for i, id := range []string{"a", "b", "c"} {
		newOrder(t, orders, id, 5, 0, now.Add(time.Duration(i)*time.Second))
		orders.UpdateStatus(id, "IN_PROGRESS")
		time.Sleep(5 * time.Millisecond) // distinct started_at
	}

	mes.restoreInProgressOrders()

	want := map[string]string{"a": "RELEASED", "b": "RELEASED", "c": "IN_PROGRESS"}
	for id, w := range want {
		if got := status(t, orders, id); got != w {
			t.Errorf("%s = %s, want %s", id, got, w)
		}
	}
	if js, _ := lineMgr.Job("line-1"); js.OrderID != "c" || js.State != line.JobRunning {
		t.Fatalf("line job = %+v, want c RUNNING", js)
	}
}

func TestCancelRunningOrderScrapsWIP(t *testing.T) {
	mes, orders, _, lineMgr, bus := setupMESLine(t)
	newOrder(t, orders, "order-1", 10, 0, time.Now())
	mes.tryDispatch("line-1")
	cycle(bus, "line-1-stage-1", true)
	cycle(bus, "line-1-stage-1", true)

	if err := mes.CancelOrder("order-1"); err != nil {
		t.Fatal(err)
	}
	js, _ := lineMgr.Job("line-1")
	led, _ := lineMgr.Ledger("line-1")
	if js.State != line.JobNone || led.InBuffers+led.InMachines != 0 || led.WIPScrapped == 0 || !led.Balanced() {
		t.Fatalf("job %+v ledger %+v, want no job, empty line, WIP scrapped", js, led)
	}
}
