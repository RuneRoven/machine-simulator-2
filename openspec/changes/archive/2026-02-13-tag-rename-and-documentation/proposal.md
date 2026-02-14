## Why

The simulator's OPC-UA tag addressing uses implicit string-based node IDs (the gopcua library auto-assigns them), making it incompatible with clients configured for the original machine-simulator which uses explicit numeric node IDs (`ns=2;i=1`, `ns=2;i=2`, etc.). Tags also lack meaningful display names for OPC-UA browsing. Additionally, the project has no user-facing documentation, so new users have no guide for configuration, API usage, or tag discovery.

## What Changes

- **BREAKING**: Add explicit numeric `NodeID` field to `TagDefinition` struct and assign sequential IDs to all 27 machine types' tag definitions
- **BREAKING**: Update OPC-UA adapter to create nodes with numeric node IDs (`ua.NewNumericNodeID`) instead of string-based names, and set proper display names on each node
- Add a `DisplayName` field to `TagDefinition` for human-readable names shown in OPC-UA browsers (e.g., "Weld Current (kA)" instead of `weld_current_kA`)
- Standardize `DataType` field values to OPC-UA conventions: `"Double"`, `"Int32"`, `"Int64"`, `"Boolean"`, `"String"` (currently lowercase `"float64"`, `"int"`, `"bool"`)
- Add comprehensive project documentation: README, configuration guide, API reference, and tag catalog

## Capabilities

### New Capabilities
- `project-documentation`: User and developer documentation covering simulator setup, configuration (profiles, lines, adapters), REST API reference, WebSocket events, tag naming conventions, and a full tag catalog per machine type

### Modified Capabilities
- `machine-types`: Add `NodeID int` and `DisplayName string` fields to `TagDefinition`; assign sequential numeric IDs (starting at 1) per machine type; standardize DataType values to OPC-UA conventions; update all 27 machine type files
- `protocol-adapters`: Update OPC-UA adapter to use `ua.NewNumericNodeID(nsIdx, uint32(td.NodeID))` for node creation and set `DisplayName` attribute on each node; ensure Modbus register map also exposes display names

## Impact

- **Code**: `internal/machine/tags.go` (TagDefinition struct), all 27 files in `internal/machine/types/`, `internal/protocol/opcua/opcua.go`, `internal/protocol/modbus/modbus.go` (register map response)
- **APIs**: `GET /api/machines/{id}/tags` response shape gains `node_id` and `display_name` fields; `GET /api/modbus/register-map` response gains `display_name`
- **Protocol clients**: OPC-UA clients must use numeric node IDs instead of string-based browsing (matches original simulator convention)
- **Documentation**: New `docs/` directory with markdown files
