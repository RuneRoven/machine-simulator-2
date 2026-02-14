## MODIFIED Requirements

### Requirement: OPC-UA adapter (built-in)
The system SHALL include a built-in OPC-UA adapter that creates one OPC-UA server per registered machine. Each server SHALL expose the machine's tag definitions as OPC-UA nodes with numeric node IDs derived from `TagDefinition.NodeID`, human-readable display names from `TagDefinition.DisplayName`, and values updating at a configurable rate.

#### Scenario: OPC-UA server per machine
- **WHEN** 4 machines are registered with the OPC-UA adapter
- **THEN** 4 independent OPC-UA servers start, each on a unique port

#### Scenario: OPC-UA port auto-assignment
- **WHEN** the OPC-UA adapter config specifies `base_port: 4840`
- **THEN** machines are assigned ports sequentially: 4840, 4841, 4842, etc., in the order they are registered

#### Scenario: OPC-UA numeric node IDs
- **WHEN** an OPC-UA client connects to a machine's server with namespace index 2
- **THEN** each tag is addressable via `ns=2;i=<NodeID>` where NodeID matches the tag's `TagDefinition.NodeID` field (e.g., `ns=2;i=1` for state, `ns=2;i=7` for the first type-specific tag)

#### Scenario: OPC-UA display names
- **WHEN** an OPC-UA client browses a machine's node tree
- **THEN** each node has a DisplayName attribute matching the tag's `TagDefinition.DisplayName` (e.g., "Welding Current (kA)")

#### Scenario: OPC-UA tag exposure
- **WHEN** an OPC-UA client connects to a machine's server
- **THEN** the client can browse and read all tag definitions returned by `machine.GetTagDefinitions()` with current values from `machine.GetTags()`

#### Scenario: OPC-UA update rate
- **WHEN** OPC-UA config specifies `update_rate_ms: 100`
- **THEN** the adapter refreshes its tag cache from the machine every 100ms

#### Scenario: OPC-UA data types match tag definitions
- **WHEN** the OPC-UA adapter creates a node for a tag with `DataType: "Double"`
- **THEN** the OPC-UA node's data type is set to `ua.TypeIDDouble` and the initial value is `float64(0)`

### Requirement: Modbus TCP adapter (built-in)
The system SHALL include a built-in Modbus TCP adapter that runs a single gateway server. Each registered machine is mapped to a Modbus slave unit ID with holding registers derived from the machine's tags.

#### Scenario: Single Modbus gateway server
- **WHEN** machines are registered with the Modbus adapter
- **THEN** a single Modbus TCP server starts on the configured port (default 502)

#### Scenario: Dynamic register mapping from machine tags
- **WHEN** a machine with 20 tags is registered with the Modbus adapter
- **THEN** the adapter maps those tags to sequential holding registers for that machine's unit ID, and exposes the register map via the REST API

#### Scenario: Modbus unit ID assignment
- **WHEN** machines are registered with the Modbus adapter
- **THEN** each machine receives a unique unit ID starting from 1, assigned in registration order (or from adapter config if specified)

#### Scenario: Modbus register map includes display names
- **WHEN** a client requests `GET /api/modbus/register-map`
- **THEN** each register entry includes a `display_name` field from the tag's `TagDefinition.DisplayName`
