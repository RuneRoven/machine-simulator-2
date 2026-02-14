## 1. TagDefinition Struct Changes

- [x] 1.1 Add `NodeID int` and `DisplayName string` fields to `TagDefinition` in `internal/machine/tags.go`
- [x] 1.2 Update `BaseTags()` to include NodeID (1-6), DisplayName, and standardized DataType values for all 6 base tags
- [x] 1.3 Update `BaseTagValues()` if needed to ensure tag name keys still match

## 2. OPC-UA Adapter Updates

- [x] 2.1 Verify gopcua server API supports explicit numeric node IDs (check `go doc` for `AddNewVariableNodeWithID` or equivalent)
- [x] 2.2 Update OPC-UA `RegisterMachine` to create nodes with `ua.NewNumericNodeID(nsIdx, uint32(td.NodeID))` instead of string-based names
- [x] 2.3 Set `DisplayName` attribute on each OPC-UA node from `td.DisplayName`
- [x] 2.4 Update the DataType `switch` statement to handle new capitalized type names (`"Double"`, `"Int32"`, `"Int64"`, `"Boolean"`, `"String"`)
- [x] 2.5 Remove the separate `stateNode` creation (state is now base tag NodeID 1)

## 3. Modbus Adapter Updates

- [x] 3.1 Add `DisplayName string` field to `RegisterEntry` in `internal/protocol/modbus/modbus.go`
- [x] 3.2 Populate `DisplayName` from `TagDefinition.DisplayName` during `RegisterMachine`

## 4. Machine Type Updates — Automotive Domain

- [x] 4.1 Update `metal_forming.go`: add NodeID (7+), DisplayName, standardize DataType to OPC-UA conventions
- [x] 4.2 Update `spot_welder.go`: add NodeID (7+), DisplayName, standardize DataType
- [x] 4.3 Update `robot_welder.go`: add NodeID (7+), DisplayName, standardize DataType
- [x] 4.4 Update `robot_pick_place.go`: add NodeID (7+), DisplayName, standardize DataType
- [x] 4.5 Update `painting_booth.go`: add NodeID (7+), DisplayName, standardize DataType

## 5. Machine Type Updates — Electronics Domain

- [x] 5.1 Update `smt_placement.go`: add NodeID (7+), DisplayName, standardize DataType
- [x] 5.2 Update `reflow_oven.go`: add NodeID (7+), DisplayName, standardize DataType
- [x] 5.3 Update `wave_solder.go`: add NodeID (7+), DisplayName, standardize DataType
- [x] 5.4 Update `aoi_inspection.go`: add NodeID (7+), DisplayName, standardize DataType

## 6. Machine Type Updates — Food & Beverage Domain

- [x] 6.1 Update `filling_machine.go`: add NodeID (7+), DisplayName, standardize DataType
- [x] 6.2 Update `sealing_machine.go`: add NodeID (7+), DisplayName, standardize DataType
- [x] 6.3 Update `labeling_machine.go`: add NodeID (7+), DisplayName, standardize DataType

## 7. Machine Type Updates — Pharma Domain

- [x] 7.1 Update `reactor_vessel.go`: add NodeID (7+), DisplayName, standardize DataType
- [x] 7.2 Update `pharma_filler.go`: add NodeID (7+), DisplayName, standardize DataType
- [x] 7.3 Update `capping_machine.go`: add NodeID (7+), DisplayName, standardize DataType

## 8. Machine Type Updates — Window Domain

- [x] 8.1 Update `profile_cutter.go`: add NodeID (7+), DisplayName, standardize DataType
- [x] 8.2 Update `corner_welder.go`: add NodeID (7+), DisplayName, standardize DataType
- [x] 8.3 Update `glass_setter.go`: add NodeID (7+), DisplayName, standardize DataType

## 9. Machine Type Updates — Furniture Domain

- [x] 9.1 Update `cnc_router.go`: add NodeID (7+), DisplayName, standardize DataType
- [x] 9.2 Update `edge_bander.go`: add NodeID (7+), DisplayName, standardize DataType
- [x] 9.3 Update `assembly_press.go`: add NodeID (7+), DisplayName, standardize DataType

## 10. Machine Type Updates — Metal Parts Domain

- [x] 10.1 Update `laser_cutter.go`: add NodeID (7+), DisplayName, standardize DataType
- [x] 10.2 Update `press_brake.go`: add NodeID (7+), DisplayName, standardize DataType
- [x] 10.3 Update `deburring_machine.go`: add NodeID (7+), DisplayName, standardize DataType

## 11. Machine Type Updates — Plastic Parts Domain

- [x] 11.1 Update `injection_molding.go`: add NodeID (7+), DisplayName, standardize DataType
- [x] 11.2 Update `extruder.go`: add NodeID (7+), DisplayName, standardize DataType
- [x] 11.3 Update `trimming_press.go`: add NodeID (7+), DisplayName, standardize DataType

## 12. Documentation — Project README

- [x] 12.1 Create `docs/README.md` with project overview, prerequisites, build/run instructions, Docker usage, profile selection
- [x] 12.2 Create or update root `README.md` with links to docs/ files

## 13. Documentation — Configuration Guide

- [x] 13.1 Create `docs/configuration.md` covering global.yaml settings
- [x] 13.2 Document line template YAML structure with annotated examples
- [x] 13.3 Document profile format and how to create custom profiles
- [x] 13.4 Document OPC-UA and Modbus adapter configuration

## 14. Documentation — API Reference

- [x] 14.1 Create `docs/api-reference.md` with all REST endpoints, methods, request/response schemas
- [x] 14.2 Add curl examples for each endpoint
- [x] 14.3 Document WebSocket endpoint, connection, and event message format with example payloads

## 15. Documentation — Tag Catalog

- [x] 15.1 Create `docs/tag-catalog.md` with base tags section (NodeID 1-6) and OPC-UA addressing guide
- [x] 15.2 Add tag tables for all 27 machine types (Name, NodeID, DisplayName, DataType, Unit, Description)

## 16. Build Verification

- [x] 16.1 Verify all packages compile (`go build ./...`)
- [x] 16.2 Run all tests (`go test ./...`)
- [x] 16.3 Rebuild Docker image and verify startup with demo-mixed profile
