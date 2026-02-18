## ADDED Requirements

### Requirement: Project README
The project SHALL include a `docs/README.md` covering project overview, prerequisites (Go, Docker), build and run instructions, Docker Compose usage, profile selection, and links to detailed documentation files.

#### Scenario: Quick start instructions
- **WHEN** a new user reads `docs/README.md`
- **THEN** they find step-by-step instructions to build and run the simulator locally and via Docker

#### Scenario: Profile selection
- **WHEN** a user reads the README
- **THEN** they find how to select a profile via `--profile` flag or `SIMULATOR_PROFILE` environment variable, with a list of available profiles

### Requirement: Configuration guide
The project SHALL include a `docs/configuration.md` documenting the YAML configuration system: global settings, line template structure, profile format, adapter configuration, and environment variable overrides.

#### Scenario: Line template documentation
- **WHEN** a user reads `docs/configuration.md`
- **THEN** they find the YAML structure for line templates including machine stages, buffer config, and recipe definitions with annotated examples

#### Scenario: Profile documentation
- **WHEN** a user reads `docs/configuration.md`
- **THEN** they find how to create a custom profile selecting specific line templates with instance counts and global overrides

#### Scenario: Adapter configuration documentation
- **WHEN** a user reads `docs/configuration.md`
- **THEN** they find OPC-UA and Modbus adapter YAML settings including enable/disable, ports, hostnames, and update rates

### Requirement: API reference
The project SHALL include a `docs/api-reference.md` documenting all REST API endpoints with HTTP method, path, request body schema, response schema, and example curl commands. It SHALL also document the WebSocket endpoint and event message format.

#### Scenario: REST endpoint documentation
- **WHEN** a user reads `docs/api-reference.md`
- **THEN** every REST endpoint is listed with method, path, parameters, request/response JSON schema, and a curl example

#### Scenario: WebSocket documentation
- **WHEN** a user reads `docs/api-reference.md`
- **THEN** they find the WebSocket endpoint (`/ws`), connection instructions, and the JSON message format for all event types with example payloads

### Requirement: Tag catalog
The project SHALL include a `docs/tag-catalog.md` listing every machine type with a table of all its tags. Each tag entry SHALL show Name, NodeID, DisplayName, DataType, Unit, and Description.

#### Scenario: Machine type tag table
- **WHEN** a user reads the tag catalog section for `spot_welder`
- **THEN** they see a table with all tags including base tags (NodeID 1-6) and type-specific tags (NodeID 7+) with Name, DisplayName, DataType, Unit, and Description

#### Scenario: Base tags documented
- **WHEN** a user reads the tag catalog
- **THEN** they find a section explaining the 6 base tags (state, cycle_count, good_count, scrap_count, cycle_time_ms, blocked_by_buffer) with their reserved NodeIDs 1-6, common to all machine types

#### Scenario: OPC-UA addressing guide
- **WHEN** a user reads the tag catalog
- **THEN** they find instructions on how to address tags via OPC-UA using numeric node IDs (e.g., `ns=1;i=7` for the first type-specific tag) and the namespace URI pattern

### Requirement: Root README update
The project root `README.md` SHALL link to the `docs/` directory for detailed documentation.

#### Scenario: Root README links to docs
- **WHEN** a user opens the project root
- **THEN** the `README.md` contains links to `docs/README.md`, `docs/configuration.md`, `docs/api-reference.md`, and `docs/tag-catalog.md`
