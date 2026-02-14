## Context

The simulator exposes machine tags via OPC-UA and Modbus. Currently, `TagDefinition` has `Name`, `DataType`, `Unit`, `Description`, `Min`, and `Max` fields. The OPC-UA adapter creates nodes using `ns.AddNewVariableNode(td.Name, initVal)`, which assigns library-internal node IDs. The original machine-simulator used explicit numeric node IDs (`ua.NewNumericNodeID(nsIdx, uint32(def.NodeID))`) with sequential integers per machine, allowing OPC-UA clients to address tags deterministically.

There are 27 machine types, each with ~15 type-specific tags plus 6 base tags (`state`, `cycle_count`, `good_count`, `scrap_count`, `cycle_time_ms`, `blocked_by_buffer`). DataType values are Go-style lowercase (`"float64"`, `"int64"`, `"bool"`). No project documentation exists.

## Goals / Non-Goals

**Goals:**

- Numeric OPC-UA node IDs matching the original simulator convention (`ns=2;i=1` through `ns=2;i=N`)
- Human-readable display names on all OPC-UA nodes (e.g., "Weld Current (kA)")
- OPC-UA-standard DataType naming (`"Double"`, `"Int64"`, `"Boolean"`, `"String"`)
- Comprehensive user documentation: README, config guide, API reference, tag catalog
- Backward-compatible REST API (new fields are additive)

**Non-Goals:**

- Changing tag `Name` values (the snake_case names remain as internal identifiers)
- Adding OPC-UA write support or method nodes
- Auto-generating documentation from code (docs are hand-written markdown)
- Changing Modbus register addressing (only adding display names to the register map)

## Decisions

### 1. TagDefinition struct changes

Add two fields to `TagDefinition`:

```go
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
```

- `NodeID`: Sequential integer (1, 2, 3...) unique within each machine type. Base tags get IDs 1-6, type-specific tags start at 7.
- `DisplayName`: Human-readable name with unit, e.g. `"Weld Current (kA)"`. Generated from `Description` + `Unit`.

**Rationale**: `NodeID` being per-machine-type (not globally unique) matches the OPC-UA model where each machine server has its own namespace. The same NodeID (e.g., 1 = `state`) means the same thing across all machines, simplifying client development.

**Alternative considered**: Auto-assign NodeIDs at registration time — rejected because deterministic IDs are essential for OPC-UA client configuration.

### 2. Base tag NodeID reservation

Base tags (present on every machine) get fixed NodeIDs 1-6:

| NodeID | Name | DisplayName | DataType |
|--------|------|-------------|----------|
| 1 | state | State | String |
| 2 | cycle_count | Cycle Count | Int64 |
| 3 | good_count | Good Count | Int64 |
| 4 | scrap_count | Scrap Count | Int64 |
| 5 | cycle_time_ms | Cycle Time (ms) | Int32 |
| 6 | blocked_by_buffer | Blocked By Buffer | Boolean |

Type-specific tags start at NodeID 7. Each machine type assigns them sequentially (7, 8, 9...).

**Rationale**: Reserved base IDs let clients know that NodeID 1 is always `state`, regardless of machine type. Matches the original simulator where common tags had consistent IDs.

### 3. DataType standardization

Map current values to OPC-UA types:

| Current | New | OPC-UA Type |
|---------|-----|-------------|
| `"float64"` | `"Double"` | ua.TypeIDDouble |
| `"float32"` | `"Float"` | ua.TypeIDFloat |
| `"int"`, `"int32"` | `"Int32"` | ua.TypeIDInt32 |
| `"int64"` | `"Int64"` | ua.TypeIDInt64 |
| `"bool"` | `"Boolean"` | ua.TypeIDBoolean |
| `"string"` | `"String"` | ua.TypeIDString |

The OPC-UA adapter's `switch` statement already handles both old and new names. Update the adapter to use OPC-UA type IDs when setting node metadata.

**Rationale**: OPC-UA clients expect standard type names. Using capitalized forms matches the OPC-UA specification.

### 4. OPC-UA adapter changes

In `RegisterMachine`, replace:
```go
node := ns.AddNewVariableNode(td.Name, initVal)
```
with numeric node ID creation and display name:
```go
nodeID := ua.NewNumericNodeID(ns.ID(), uint32(td.NodeID))
node := ns.AddNewVariableNodeWithID(td.Name, nodeID, initVal)
node.SetAttribute(ua.AttributeIDDisplayName, ua.LocalizedText{Text: td.DisplayName})
```

Remove the separate `stateNode` creation (it's now in base tags with NodeID 1).

Check if the gopcua server library supports `AddNewVariableNodeWithID` or a similar method for explicit node IDs. If not, use `ns.AddNode()` with a manually constructed `ua.NodeID`.

### 5. DisplayName generation pattern

For consistency, use: `Description` if it reads well as a display name, otherwise `titleCase(Name)` + ` (Unit)`.

Examples:
- `{Name: "weld_current_ka", Unit: "kA", Description: "Welding current"}` → `"Welding Current (kA)"`
- `{Name: "state", Description: "Machine state"}` → `"State"`
- `{Name: "cycle_count", Description: "Total cycles completed"}` → `"Cycle Count"`

Populate `DisplayName` explicitly in each machine type rather than generating it at runtime. This keeps it visible and editable in the code.

### 6. Documentation structure

```
docs/
├── README.md              # Overview, quick start, architecture summary
├── configuration.md       # Profiles, lines, global settings, adapters, env vars
├── api-reference.md       # REST endpoints, request/response schemas, WebSocket events
└── tag-catalog.md         # Per-machine-type tag table (Name, NodeID, DisplayName, DataType, Unit, Description)
```

- `docs/README.md`: How to build, run, Docker usage, profile selection
- `docs/configuration.md`: YAML file structure, how to create profiles and lines
- `docs/api-reference.md`: All REST endpoints with example requests/responses, WebSocket message format
- `docs/tag-catalog.md`: Auto-formatted tables for each machine type listing all tags with their NodeID, DisplayName, DataType, Unit, and ranges

The root `README.md` will be updated to point to `docs/` for detailed documentation.

**Rationale**: Separate docs files keep each topic focused. The tag catalog is the most unique artifact — OPC-UA/Modbus clients need this reference.

## Risks / Trade-offs

**[gopcua API for explicit node IDs]** → The `gopcua/opcua/server` package may not have a direct method for creating variable nodes with explicit numeric IDs. Mitigation: check `go doc github.com/gopcua/opcua/server` at implementation time. If needed, construct nodes manually using lower-level APIs or use `AddNode` with a pre-built `ua.Node`.

**[27 files to update]** → Every machine type file needs NodeID and DisplayName added to every tag definition. This is repetitive but straightforward. Mitigation: update `BaseTags()` first as the template, then batch-update type files. Each file is self-contained so this can be parallelized.

**[OPC-UA client compatibility]** → Existing clients using string-based browsing will break. Mitigation: this is intentional (proposal marks it BREAKING), and the target is compatibility with the original simulator's addressing scheme.

**[Documentation drift]** → Docs can become stale as the codebase evolves. Mitigation: keep docs in the repo (not a wiki), link them from the README, and keep them reference-focused (not tutorials that need constant updating).

## Open Questions

- Exact `gopcua/opcua/server` API for explicit numeric node IDs — needs verification at implementation time
- Whether to include a machine type index page in the tag catalog or one flat document with all types
