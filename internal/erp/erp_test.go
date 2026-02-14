package erp

import (
	"testing"
	"time"

	"machine-simulator-2/internal/event"
	"machine-simulator-2/internal/storage"
)

func setupTest(t *testing.T) (*ERP, *storage.OrderRepository, *event.Bus) {
	t.Helper()
	db, err := storage.NewDB(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	orders := storage.NewOrderRepository(db)
	bus := event.New()

	cfg := Config{
		Mode:                "auto",
		AutoOrderIntervalMS: 100,
		MaxBacklog:          3,
		TimeScale:           1.0,
	}

	e := New(cfg, orders, bus)
	e.RegisterProduct(ProductRegistration{
		LineInstanceID: "line-1",
		LineTemplate:   "test-line",
		ProductID:      "PROD-A",
		PlannedQtyMin:  10,
		PlannedQtyMax:  50,
	})

	return e, orders, bus
}

func TestCreateManualOrder(t *testing.T) {
	e, orders, _ := setupTest(t)

	order, err := e.CreateManualOrder("line-1", "test-line", "PROD-A", 100, 75)
	if err != nil {
		t.Fatal(err)
	}
	if order.PlannedQty != 100 {
		t.Errorf("expected qty 100, got %d", order.PlannedQty)
	}

	got, err := orders.GetByID(order.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "CREATED" {
		t.Errorf("expected CREATED, got %s", got.Status)
	}
}

func TestOrderLifecycle(t *testing.T) {
	e, orders, _ := setupTest(t)

	order, _ := e.CreateManualOrder("line-1", "test-line", "PROD-A", 50, 50)

	// Release
	if err := e.ReleaseOrder(order.ID); err != nil {
		t.Fatal(err)
	}
	got, _ := orders.GetByID(order.ID)
	if got.Status != "RELEASED" {
		t.Errorf("expected RELEASED, got %s", got.Status)
	}

	// Close
	if err := e.CloseOrder(order.ID); err != nil {
		t.Fatal(err)
	}
	got, _ = orders.GetByID(order.ID)
	if got.Status != "CLOSED" {
		t.Errorf("expected CLOSED, got %s", got.Status)
	}
}

func TestCancelOrder(t *testing.T) {
	e, orders, _ := setupTest(t)

	order, _ := e.CreateManualOrder("line-1", "test-line", "PROD-A", 50, 50)
	if err := e.CancelOrder(order.ID); err != nil {
		t.Fatal(err)
	}
	got, _ := orders.GetByID(order.ID)
	if got.Status != "CANCELLED" {
		t.Errorf("expected CANCELLED, got %s", got.Status)
	}
}

func TestBacklogControl(t *testing.T) {
	e, orders, bus := setupTest(t)

	var createdCount int
	bus.Subscribe(event.OrderCreated, func(ev event.Event) {
		createdCount++
	})

	// Fill backlog to max (3)
	for i := 0; i < 3; i++ {
		e.CreateManualOrder("line-1", "test-line", "PROD-A", 10, 50)
	}

	// Auto-generate should be skipped due to backlog
	e.generateAutoOrder()

	list, _ := orders.List(storage.OrderFilter{LineInstanceID: "line-1"})
	if len(list) != 3 {
		t.Errorf("expected 3 orders (backlog full), got %d", len(list))
	}
}

func TestEventPublished(t *testing.T) {
	e, _, bus := setupTest(t)

	received := make(chan event.Event, 5)
	bus.Subscribe(event.OrderCreated, func(ev event.Event) {
		received <- ev
	})

	e.CreateManualOrder("line-1", "test-line", "PROD-A", 50, 50)

	select {
	case ev := <-received:
		if ev.LineID != "line-1" {
			t.Errorf("expected line-1, got %s", ev.LineID)
		}
	case <-time.After(time.Second):
		t.Error("timeout waiting for event")
	}
}
