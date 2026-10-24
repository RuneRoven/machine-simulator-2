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

// FlowTags returns the material-flow tags present on every machine. They use
// the reserved range from NodeID 1001 so that adding them never renumbers the
// type-specific tags (7+) that clients already address.
func FlowTags() []TagDefinition {
	return []TagDefinition{
		{Name: "starved", DisplayName: "Starved", NodeID: 1001, DataType: "Boolean", Description: "Whether machine is waiting for a part (empty in-buffer, or no release at stage 1)"},
		{Name: "in_buffer_level", DisplayName: "In-Buffer Level", NodeID: 1002, DataType: "Int32", Description: "Parts in the buffer feeding this machine (0 at stage 1)"},
		{Name: "in_buffer_capacity", DisplayName: "In-Buffer Capacity", NodeID: 1003, DataType: "Int32", Description: "Capacity of the buffer feeding this machine (0 at stage 1)"},
		{Name: "out_buffer_level", DisplayName: "Out-Buffer Level", NodeID: 1004, DataType: "Int32", Description: "Parts in the buffer after this machine (0 at the last stage)"},
		{Name: "out_buffer_capacity", DisplayName: "Out-Buffer Capacity", NodeID: 1005, DataType: "Int32", Description: "Capacity of the buffer after this machine (0 at the last stage)"},
	}
}

// AllTagDefinitions returns a machine's tag definitions followed by the flow
// tags. Protocol adapters use it; the flow tags come last so Modbus register
// addresses of existing tags do not move.
func AllTagDefinitions(m Machine) []TagDefinition {
	return append(m.GetTagDefinitions(), FlowTags()...)
}

// BaseTagValues returns the standard tag values from a BaseMachine (must be called with lock held).
func BaseTagValues(m *BaseMachine) map[string]interface{} {
	tags := map[string]interface{}{
		"state":             string(m.GetStateUnlocked()),
		"cycle_count":       m.GetCycleCount(),
		"good_count":        m.GetGoodCount(),
		"scrap_count":       m.GetScrapCount(),
		"cycle_time_ms":     m.GetCycleTimeMS(),
		"blocked_by_buffer": m.GetBlockedByBuffer(),
	}
	m.addFlowValues(tags)
	return tags
}
