package storage

import (
	"database/sql"
	"fmt"
	"time"
)

// Order represents a production order.
type Order struct {
	ID             string     `json:"id"`
	ProductID      string     `json:"product_id"`
	LineInstanceID string     `json:"line_instance_id"`
	LineTemplate   string     `json:"line_template"`
	Status         string     `json:"status"`
	PlannedQty     int        `json:"planned_qty"`
	GoodQty        int        `json:"good_qty"`
	ScrapQty       int        `json:"scrap_qty"`
	Priority       int        `json:"priority"`
	CreatedAt      time.Time  `json:"created_at"`
	StartedAt      *time.Time `json:"started_at,omitempty"`
	ClosedAt       *time.Time `json:"closed_at,omitempty"`
}

// OrderFilter holds filter criteria for listing orders.
type OrderFilter struct {
	Status         string
	LineInstanceID string
	ProductID      string
	Limit          int
	Offset         int
}

// OrderRepository provides CRUD for orders.
type OrderRepository struct {
	db *sql.DB
}

// NewOrderRepository creates a new order repository.
func NewOrderRepository(db *sql.DB) *OrderRepository {
	return &OrderRepository{db: db}
}

// Create inserts a new order.
func (r *OrderRepository) Create(o *Order) error {
	_, err := r.db.Exec(
		`INSERT INTO orders (id, product_id, line_instance_id, line_template, status, planned_qty, good_qty, scrap_qty, priority, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		o.ID, o.ProductID, o.LineInstanceID, o.LineTemplate, o.Status,
		o.PlannedQty, o.GoodQty, o.ScrapQty, o.Priority, o.CreatedAt,
	)
	return err
}

// GetByID retrieves an order by ID.
func (r *OrderRepository) GetByID(id string) (*Order, error) {
	o := &Order{}
	err := r.db.QueryRow(
		`SELECT id, product_id, line_instance_id, line_template, status, planned_qty, good_qty, scrap_qty, priority, created_at, started_at, closed_at
		 FROM orders WHERE id = ?`, id,
	).Scan(&o.ID, &o.ProductID, &o.LineInstanceID, &o.LineTemplate, &o.Status,
		&o.PlannedQty, &o.GoodQty, &o.ScrapQty, &o.Priority, &o.CreatedAt, &o.StartedAt, &o.ClosedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("order %q not found", id)
	}
	return o, err
}

// List returns orders matching the filter.
func (r *OrderRepository) List(f OrderFilter) ([]*Order, error) {
	query := `SELECT id, product_id, line_instance_id, line_template, status, planned_qty, good_qty, scrap_qty, priority, created_at, started_at, closed_at FROM orders WHERE 1=1`
	args := []interface{}{}

	if f.Status != "" {
		query += " AND status = ?"
		args = append(args, f.Status)
	}
	if f.LineInstanceID != "" {
		query += " AND line_instance_id = ?"
		args = append(args, f.LineInstanceID)
	}
	if f.ProductID != "" {
		query += " AND product_id = ?"
		args = append(args, f.ProductID)
	}

	query += " ORDER BY created_at DESC"

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

	var orders []*Order
	for rows.Next() {
		o := &Order{}
		if err := rows.Scan(&o.ID, &o.ProductID, &o.LineInstanceID, &o.LineTemplate, &o.Status,
			&o.PlannedQty, &o.GoodQty, &o.ScrapQty, &o.Priority, &o.CreatedAt, &o.StartedAt, &o.ClosedAt); err != nil {
			return nil, err
		}
		orders = append(orders, o)
	}
	return orders, rows.Err()
}

// UpdateStatus changes an order's status.
func (r *OrderRepository) UpdateStatus(id, status string) error {
	now := time.Now()
	var query string
	var args []interface{}

	switch status {
	case "IN_PROGRESS":
		query = `UPDATE orders SET status = ?, started_at = ? WHERE id = ?`
		args = []interface{}{status, now, id}
	case "CLOSED", "CANCELLED":
		query = `UPDATE orders SET status = ?, closed_at = ? WHERE id = ?`
		args = []interface{}{status, now, id}
	default:
		query = `UPDATE orders SET status = ? WHERE id = ?`
		args = []interface{}{status, id}
	}

	res, err := r.db.Exec(query, args...)
	if err != nil {
		return err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("order %q not found", id)
	}
	return nil
}

// UpdateQuantities updates good_qty and scrap_qty for an order.
func (r *OrderRepository) UpdateQuantities(id string, goodQty, scrapQty int) error {
	_, err := r.db.Exec(
		`UPDATE orders SET good_qty = ?, scrap_qty = ? WHERE id = ?`,
		goodQty, scrapQty, id,
	)
	return err
}

// CountByStatus returns the count of orders with the given status for a line.
func (r *OrderRepository) CountByStatus(lineInstanceID, status string) (int, error) {
	var count int
	err := r.db.QueryRow(
		`SELECT COUNT(*) FROM orders WHERE line_instance_id = ? AND status = ?`,
		lineInstanceID, status,
	).Scan(&count)
	return count, err
}
