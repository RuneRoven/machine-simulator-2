package storage

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

const schema = `
CREATE TABLE IF NOT EXISTS orders (
    id TEXT PRIMARY KEY,
    product_id TEXT NOT NULL,
    line_instance_id TEXT NOT NULL,
    line_template TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'CREATED',
    planned_qty INTEGER NOT NULL,
    good_qty INTEGER NOT NULL DEFAULT 0,
    scrap_qty INTEGER NOT NULL DEFAULT 0,
    priority INTEGER NOT NULL DEFAULT 50,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    started_at TIMESTAMP,
    closed_at TIMESTAMP
);

CREATE TABLE IF NOT EXISTS operations (
    id TEXT PRIMARY KEY,
    order_id TEXT NOT NULL REFERENCES orders(id),
    machine_id TEXT NOT NULL,
    stage INTEGER NOT NULL,
    status TEXT NOT NULL DEFAULT 'PENDING',
    good_qty INTEGER NOT NULL DEFAULT 0,
    scrap_qty INTEGER NOT NULL DEFAULT 0,
    started_at TIMESTAMP,
    completed_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_orders_status ON orders(status);
CREATE INDEX IF NOT EXISTS idx_orders_line_instance ON orders(line_instance_id);
CREATE INDEX IF NOT EXISTS idx_operations_order_id ON operations(order_id);
CREATE INDEX IF NOT EXISTS idx_operations_machine_id ON operations(machine_id);
`

// NewDB opens a SQLite database and runs migrations.
// Use ":memory:" for testing.
func NewDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	// Enable WAL mode for better concurrent access
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("setting WAL mode: %w", err)
	}

	// Wait up to 5s for locks instead of failing immediately with SQLITE_BUSY
	if _, err := db.Exec("PRAGMA busy_timeout=5000"); err != nil {
		db.Close()
		return nil, fmt.Errorf("setting busy timeout: %w", err)
	}

	// SQLite only supports one writer at a time; serialize all access
	db.SetMaxOpenConns(1)

	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("creating schema: %w", err)
	}

	return db, nil
}
