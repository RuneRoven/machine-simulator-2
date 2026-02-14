## MODIFIED Requirements

### Requirement: Machine interface compliance
Every machine type SHALL implement the full Machine interface: identity (ID, Type, Role, LineInstanceID), lifecycle (Start, Stop, SendCommand), state (GetState, GetBaseData), tags (GetTags, GetTagDefinitions), production (AssignOrder, ClearOrder, ApplyRecipeOverrides), and line integration (SetOnPartComplete, CanAcceptPart, IsBlockedByBuffer, SetBlockedByBuffer).

#### Scenario: BaseMachine provides default implementation
- **WHEN** a new machine type embeds `*BaseMachine`
- **THEN** it inherits default implementations for all interface methods except `GetTags()`, `GetTagDefinitions()`, and `Start()`

#### Scenario: Type-specific tag definitions
- **WHEN** a machine type implements `GetTagDefinitions()`
- **THEN** it returns tag definitions specific to that machine type, each with Name, NodeID, DisplayName, DataType, Unit, and Description fields populated

### Requirement: Minimum 15 tags per machine type
Each machine type SHALL define a minimum of 15 tag definitions covering process parameters, quality metrics, and machine health indicators.

#### Scenario: Tag count check
- **WHEN** `GetTagDefinitions()` is called on any machine type
- **THEN** at least 15 tag definitions are returned, each with a unique NodeID within that machine type

## ADDED Requirements

### Requirement: Numeric NodeID on tag definitions
The `TagDefinition` struct SHALL include a `NodeID int` field. Base tags (state, cycle_count, good_count, scrap_count, cycle_time_ms, blocked_by_buffer) SHALL have reserved NodeIDs 1 through 6. Type-specific tags SHALL have sequential NodeIDs starting at 7.

#### Scenario: Base tag NodeIDs are consistent
- **WHEN** `BaseTags()` is called
- **THEN** it returns 6 tag definitions with NodeIDs 1, 2, 3, 4, 5, 6 respectively

#### Scenario: Type-specific NodeIDs start at 7
- **WHEN** `GetTagDefinitions()` is called on `spot_welder`
- **THEN** the first 6 entries are base tags (NodeID 1-6) and the type-specific tags start at NodeID 7

#### Scenario: NodeIDs are unique within a machine type
- **WHEN** `GetTagDefinitions()` is called on any machine type
- **THEN** no two tag definitions share the same NodeID

### Requirement: DisplayName on tag definitions
The `TagDefinition` struct SHALL include a `DisplayName string` field containing a human-readable label suitable for OPC-UA browsing. The display name SHALL include the unit in parentheses when a unit is present (e.g., "Weld Current (kA)").

#### Scenario: DisplayName with unit
- **WHEN** a tag has `Name: "weld_current_ka"`, `Unit: "kA"`, `Description: "Welding current"`
- **THEN** its `DisplayName` is `"Welding Current (kA)"`

#### Scenario: DisplayName without unit
- **WHEN** a tag has `Name: "state"`, `Unit: ""`, `Description: "Machine state"`
- **THEN** its `DisplayName` is `"State"`

#### Scenario: All tags have display names
- **WHEN** `GetTagDefinitions()` is called on any machine type
- **THEN** every returned TagDefinition has a non-empty `DisplayName`

### Requirement: Standardized DataType values
All tag definitions SHALL use OPC-UA standard data type names: `"Double"` (was `"float64"`), `"Float"` (was `"float32"`), `"Int32"` (was `"int"` or `"int32"`), `"Int64"` (was `"int64"`), `"Boolean"` (was `"bool"`), `"String"` (was `"string"`).

#### Scenario: Float tag uses Double
- **WHEN** a tag previously had `DataType: "float64"`
- **THEN** it now has `DataType: "Double"`

#### Scenario: Integer tag uses Int32 or Int64
- **WHEN** a tag previously had `DataType: "int"` or `DataType: "int64"`
- **THEN** it now has `DataType: "Int32"` or `DataType: "Int64"` respectively

#### Scenario: Boolean tag uses Boolean
- **WHEN** a tag previously had `DataType: "bool"`
- **THEN** it now has `DataType: "Boolean"`

#### Scenario: String tag uses String
- **WHEN** a tag previously had `DataType: "string"`
- **THEN** it now has `DataType: "String"`
