package machine

// TagDefinition describes a single tag (sensor/metric) exposed by a machine.
type TagDefinition struct {
	Name        string   `json:"name"`
	DisplayName string   `json:"display_name"`
	NodeID      int      `json:"node_id"`
	DataType    string   `json:"data_type"`
	Unit        string   `json:"unit,omitempty"`
	Description string   `json:"description"`
	Min         *float64 `json:"min,omitempty"`
	Max         *float64 `json:"max,omitempty"`
}

// FloatPtr returns a pointer to a float64 value (helper for tag min/max).
func FloatPtr(v float64) *float64 { return &v }

// BaseTags returns the standard tags present on every machine (NodeIDs 1-6).
func BaseTags() []TagDefinition {
	return []TagDefinition{
		{Name: "state", DisplayName: "State", NodeID: 1, DataType: "String", Description: "Machine state (IDLE, RUNNING, STOPPED, FAULT, MAINTENANCE)"},
		{Name: "cycle_count", DisplayName: "Cycle Count", NodeID: 2, DataType: "Int64", Description: "Total cycles completed"},
		{Name: "good_count", DisplayName: "Good Count", NodeID: 3, DataType: "Int64", Description: "Good parts produced"},
		{Name: "scrap_count", DisplayName: "Scrap Count", NodeID: 4, DataType: "Int64", Description: "Scrap parts produced"},
		{Name: "cycle_time_ms", DisplayName: "Cycle Time (ms)", NodeID: 5, DataType: "Int32", Unit: "ms", Description: "Current cycle time setting"},
		{Name: "blocked_by_buffer", DisplayName: "Blocked By Buffer", NodeID: 6, DataType: "Boolean", Description: "Whether machine is blocked by a full downstream buffer"},
	}
}

// BaseTagValues returns the standard tag values from a BaseMachine (must be called with lock held).
func BaseTagValues(m *BaseMachine) map[string]interface{} {
	return map[string]interface{}{
		"state":             string(m.GetStateUnlocked()),
		"cycle_count":       m.GetCycleCount(),
		"good_count":        m.GetGoodCount(),
		"scrap_count":       m.GetScrapCount(),
		"cycle_time_ms":     m.GetCycleTimeMS(),
		"blocked_by_buffer": m.GetBlockedByBuffer(),
	}
}
