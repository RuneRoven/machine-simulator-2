package storage

import (
	"maps"
	"testing"
)

func TestLineMaterialSaveReplacesAndLoads(t *testing.T) {
	db, err := NewDB(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	r := NewLineMaterialRepository(db)

	if err := r.Save("window-frame-1", map[string]float64{"GLASS": 12, "PROFILE": 52.8}); err != nil {
		t.Fatal(err)
	}
	if err := r.Save("welding-1", map[string]float64{"TUBE": 3}); err != nil {
		t.Fatal(err)
	}
	// A later save replaces the line's store: used-up material is gone.
	if err := r.Save("window-frame-1", map[string]float64{"PROFILE": 8.8}); err != nil {
		t.Fatal(err)
	}
	got, err := r.Load()
	if err != nil {
		t.Fatal(err)
	}
	if !maps.Equal(got["window-frame-1"], map[string]float64{"PROFILE": 8.8}) || !maps.Equal(got["welding-1"], map[string]float64{"TUBE": 3}) {
		t.Errorf("loaded %v", got)
	}
	if err := r.Save("welding-1", map[string]float64{}); err != nil {
		t.Fatal(err)
	}
	if got, _ := r.Load(); got["welding-1"] != nil {
		t.Errorf("empty save left %v", got["welding-1"])
	}
}
