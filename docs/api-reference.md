# API Reference

The machine simulator exposes a REST API (default port `8081`) and a WebSocket endpoint for real-time events.

Base URL: `http://localhost:8081`

All responses use `Content-Type: application/json`. Errors return a JSON object with an `error` field:

```json
{"error": "description of the problem"}
```

---

## REST API

### Health

#### GET /health

Returns the health status of the simulator.

**Response** `200 OK`

```json
{"status": "ok"}
```

**Example**

```bash
curl http://localhost:8081/health
```

---

### Lines

#### GET /api/lines

List all production lines.

**Response** `200 OK`

```json
[
  {
    "instance_id": "line-aluminum-windows-1",
    "template_name": "aluminum-windows",
    "domain": "window-manufacturing",
    "machine_ids": ["profile-cutter-1", "corner-welder-1", "glass-setter-1"],
    "recipes": [
      {"product_id": "ALU-STD-001", "description": "Standard Aluminum Window"}
    ]
  }
]
```

**Example**

```bash
curl http://localhost:8081/api/lines
```

---

#### GET /api/lines/{id}

Get details for a single production line.

**Path Parameters**

| Parameter | Type   | Description          |
|-----------|--------|----------------------|
| `id`      | string | Line instance ID     |

**Response** `200 OK`

```json
{
  "instance_id": "line-aluminum-windows-1",
  "template_name": "aluminum-windows",
  "domain": "window-manufacturing",
  "machine_ids": ["profile-cutter-1", "corner-welder-1", "glass-setter-1"],
  "recipes": [
    {"product_id": "ALU-STD-001", "description": "Standard Aluminum Window"}
  ]
}
```

**Error** `404 Not Found` -- line not found

**Example**

```bash
curl http://localhost:8081/api/lines/line-aluminum-windows-1
```

---

#### GET /api/lines/{id}/buffers

Get all inter-machine buffers for a line.

**Path Parameters**

| Parameter | Type   | Description          |
|-----------|--------|----------------------|
| `id`      | string | Line instance ID     |

**Response** `200 OK`

```json
[
  {
    "id": "buf-profile-cutter-1-corner-welder-1",
    "upstream_machine_id": "profile-cutter-1",
    "downstream_machine_id": "corner-welder-1",
    "capacity": 10,
    "count": 3,
    "total_throughput": 142
  }
]
```

**Error** `404 Not Found` -- line not found

**Example**

```bash
curl http://localhost:8081/api/lines/line-aluminum-windows-1/buffers
```

---

#### POST /api/lines/{id}/start

Start all idle machines in a line. Machines already running or in other states are unaffected.

**Path Parameters**

| Parameter | Type   | Description          |
|-----------|--------|----------------------|
| `id`      | string | Line instance ID     |

**Response** `200 OK`

```json
{"status": "started"}
```

**Error** `404 Not Found` -- line not found

**Example**

```bash
curl -X POST http://localhost:8081/api/lines/line-aluminum-windows-1/start
```

---

#### POST /api/lines/{id}/stop

Stop all running machines in a line. Machines not in the RUNNING state are unaffected.

**Path Parameters**

| Parameter | Type   | Description          |
|-----------|--------|----------------------|
| `id`      | string | Line instance ID     |

**Response** `200 OK`

```json
{"status": "stopped"}
```

**Error** `404 Not Found` -- line not found

**Example**

```bash
curl -X POST http://localhost:8081/api/lines/line-aluminum-windows-1/stop
```

---

### Machines

#### GET /api/machines

List all machines across all lines.

**Response** `200 OK`

```json
[
  {
    "id": "profile-cutter-1",
    "type": "profile-cutter",
    "role": "cutting",
    "line_instance_id": "line-aluminum-windows-1",
    "state": "RUNNING",
    "order_id": "ORD-00042",
    "operation_id": "OP-00042-1",
    "tags": {
      "state": "RUNNING",
      "cycle_time_ms": 5000,
      "good_count": 87,
      "scrap_count": 2,
      "temperature": 42.5,
      "vibration": 1.2
    }
  }
]
```

**Example**

```bash
curl http://localhost:8081/api/machines
```

---

#### GET /api/machines/{id}

Get details for a single machine.

**Path Parameters**

| Parameter | Type   | Description   |
|-----------|--------|---------------|
| `id`      | string | Machine ID    |

**Response** `200 OK`

```json
{
  "id": "profile-cutter-1",
  "type": "profile-cutter",
  "role": "cutting",
  "line_instance_id": "line-aluminum-windows-1",
  "state": "RUNNING",
  "order_id": "ORD-00042",
  "operation_id": "OP-00042-1",
  "tags": {
    "state": "RUNNING",
    "cycle_time_ms": 5000,
    "good_count": 87,
    "scrap_count": 2
  }
}
```

**Error** `404 Not Found` -- machine not found

**Example**

```bash
curl http://localhost:8081/api/machines/profile-cutter-1
```

---

#### GET /api/machines/{id}/tags

Get the current OPC UA / Modbus tag values for a machine. Tags are the real-time process variables exposed by the machine (state, counters, sensor readings, etc.).

**Path Parameters**

| Parameter | Type   | Description   |
|-----------|--------|---------------|
| `id`      | string | Machine ID    |

**Response** `200 OK`

```json
{
  "state": "RUNNING",
  "cycle_time_ms": 5000,
  "good_count": 87,
  "scrap_count": 2,
  "temperature": 42.5,
  "vibration": 1.2,
  "speed_rpm": 3200
}
```

Tag names and types vary by machine type.

**Error** `404 Not Found` -- machine not found

**Example**

```bash
curl http://localhost:8081/api/machines/profile-cutter-1/tags
```

---

#### POST /api/machines/{id}/command

Send a control command to a machine.

**Path Parameters**

| Parameter | Type   | Description   |
|-----------|--------|---------------|
| `id`      | string | Machine ID    |

**Request Body**

```json
{"command": "START"}
```

| Field     | Type   | Required | Description                                              |
|-----------|--------|----------|----------------------------------------------------------|
| `command` | string | yes      | One of: `START`, `STOP`, `RESET`, `FAULT`, `MAINTENANCE` |

**Valid commands and their effects:**

| Command       | Description                                             |
|---------------|---------------------------------------------------------|
| `START`       | Transition from IDLE to RUNNING                         |
| `STOP`        | Transition from RUNNING to STOPPED                      |
| `RESET`       | Transition from FAULT, STOPPED, or MAINTENANCE to IDLE  |
| `FAULT`       | Transition from RUNNING to FAULT                        |
| `MAINTENANCE` | Transition from RUNNING to MAINTENANCE                  |

**Valid state transitions:**

| Current State | Allowed Transitions                        |
|---------------|--------------------------------------------|
| IDLE          | RUNNING                                    |
| RUNNING       | IDLE, STOPPED, FAULT, MAINTENANCE          |
| STOPPED       | IDLE                                       |
| FAULT         | IDLE                                       |
| MAINTENANCE   | IDLE                                       |

**Response** `200 OK`

```json
{"status": "ok"}
```

**Errors**
- `400 Bad Request` -- invalid request body
- `404 Not Found` -- machine not found
- `409 Conflict` -- invalid state transition

**Example**

```bash
curl -X POST http://localhost:8081/api/machines/profile-cutter-1/command \
  -H "Content-Type: application/json" \
  -d '{"command": "START"}'
```

---

### Orders

#### GET /api/orders

List production orders with optional filtering.

**Query Parameters**

| Parameter         | Type   | Description                          |
|-------------------|--------|--------------------------------------|
| `status`          | string | Filter by status (e.g. `RELEASED`)   |
| `line_instance_id`| string | Filter by line instance ID           |
| `product_id`      | string | Filter by product ID                 |
| `limit`           | int    | Maximum number of results            |
| `offset`          | int    | Number of results to skip            |

**Response** `200 OK`

```json
[
  {
    "id": "ORD-00042",
    "product_id": "ALU-STD-001",
    "line_instance_id": "line-aluminum-windows-1",
    "line_template": "aluminum-windows",
    "status": "IN_PROGRESS",
    "planned_qty": 100,
    "good_qty": 87,
    "scrap_qty": 2,
    "priority": 50,
    "created_at": "2026-02-13T10:00:00Z",
    "started_at": "2026-02-13T10:05:00Z"
  }
]
```

**Example**

```bash
curl "http://localhost:8081/api/orders?status=RELEASED&limit=10"
```

---

#### GET /api/orders/{id}

Get details for a single order.

**Path Parameters**

| Parameter | Type   | Description   |
|-----------|--------|---------------|
| `id`      | string | Order ID      |

**Response** `200 OK`

```json
{
  "id": "ORD-00042",
  "product_id": "ALU-STD-001",
  "line_instance_id": "line-aluminum-windows-1",
  "line_template": "aluminum-windows",
  "status": "IN_PROGRESS",
  "planned_qty": 100,
  "good_qty": 87,
  "scrap_qty": 2,
  "priority": 50,
  "created_at": "2026-02-13T10:00:00Z",
  "started_at": "2026-02-13T10:05:00Z"
}
```

**Error** `404 Not Found` -- order not found

**Example**

```bash
curl http://localhost:8081/api/orders/ORD-00042
```

---

#### POST /api/orders

Create a new production order.

**Request Body**

```json
{
  "line_instance_id": "line-aluminum-windows-1",
  "line_template": "aluminum-windows",
  "product_id": "ALU-STD-001",
  "planned_qty": 100,
  "priority": 50
}
```

| Field              | Type   | Required | Description                                  |
|--------------------|--------|----------|----------------------------------------------|
| `line_instance_id` | string | yes      | Target line instance                         |
| `line_template`    | string | yes      | Line template name                           |
| `product_id`       | string | yes      | Product/recipe to produce                    |
| `planned_qty`      | int    | yes      | Number of units to produce                   |
| `priority`         | int    | no       | Order priority (default: 50, higher = first) |

**Response** `201 Created`

```json
{
  "id": "ORD-00043",
  "product_id": "ALU-STD-001",
  "line_instance_id": "line-aluminum-windows-1",
  "line_template": "aluminum-windows",
  "status": "CREATED",
  "planned_qty": 100,
  "good_qty": 0,
  "scrap_qty": 0,
  "priority": 50,
  "created_at": "2026-02-13T12:00:00Z"
}
```

**Error** `400 Bad Request` -- invalid request body

**Example**

```bash
curl -X POST http://localhost:8081/api/orders \
  -H "Content-Type: application/json" \
  -d '{
    "line_instance_id": "line-aluminum-windows-1",
    "line_template": "aluminum-windows",
    "product_id": "ALU-STD-001",
    "planned_qty": 100,
    "priority": 50
  }'
```

---

#### PUT /api/orders/{id}/status

Update the status of an order. Only specific transitions are supported.

**Path Parameters**

| Parameter | Type   | Description   |
|-----------|--------|---------------|
| `id`      | string | Order ID      |

**Request Body**

```json
{"status": "RELEASED"}
```

| Field    | Type   | Required | Description                                  |
|----------|--------|----------|----------------------------------------------|
| `status` | string | yes      | One of: `RELEASED`, `CLOSED`, `CANCELLED`    |

**Status transitions:**

| Status      | Effect                                           |
|-------------|--------------------------------------------------|
| `RELEASED`  | Release the order for production via ERP          |
| `CLOSED`    | Close a completed order via ERP                   |
| `CANCELLED` | Cancel the order via MES                          |

**Response** `200 OK`

```json
{"status": "RELEASED"}
```

**Errors**
- `400 Bad Request` -- invalid request body or invalid status value
- `409 Conflict` -- invalid state transition for the current order state

**Example**

```bash
curl -X PUT http://localhost:8081/api/orders/ORD-00042/status \
  -H "Content-Type: application/json" \
  -d '{"status": "RELEASED"}'
```

---

#### DELETE /api/orders/{id}

Cancel an order. This is a convenience endpoint equivalent to setting status to `CANCELLED`.

**Path Parameters**

| Parameter | Type   | Description   |
|-----------|--------|---------------|
| `id`      | string | Order ID      |

**Response** `200 OK`

```json
{"status": "cancelled"}
```

**Error** `409 Conflict` -- order cannot be cancelled in its current state

**Example**

```bash
curl -X DELETE http://localhost:8081/api/orders/ORD-00042
```

---

### Recipes

#### GET /api/recipes

List all available product recipes across all line templates.

**Response** `200 OK`

```json
[
  {
    "product_id": "ALU-STD-001",
    "description": "Standard Aluminum Window",
    "line_template": "aluminum-windows",
    "operations": [
      {"machine_type": "profile-cutter", "cycle_time_ms": 5000},
      {"machine_type": "corner-welder", "cycle_time_ms": 8000},
      {"machine_type": "glass-setter", "cycle_time_ms": 6000}
    ]
  }
]
```

**Example**

```bash
curl http://localhost:8081/api/recipes
```

---

#### GET /api/lines/{id}/recipes

List recipes available for a specific line.

**Path Parameters**

| Parameter | Type   | Description          |
|-----------|--------|----------------------|
| `id`      | string | Line instance ID     |

**Response** `200 OK`

```json
[
  {
    "product_id": "ALU-STD-001",
    "description": "Standard Aluminum Window",
    "line_template": "aluminum-windows",
    "operations": [...]
  }
]
```

**Error** `404 Not Found` -- line not found

**Example**

```bash
curl http://localhost:8081/api/lines/line-aluminum-windows-1/recipes
```

---

### Simulation

#### GET /api/simulation

Get the current simulation state.

**Response** `200 OK`

```json
{
  "time_scale": 1.0,
  "random_factor": 0.5,
  "machine_count": 12,
  "line_count": 3
}
```

| Field           | Type  | Description                                      |
|-----------------|-------|--------------------------------------------------|
| `time_scale`    | float | Current time multiplier (1.0 = real time)        |
| `random_factor` | float | Randomness factor for breakdowns and scrap       |
| `machine_count` | int   | Total number of machines in the simulation       |
| `line_count`    | int   | Total number of production lines                 |

**Example**

```bash
curl http://localhost:8081/api/simulation
```

---

#### PUT /api/simulation/timescale

Set the simulation time scale. Higher values make the simulation run faster.

**Request Body**

```json
{"time_scale": 5.0}
```

| Field        | Type  | Required | Description                        |
|--------------|-------|----------|------------------------------------|
| `time_scale` | float | yes      | Time multiplier (range: 0.1--100)  |

**Response** `200 OK`

```json
{"time_scale": 5.0}
```

**Error** `400 Bad Request` -- value out of range or invalid body

**Example**

```bash
curl -X PUT http://localhost:8081/api/simulation/timescale \
  -H "Content-Type: application/json" \
  -d '{"time_scale": 5.0}'
```

---

#### PUT /api/simulation/random-factor

Set the random factor that controls variability in breakdowns, scrap rates, and cycle time jitter.

**Request Body**

```json
{"random_factor": 0.3}
```

| Field           | Type  | Required | Description                       |
|-----------------|-------|----------|-----------------------------------|
| `random_factor` | float | yes      | Randomness factor (range: 0--1)   |

- `0` = fully deterministic (no random breakdowns or scrap)
- `1` = maximum randomness

**Response** `200 OK`

```json
{"random_factor": 0.3}
```

**Error** `400 Bad Request` -- value out of range or invalid body

**Example**

```bash
curl -X PUT http://localhost:8081/api/simulation/random-factor \
  -H "Content-Type: application/json" \
  -d '{"random_factor": 0.3}'
```

---

### Settings

#### GET /api/settings/machines

Get configurable settings for all machines.

**Response** `200 OK`

```json
[
  {
    "id": "profile-cutter-1",
    "type": "profile-cutter",
    "line_instance_id": "line-aluminum-windows-1",
    "cycle_time_ms": 5000,
    "scrap_rate": 0.02,
    "breakdown_probability": 0.001
  }
]
```

**Example**

```bash
curl http://localhost:8081/api/settings/machines
```

---

#### PUT /api/settings/machines/{id}

Update settings for a specific machine. All fields are optional; only provided fields are updated.

**Path Parameters**

| Parameter | Type   | Description   |
|-----------|--------|---------------|
| `id`      | string | Machine ID    |

**Request Body**

```json
{
  "cycle_time_ms": 4000,
  "scrap_rate": 0.01,
  "breakdown_probability": 0.005
}
```

| Field                   | Type  | Required | Description                                   |
|-------------------------|-------|----------|-----------------------------------------------|
| `cycle_time_ms`         | int   | no       | Cycle time in milliseconds                    |
| `scrap_rate`            | float | no       | Probability of producing scrap (0--1)         |
| `breakdown_probability` | float | no       | Probability of breakdown per cycle (0--1)     |

**Response** `200 OK`

```json
{"status": "updated"}
```

**Errors**
- `400 Bad Request` -- invalid request body
- `404 Not Found` -- machine not found

**Example**

```bash
curl -X PUT http://localhost:8081/api/settings/machines/profile-cutter-1 \
  -H "Content-Type: application/json" \
  -d '{"cycle_time_ms": 4000, "scrap_rate": 0.01}'
```

---

#### GET /api/settings/buffers

Get settings for all inter-machine buffers.

**Response** `200 OK`

```json
[
  {
    "id": "buf-profile-cutter-1-corner-welder-1",
    "line": "line-aluminum-windows-1",
    "upstream_machine_id": "profile-cutter-1",
    "downstream_machine_id": "corner-welder-1",
    "capacity": 10,
    "count": 3
  }
]
```

**Example**

```bash
curl http://localhost:8081/api/settings/buffers
```

---

#### PUT /api/settings/buffers/{id}

Update the capacity of a buffer.

**Path Parameters**

| Parameter | Type   | Description   |
|-----------|--------|---------------|
| `id`      | string | Buffer ID     |

**Request Body**

```json
{"capacity": 20}
```

| Field      | Type | Required | Description                       |
|------------|------|----------|-----------------------------------|
| `capacity` | int  | yes      | New buffer capacity (minimum: 1)  |

**Response** `200 OK`

```json
{"status": "updated"}
```

**Errors**
- `400 Bad Request` -- invalid body or capacity < 1
- `404 Not Found` -- buffer not found

**Example**

```bash
curl -X PUT http://localhost:8081/api/settings/buffers/buf-profile-cutter-1-corner-welder-1 \
  -H "Content-Type: application/json" \
  -d '{"capacity": 20}'
```

---

### Modbus

#### GET /api/modbus/register-map

Get the Modbus register map showing how machine tags are mapped to Modbus holding/input registers.

**Response** `200 OK`

The response shape depends on the Modbus adapter configuration. Typically a mapping of machine IDs to register addresses.

**Error** `404 Not Found` -- Modbus adapter not enabled

**Example**

```bash
curl http://localhost:8081/api/modbus/register-map
```

---

## WebSocket

### Connection

**Endpoint:** `ws://localhost:8081/ws`

Connect using any WebSocket client. The server accepts connections from any origin.

**Connection settings:**

| Setting         | Value      |
|-----------------|------------|
| Ping interval   | 30 seconds |
| Read timeout    | 60 seconds |
| Write timeout   | 10 seconds |
| Read limit      | 512 bytes  |
| Send buffer     | 256 messages |

The server sends periodic ping frames every 30 seconds. Clients must respond with pong frames within the 60-second read timeout or the connection will be closed.

**Example connection with websocat:**

```bash
websocat ws://localhost:8081/ws
```

**Example connection with JavaScript:**

```javascript
const ws = new WebSocket("ws://localhost:8081/ws");
ws.onmessage = (event) => {
  const msg = JSON.parse(event.data);
  console.log(msg.type, msg);
};
```

---

### Message Format

All messages sent from the server follow this structure:

```json
{
  "type": "machine.state_change",
  "machine_id": "profile-cutter-1",
  "line_id": "line-aluminum-windows-1",
  "data": { ... },
  "timestamp": 1707820800000
}
```

| Field        | Type   | Description                                        |
|--------------|--------|----------------------------------------------------|
| `type`       | string | Event type identifier                              |
| `machine_id` | string | Machine that produced the event (may be empty)    |
| `line_id`    | string | Line associated with the event (may be empty)     |
| `data`       | object | Event-specific payload                             |
| `timestamp`  | int    | Unix timestamp in milliseconds                     |

---

### Event Types

#### machine.state_change

Emitted when a machine transitions between states.

```json
{
  "type": "machine.state_change",
  "machine_id": "profile-cutter-1",
  "line_id": "line-aluminum-windows-1",
  "data": {
    "previous_state": "IDLE",
    "new_state": "RUNNING"
  },
  "timestamp": 1707820800000
}
```

---

#### machine.cycle_complete

Emitted when a machine completes one production cycle.

```json
{
  "type": "machine.cycle_complete",
  "machine_id": "profile-cutter-1",
  "line_id": "line-aluminum-windows-1",
  "data": {
    "cycle_time_ms": 5000,
    "is_scrap": false,
    "good_count": 88,
    "scrap_count": 2
  },
  "timestamp": 1707820805000
}
```

---

#### line.part_moved

Emitted when a part moves through a buffer from one machine to the next.

```json
{
  "type": "line.part_moved",
  "machine_id": "",
  "line_id": "line-aluminum-windows-1",
  "data": {
    "buffer_id": "buf-profile-cutter-1-corner-welder-1",
    "from_machine": "profile-cutter-1",
    "to_machine": "corner-welder-1",
    "buffer_count": 2
  },
  "timestamp": 1707820806000
}
```

---

#### order.created

Emitted when a new production order is created.

```json
{
  "type": "order.created",
  "machine_id": "",
  "line_id": "line-aluminum-windows-1",
  "data": {
    "order_id": "ORD-00043",
    "product_id": "ALU-STD-001",
    "planned_qty": 100
  },
  "timestamp": 1707820800000
}
```

---

#### order.released

Emitted when an order is released for production.

```json
{
  "type": "order.released",
  "machine_id": "",
  "line_id": "line-aluminum-windows-1",
  "data": {
    "order_id": "ORD-00043"
  },
  "timestamp": 1707820801000
}
```

---

#### order.progress

Emitted when an order's production count is updated (after each cycle).

```json
{
  "type": "order.progress",
  "machine_id": "",
  "line_id": "line-aluminum-windows-1",
  "data": {
    "order_id": "ORD-00043",
    "good_qty": 50,
    "scrap_qty": 1,
    "planned_qty": 100
  },
  "timestamp": 1707820900000
}
```

---

#### order.closed

Emitted when an order finishes production or is manually closed.

```json
{
  "type": "order.closed",
  "machine_id": "",
  "line_id": "line-aluminum-windows-1",
  "data": {
    "order_id": "ORD-00043",
    "good_qty": 98,
    "scrap_qty": 2
  },
  "timestamp": 1707821000000
}
```

---

#### buffer.full

Emitted when a buffer reaches its maximum capacity, causing the upstream machine to become blocked.

```json
{
  "type": "buffer.full",
  "machine_id": "",
  "line_id": "line-aluminum-windows-1",
  "data": {
    "buffer_id": "buf-profile-cutter-1-corner-welder-1",
    "capacity": 10
  },
  "timestamp": 1707820810000
}
```

---

#### buffer.empty

Emitted when a buffer becomes empty, causing the downstream machine to become starved.

```json
{
  "type": "buffer.empty",
  "machine_id": "",
  "line_id": "line-aluminum-windows-1",
  "data": {
    "buffer_id": "buf-profile-cutter-1-corner-welder-1"
  },
  "timestamp": 1707820820000
}
```

---

#### line.deadlock_recovered

Emitted when the simulator detects and recovers from a deadlock (all machines in a line are blocked or starved).

```json
{
  "type": "line.deadlock_recovered",
  "machine_id": "",
  "line_id": "line-aluminum-windows-1",
  "data": {
    "action": "buffer_flush",
    "details": "Flushed buffer buf-profile-cutter-1-corner-welder-1"
  },
  "timestamp": 1707820830000
}
```

---

## CORS

The API allows cross-origin requests with the following settings:

| Setting              | Value                                      |
|----------------------|--------------------------------------------|
| Allowed origins      | `*` (all origins)                          |
| Allowed methods      | `GET`, `POST`, `PUT`, `DELETE`, `OPTIONS`  |
| Allowed headers      | `Accept`, `Content-Type`                   |
| Allow credentials    | `false`                                    |
| Max age (preflight)  | 300 seconds                                |
