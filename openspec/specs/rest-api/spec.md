## ADDED Requirements

### Requirement: Line instance endpoints
The REST API SHALL provide endpoints for listing and inspecting active line instances.

#### Scenario: List all active line instances
- **WHEN** `GET /api/lines`
- **THEN** response includes all active line instances with their name, template, domain, machine count, buffer count, and current status

#### Scenario: Get line instance detail
- **WHEN** `GET /api/lines/{instanceId}`
- **THEN** response includes the full line state: machine IDs and states, buffer IDs with fill levels, active order (if any), active product, and available recipes

#### Scenario: List line buffers
- **WHEN** `GET /api/lines/{instanceId}/buffers`
- **THEN** response includes each buffer with its upstream/downstream machine IDs, current count, max capacity, and throughput counter

#### Scenario: Start all machines on a line
- **WHEN** `POST /api/lines/{instanceId}/start`
- **THEN** all machines on that line instance receive a START command

#### Scenario: Stop all machines on a line
- **WHEN** `POST /api/lines/{instanceId}/stop`
- **THEN** all machines on that line instance receive a STOP command

### Requirement: Machine endpoints
The REST API SHALL provide endpoints for listing machines, viewing details, reading tags, and sending commands.

#### Scenario: List all machines
- **WHEN** `GET /api/machines`
- **THEN** response includes all active machines with their ID, type, role, line instance, and current state

#### Scenario: Get machine detail
- **WHEN** `GET /api/machines/{id}`
- **THEN** response includes all machine data: state, type, role, line instance, current order, cycle count, good/scrap counts, settings, and tag values

#### Scenario: Get machine tags
- **WHEN** `GET /api/machines/{id}/tags`
- **THEN** response includes current tag values as a key-value map

#### Scenario: Send machine command
- **WHEN** `POST /api/machines/{id}/command` with body `{ "command": "START" }`
- **THEN** the command is sent to the machine and the response confirms acceptance

### Requirement: Order endpoints
The REST API SHALL provide CRUD operations for orders with filtering capabilities.

#### Scenario: List orders with filters
- **WHEN** `GET /api/orders?status=IN_PROGRESS&line_instance=automotive-welding-1`
- **THEN** response includes only orders matching both filters, with pagination support (limit/offset)

#### Scenario: Create order
- **WHEN** `POST /api/orders` with `{ "line_instance_id": "...", "product_id": "...", "planned_qty": 100 }`
- **THEN** the order is created, validated (product must be in line's recipe list), and returned with status CREATED

#### Scenario: Create order with invalid product
- **WHEN** `POST /api/orders` specifies a product_id not in the target line's recipe list
- **THEN** response is 400 with an error message listing valid products for that line

#### Scenario: Get order detail
- **WHEN** `GET /api/orders/{id}`
- **THEN** response includes the order with all fields plus its operations list

#### Scenario: Cancel order
- **WHEN** `DELETE /api/orders/{id}` (or `PUT /api/orders/{id}/status` with `CANCELLED`)
- **THEN** the order is cancelled if not already CLOSED

### Requirement: Recipe endpoints
The REST API SHALL expose recipe information for active line instances.

#### Scenario: List recipes for a line instance
- **WHEN** `GET /api/lines/{instanceId}/recipes`
- **THEN** response includes all recipes from the line template with product_id, description, planned_qty_range, and per-stage overrides summary

#### Scenario: List all available products
- **WHEN** `GET /api/recipes`
- **THEN** response includes all products across all active line instances, grouped by line template

### Requirement: Simulation control endpoints
The REST API SHALL provide endpoints for controlling simulation parameters at runtime.

#### Scenario: Get simulation status
- **WHEN** `GET /api/simulation`
- **THEN** response includes time_scale, random_factor, uptime, active profile name, and machine/line counts

#### Scenario: Update time scale
- **WHEN** `PUT /api/simulation/timescale` with `{ "value": 10.0 }`
- **THEN** the simulation time scale changes to 10.0 and the response confirms the new value

#### Scenario: Update random factor
- **WHEN** `PUT /api/simulation/random-factor` with `{ "value": 0.2 }`
- **THEN** the random variance factor changes to 0.2

### Requirement: Settings endpoints
The REST API SHALL provide endpoints for runtime machine and buffer settings adjustments.

#### Scenario: Update machine settings
- **WHEN** `PUT /api/settings/machines/{id}` with `{ "cycle_time_ms": 3000, "scrap_rate": 0.05 }`
- **THEN** the machine's runtime settings are updated (these are separate from recipe overrides and persist until the next recipe change)

#### Scenario: Update buffer capacity
- **WHEN** `PUT /api/settings/buffers/{id}` with `{ "capacity": 20 }`
- **THEN** the buffer's max capacity changes to 20

### Requirement: WebSocket real-time endpoint
The system SHALL provide a WebSocket endpoint at `/ws` for streaming real-time state updates including machine state changes, order events, buffer level changes, and simulation parameter changes.

#### Scenario: WebSocket connection
- **WHEN** a client connects to `ws://localhost:8081/ws`
- **THEN** the client receives JSON messages for all state change events

#### Scenario: Machine state change message
- **WHEN** a machine transitions from IDLE to RUNNING
- **THEN** connected WebSocket clients receive `{ "type": "machine.state_change", "machine_id": "...", "old_state": "IDLE", "new_state": "RUNNING" }`

### Requirement: Health endpoint
The system SHALL expose `GET /health` returning 200 OK when the simulator is running.

#### Scenario: Health check
- **WHEN** `GET /health`
- **THEN** response is 200 with `{ "status": "ok", "profile": "<active-profile>", "uptime": "..." }`
