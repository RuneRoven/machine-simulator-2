package storage

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func setupTestDB(t *testing.T) (*OrderRepository, *OperationRepository) {
	t.Helper()
	db, err := NewDB(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return NewOrderRepository(db), NewOperationRepository(db)
}

func TestOrderCRUD(t *testing.T) {
	orders, _ := setupTestDB(t)

	o := &Order{
		ID:             uuid.New().String(),
		ProductID:      "PROD-A",
		LineInstanceID: "line-1",
		LineTemplate:   "test-line",
		Status:         "CREATED",
		PlannedQty:     100,
		Priority:       75,
		CreatedAt:      time.Now(),
	}

	if err := orders.Create(o); err != nil {
		t.Fatal(err)
	}

	got, err := orders.GetByID(o.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ProductID != "PROD-A" {
		t.Errorf("expected PROD-A, got %s", got.ProductID)
	}
	if got.Status != "CREATED" {
		t.Errorf("expected CREATED, got %s", got.Status)
	}
}

func TestOrderStatusUpdate(t *testing.T) {
	orders, _ := setupTestDB(t)

	o := &Order{
		ID: uuid.New().String(), ProductID: "P", LineInstanceID: "L",
		LineTemplate: "T", Status: "CREATED", PlannedQty: 50, CreatedAt: time.Now(),
	}
	orders.Create(o)

	orders.UpdateStatus(o.ID, "RELEASED")
	got, _ := orders.GetByID(o.ID)
	if got.Status != "RELEASED" {
		t.Errorf("expected RELEASED, got %s", got.Status)
	}

	orders.UpdateStatus(o.ID, "IN_PROGRESS")
	got, _ = orders.GetByID(o.ID)
	if got.StartedAt == nil {
		t.Error("expected started_at to be set")
	}

	orders.UpdateStatus(o.ID, "CLOSED")
	got, _ = orders.GetByID(o.ID)
	if got.ClosedAt == nil {
		t.Error("expected closed_at to be set")
	}
}

func TestOrderListWithFilters(t *testing.T) {
	orders, _ := setupTestDB(t)

	for i, status := range []string{"CREATED", "CREATED", "RELEASED", "CLOSED"} {
		o := &Order{
			ID: uuid.New().String(), ProductID: "P", LineInstanceID: "line-1",
			LineTemplate: "T", Status: status, PlannedQty: 50 + i, CreatedAt: time.Now(),
		}
		orders.Create(o)
	}

	// Filter by status
	list, err := orders.List(OrderFilter{Status: "CREATED"})
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Errorf("expected 2 CREATED orders, got %d", len(list))
	}

	// Limit
	list, _ = orders.List(OrderFilter{Limit: 2})
	if len(list) != 2 {
		t.Errorf("expected 2 orders with limit, got %d", len(list))
	}
}

func TestOrderQuantities(t *testing.T) {
	orders, _ := setupTestDB(t)

	o := &Order{
		ID: uuid.New().String(), ProductID: "P", LineInstanceID: "L",
		LineTemplate: "T", Status: "CREATED", PlannedQty: 100, CreatedAt: time.Now(),
	}
	orders.Create(o)

	orders.UpdateQuantities(o.ID, 45, 5)
	got, _ := orders.GetByID(o.ID)
	if got.GoodQty != 45 || got.ScrapQty != 5 {
		t.Errorf("expected 45/5, got %d/%d", got.GoodQty, got.ScrapQty)
	}
}

func TestCountByStatus(t *testing.T) {
	orders, _ := setupTestDB(t)

	for i := 0; i < 3; i++ {
		o := &Order{
			ID: uuid.New().String(), ProductID: "P", LineInstanceID: "line-1",
			LineTemplate: "T", Status: "CREATED", PlannedQty: 50, CreatedAt: time.Now(),
		}
		orders.Create(o)
	}

	count, err := orders.CountByStatus("line-1", "CREATED")
	if err != nil {
		t.Fatal(err)
	}
	if count != 3 {
		t.Errorf("expected 3, got %d", count)
	}
}

func TestOperationCRUD(t *testing.T) {
	_, ops := setupTestDB(t)

	op := &Operation{
		ID:      uuid.New().String(),
		OrderID: "order-1",
		MachineID: "machine-1",
		Stage:   1,
		Status:  "PENDING",
	}

	if err := ops.Create(op); err != nil {
		t.Fatal(err)
	}

	got, err := ops.GetByID(op.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Stage != 1 || got.Status != "PENDING" {
		t.Errorf("unexpected values: stage=%d, status=%s", got.Stage, got.Status)
	}
}

func TestOperationListByOrder(t *testing.T) {
	_, ops := setupTestDB(t)

	for _, stage := range []int{3, 1, 2} { // insert out of order
		op := &Operation{
			ID: uuid.New().String(), OrderID: "order-1",
			MachineID: "m" + string(rune('0'+stage)), Stage: stage, Status: "PENDING",
		}
		ops.Create(op)
	}

	list, err := ops.ListByOrderID("order-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 3 {
		t.Fatalf("expected 3 ops, got %d", len(list))
	}
	// Should be ordered by stage
	if list[0].Stage != 1 || list[1].Stage != 2 || list[2].Stage != 3 {
		t.Error("operations not ordered by stage")
	}
}

func TestOperationUpdateQuantities(t *testing.T) {
	_, ops := setupTestDB(t)

	op := &Operation{
		ID: uuid.New().String(), OrderID: "o1", MachineID: "m1", Stage: 1, Status: "IN_PROGRESS",
	}
	ops.Create(op)
	ops.UpdateQuantities(op.ID, 10, 2)

	got, _ := ops.GetByID(op.ID)
	if got.GoodQty != 10 || got.ScrapQty != 2 {
		t.Errorf("expected 10/2, got %d/%d", got.GoodQty, got.ScrapQty)
	}
}
