package storage

import (
	"database/sql"
	"fmt"
)

// LineMaterialRepository keeps each line's line-side material store, so
// material staged to a line is still there after a simulator restart, as it
// would be on a shop floor.
type LineMaterialRepository struct {
	db *sql.DB
}

// NewLineMaterialRepository creates a new line material repository.
func NewLineMaterialRepository(db *sql.DB) *LineMaterialRepository {
	return &LineMaterialRepository{db: db}
}

// Load returns the stored material per line instance.
func (r *LineMaterialRepository) Load() (map[string]map[string]float64, error) {
	rows, err := r.db.Query(`SELECT line_instance_id, material, qty FROM line_material`)
	if err != nil {
		return nil, fmt.Errorf("loading line material: %w", err)
	}
	defer rows.Close()
	out := map[string]map[string]float64{}
	for rows.Next() {
		var line, material string
		var qty float64
		if err := rows.Scan(&line, &material, &qty); err != nil {
			return nil, fmt.Errorf("scanning line material: %w", err)
		}
		if out[line] == nil {
			out[line] = map[string]float64{}
		}
		out[line][material] = qty
	}
	return out, rows.Err()
}

// Save replaces the stored material of one line.
func (r *LineMaterialRepository) Save(lineID string, stock map[string]float64) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM line_material WHERE line_instance_id = ?`, lineID); err != nil {
		return fmt.Errorf("saving line material of %s: %w", lineID, err)
	}
	for material, qty := range stock {
		if _, err := tx.Exec(`INSERT INTO line_material (line_instance_id, material, qty) VALUES (?, ?, ?)`,
			lineID, material, qty); err != nil {
			return fmt.Errorf("saving line material of %s: %w", lineID, err)
		}
	}
	return tx.Commit()
}
