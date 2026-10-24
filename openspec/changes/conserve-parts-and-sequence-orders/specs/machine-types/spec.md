## MODIFIED Requirements

### Requirement: Numeric NodeID on tag definitions
The `TagDefinition` struct SHALL include a `NodeID int` field. Base tags (state, cycle_count, good_count, scrap_count, cycle_time_ms, blocked_by_buffer) SHALL have reserved NodeIDs 1 through 6. Type-specific tags SHALL have sequential NodeIDs starting at 7. Flow tags present on every machine SHALL use the reserved range starting at 1001 and SHALL be listed after all other tags (`AllTagDefinitions`), so that existing NodeIDs and Modbus register addresses never change.

#### Scenario: Base tag NodeIDs are consistent
- **WHEN** `BaseTags()` is called
- **THEN** it returns the 6 tag definitions with NodeIDs 1–6, and `FlowTags()` returns NodeIDs 1001–1005

#### Scenario: Type-specific NodeIDs start at 7
- **WHEN** `AllTagDefinitions()` is called on `spot_welder`
- **THEN** the base tags keep NodeIDs 1–6, the type-specific tags start at NodeID 7, and the flow tags 1001–1005 come last

#### Scenario: NodeIDs are unique within a machine type
- **WHEN** `AllTagDefinitions()` is called on any machine type
- **THEN** no two tag definitions share the same NodeID

## ADDED Requirements

### Requirement: Flow tags on every machine
Every machine SHALL expose: `starved` (1001, Boolean, waiting on an empty in-buffer), `in_buffer_level` (1002, Int32), `in_buffer_capacity` (1003, Int32), `out_buffer_level` (1004, Int32), `out_buffer_capacity` (1005, Int32). The first stage SHALL report in-buffer level and capacity 0; the last stage SHALL report out-buffer level and capacity 0.

#### Scenario: Middle stage reports both buffers
- **WHEN** stage 2 of a 4-stage line is read
- **THEN** `in_buffer_level` equals the count of the buffer from stage 1 and `out_buffer_level` the count of the buffer to stage 3, with their capacities

#### Scenario: Starved flag
- **WHEN** stage 3 finishes a part and its in-buffer is empty
- **THEN** `starved` is true until a part arrives
