package storage

import (
	"database/sql"
	"fmt"
	"time"
)

// Operation represents a work assignment at a specific machine stage.
type Operation struct {
	ID                string     `json:"id"`
	OrderID           string     `json:"order_id"`
	MachineID         string     `json:"machine_id"`
	Stage             int        `json:"stage"`
	Status            string     `json:"status"`
	PlannedCycleTimeMS int       `json:"planned_cycle_time_ms"`
	GoodQty           int        `json:"good_qty"`
	ScrapQty          int        `json:"scrap_qty"`
	StartedAt         *time.Time `json:"started_at,omitempty"`
	CompletedAt       *time.Time `json:"completed_at,omitempty"`
}

// OperationRepository provides CRUD for operations.
type OperationRepository struct {
	db *sql.DB
}

// NewOperationRepository creates a new operation repository.
func NewOperationRepository(db *sql.DB) *OperationRepository {
	return &OperationRepository{db: db}
}

// Create inserts a new operation.
func (r *OperationRepository) Create(op *Operation) error {
	_, err := r.db.Exec(
		`INSERT INTO operations (id, order_id, machine_id, stage, status, planned_cycle_time_ms, good_qty, scrap_qty, started_at, completed_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		op.ID, op.OrderID, op.MachineID, op.Stage, op.Status,
		op.PlannedCycleTimeMS, op.GoodQty, op.ScrapQty, op.StartedAt, op.CompletedAt,
	)
	return err
}

// GetByID retrieves an operation by ID.
func (r *OperationRepository) GetByID(id string) (*Operation, error) {
	op := &Operation{}
	err := r.db.QueryRow(
		`SELECT id, order_id, machine_id, stage, status, planned_cycle_time_ms, good_qty, scrap_qty, started_at, completed_at
		 FROM operations WHERE id = ?`, id,
	).Scan(&op.ID, &op.OrderID, &op.MachineID, &op.Stage, &op.Status,
		&op.PlannedCycleTimeMS, &op.GoodQty, &op.ScrapQty, &op.StartedAt, &op.CompletedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("operation %q not found", id)
	}
	return op, err
}

// ListByOrderID returns all operations for an order, ordered by stage.
func (r *OperationRepository) ListByOrderID(orderID string) ([]*Operation, error) {
	rows, err := r.db.Query(
		`SELECT id, order_id, machine_id, stage, status, planned_cycle_time_ms, good_qty, scrap_qty, started_at, completed_at
		 FROM operations WHERE order_id = ? ORDER BY stage ASC`, orderID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ops []*Operation
	for rows.Next() {
		op := &Operation{}
		if err := rows.Scan(&op.ID, &op.OrderID, &op.MachineID, &op.Stage, &op.Status,
			&op.PlannedCycleTimeMS, &op.GoodQty, &op.ScrapQty, &op.StartedAt, &op.CompletedAt); err != nil {
			return nil, err
		}
		ops = append(ops, op)
	}
	return ops, rows.Err()
}

// UpdateStatus changes an operation's status.
func (r *OperationRepository) UpdateStatus(id, status string) error {
	now := time.Now()
	var query string
	var args []interface{}

	switch status {
	case "IN_PROGRESS":
		query = `UPDATE operations SET status = ?, started_at = ? WHERE id = ?`
		args = []interface{}{status, now, id}
	case "COMPLETED":
		query = `UPDATE operations SET status = ?, completed_at = ? WHERE id = ?`
		args = []interface{}{status, now, id}
	default:
		query = `UPDATE operations SET status = ? WHERE id = ?`
		args = []interface{}{status, id}
	}

	res, err := r.db.Exec(query, args...)
	if err != nil {
		return err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("operation %q not found", id)
	}
	return nil
}

// UpdateQuantities updates good_qty and scrap_qty for an operation.
func (r *OperationRepository) UpdateQuantities(id string, goodQty, scrapQty int) error {
	_, err := r.db.Exec(
		`UPDATE operations SET good_qty = ?, scrap_qty = ? WHERE id = ?`,
		goodQty, scrapQty, id,
	)
	return err
}

// OperationFilter holds filter criteria for listing operations.
type OperationFilter struct {
	OrderID        string
	MachineID      string
	Status         string
	LineInstanceID string // filters by joining with orders table
	Limit          int
	Offset         int
}

// List returns operations matching the filter.
func (r *OperationRepository) List(f OperationFilter) ([]*Operation, error) {
	query := `SELECT o.id, o.order_id, o.machine_id, o.stage, o.status, o.planned_cycle_time_ms, o.good_qty, o.scrap_qty, o.started_at, o.completed_at FROM operations o`
	args := []interface{}{}

	if f.LineInstanceID != "" {
		query += ` JOIN orders ord ON o.order_id = ord.id`
	}

	query += ` WHERE 1=1`

	if f.OrderID != "" {
		query += " AND o.order_id = ?"
		args = append(args, f.OrderID)
	}
	if f.MachineID != "" {
		query += " AND o.machine_id = ?"
		args = append(args, f.MachineID)
	}
	if f.Status != "" {
		query += " AND o.status = ?"
		args = append(args, f.Status)
	}
	if f.LineInstanceID != "" {
		query += " AND ord.line_instance_id = ?"
		args = append(args, f.LineInstanceID)
	}

	query += " ORDER BY o.order_id, o.stage ASC"

	if f.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, f.Limit)
	}
	if f.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, f.Offset)
	}

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ops []*Operation
	for rows.Next() {
		op := &Operation{}
		if err := rows.Scan(&op.ID, &op.OrderID, &op.MachineID, &op.Stage, &op.Status,
			&op.PlannedCycleTimeMS, &op.GoodQty, &op.ScrapQty, &op.StartedAt, &op.CompletedAt); err != nil {
			return nil, err
		}
		ops = append(ops, op)
	}
	return ops, rows.Err()
}

// GetByMachineAndOrder finds the operation for a given machine and order.
func (r *OperationRepository) GetByMachineAndOrder(machineID, orderID string) (*Operation, error) {
	op := &Operation{}
	err := r.db.QueryRow(
		`SELECT id, order_id, machine_id, stage, status, planned_cycle_time_ms, good_qty, scrap_qty, started_at, completed_at
		 FROM operations WHERE machine_id = ? AND order_id = ?`, machineID, orderID,
	).Scan(&op.ID, &op.OrderID, &op.MachineID, &op.Stage, &op.Status,
		&op.PlannedCycleTimeMS, &op.GoodQty, &op.ScrapQty, &op.StartedAt, &op.CompletedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return op, err
}
