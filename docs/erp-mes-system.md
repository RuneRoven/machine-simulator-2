# ERP & MES System

## Overview

The simulator includes an ERP (Enterprise Resource Planning) and MES (Manufacturing Execution System) that manage the full production order lifecycle. The ERP generates and releases orders while the MES dispatches them to production lines, tracks progress, and closes orders when complete.

---

## Order Lifecycle

```
CREATED ──► RELEASED ──► IN_PROGRESS ──► CLOSED
                                     └──► CANCELLED
```

| Status | Set by | Description | Timestamp |
|--------|--------|-------------|-----------|
| `CREATED` | ERP | Order generated, awaiting release | `created_at` |
| `RELEASED` | ERP/MES | Released to MES for dispatch | — |
| `IN_PROGRESS` | MES | Dispatched to line, production started | `started_at` |
| `CLOSED` | MES | Completed (`good_qty >= planned_qty`) | `closed_at` |
| `CANCELLED` | API | Manually cancelled | `closed_at` |

---

## Data Structures

### Order

```json
{
  "id": "uuid",
  "product_id": "FRAME-WELD-A",
  "line_instance_id": "line-automotive-welding-1",
  "line_template": "automotive-welding",
  "status": "IN_PROGRESS",
  "planned_qty": 100,
  "good_qty": 45,
  "scrap_qty": 3,
  "priority": 75,
  "created_at": "2026-01-15T08:00:00Z",
  "started_at": "2026-01-15T08:02:00Z",
  "closed_at": null
}
```

- **`good_qty` / `scrap_qty`**: Only updated from the **last machine** in the line (finished goods output). Intermediate stages track quantities at the operation level.
- **`priority`**: Auto-orders get a random priority between 50–99. Currently used by the `priority` dispatch strategy.

### Operation

Each order produces one operation per machine stage in the line.

```json
{
  "id": "uuid",
  "order_id": "uuid",
  "machine_id": "line-automotive-welding-1-stage-1",
  "stage": 1,
  "status": "IN_PROGRESS",
  "planned_cycle_time_ms": 4500,
  "good_qty": 48,
  "scrap_qty": 2,
  "started_at": "2026-01-15T08:02:00Z",
  "completed_at": null
}
```

- **`planned_cycle_time_ms`**: The expected cycle time for this operation, set at dispatch time. Uses the recipe stage override if defined, otherwise falls back to the machine's default cycle time. This field enables OEE performance ratio calculation.

Operation statuses:

| Status | Description |
|--------|-------------|
| `PENDING` | Created but machine not yet started |
| `IN_PROGRESS` | Machine actively processing |
| `COMPLETED` | Set when the entire order completes |
| `CANCELLED` | Set when the order is cancelled |

### Recipe

Recipes define product-specific parameters per line template. They are configured in the line template YAML files under `config/lines/`.

```yaml
recipes:
  - product_id: "FRAME-WELD-A"
    description: "Automotive frame welding assembly"
    planned_qty_range: [50, 200]
    overrides:
      1:
        cycle_time_ms: 4500
        scrap_rate: 0.03
      2:
        cycle_time_ms: 6000
        breakdown_probability: 0.001
```

- **`planned_qty_range`**: Used by ERP auto-order generation to pick a random quantity in this range.
- **`overrides`**: Per-stage overrides for `cycle_time_ms`, `scrap_rate`, and `breakdown_probability`. These are applied to machines when an order is dispatched and reverted when the order completes.

---

## ERP Simulator

### Configuration

```yaml
erp:
  mode: "auto"                    # "auto" or "manual"
  auto_order_interval_ms: 30000   # Base interval between auto-generated orders
  max_backlog: 5                  # Max pending orders per line instance
```

### Auto Mode

In auto mode, the ERP runs a background loop that periodically generates orders:

1. **Interval**: Every `auto_order_interval_ms` milliseconds, adjusted by `time_scale` (e.g., at 10x time scale, a 30s interval becomes 3s)
2. **Product selection**: Randomly picks from all registered products across all active lines
3. **Quantity**: Random value within the recipe's `planned_qty_range`
4. **Backlog control**: Counts orders with status `CREATED`, `RELEASED`, or `IN_PROGRESS` per line. Skips generation if count >= `max_backlog`
5. **Auto-release**: MES subscribes to `order.created` events and immediately releases and dispatches new orders

### Manual Mode

In manual mode:
- No automatic order generation
- Orders are created via the REST API: `POST /api/orders`
- MES still auto-releases and dispatches orders on creation

### Product Registration

At startup, the ERP registers all products from all active line instances. Each registration ties a product ID to a specific line:

```
Line: line-automotive-welding-1
  Product: FRAME-WELD-A (qty range: 50–200)
  Product: FRAME-WELD-B (qty range: 100–500)
```

---

## MES Simulator

### Configuration

```yaml
mes:
  dispatch_interval_ms: 5000      # How often to check for released orders
  dispatch_strategy: "round-robin" # "round-robin", "least-busy", "priority"
```

### Dispatch Strategies

When multiple instances of the same line template exist, the MES uses a strategy to select which instance receives an order:

| Strategy | Behavior |
|----------|----------|
| `round-robin` | Cycles through instances sequentially |
| `least-busy` | Picks the instance with fewest `IN_PROGRESS` orders |
| `priority` | Prefers idle instances; falls back to least-busy |

### Order Dispatch Flow

When an order is released, the MES dispatches it:

1. **Look up recipe** for the order's product and line template
2. **Create operations** — one per machine stage in the line, each with:
   - `planned_cycle_time_ms` from recipe override or machine default
   - Status `PENDING`
3. **Update order** to `IN_PROGRESS` (sets `started_at`)
4. **Apply recipe overrides** to machines (cycle time, scrap rate, breakdown probability)
5. **Assign order** to all machines in the line
6. **Start first machine** only — downstream machines wait for upstream parts

### Progress Tracking

When a machine completes a cycle:

1. **Good parts** are routed to the downstream buffer; scrap parts are counted but not routed
2. **Operation quantities** are updated (per-machine `good_qty` / `scrap_qty`)
3. **Downstream machine** starts when it receives a part from its input buffer
4. **Order quantities** are updated only from the **last machine** (finished goods)
5. When `good_qty >= planned_qty` on the order, the MES:
   - Marks all operations as `COMPLETED`
   - Closes the order (`CLOSED`)
   - Reverts recipe overrides on all machines
   - Clears order assignments

---

## ERP-to-MES-to-Machine Data Flow

This shows the conceptual data exchange at each boundary:

### ERP → MES (Order Release)

```json
{
  "id": "uuid",
  "product_id": "FRAME-WELD-A",
  "line_instance_id": "line-automotive-welding-1",
  "line_template": "automotive-welding",
  "status": "RELEASED",
  "planned_qty": 100,
  "priority": 75
}
```

### MES → Machine (Dispatch)

For each machine in the line, MES creates an operation and assigns it:

```json
{
  "order_id": "uuid",
  "operation_id": "uuid",
  "product_id": "FRAME-WELD-A",
  "planned_cycle_time_ms": 4500,
  "recipe_overrides": {
    "cycle_time_ms": 4500,
    "scrap_rate": 0.03
  }
}
```

### Machine → MES (Cycle Complete)

On each cycle, the machine publishes an event:

```json
{
  "machine_id": "line-automotive-welding-1-stage-1",
  "order_id": "uuid",
  "good": true
}
```

### MES → ERP (Order Closed)

When the order completes:

```json
{
  "id": "uuid",
  "status": "CLOSED",
  "planned_qty": 100,
  "good_qty": 100,
  "scrap_qty": 7,
  "started_at": "2026-01-15T08:02:00Z",
  "closed_at": "2026-01-15T09:48:00Z"
}
```

---

## OEE Inputs

The system captures the raw data needed for OEE (Overall Equipment Effectiveness) calculation:

| OEE Component | Data Source |
|---------------|------------|
| **Availability** | Machine `run_time_sec` and `down_time_sec` (tracked per machine via tags) |
| **Performance** | `planned_cycle_time_ms` on operations vs actual cycle time from machine |
| **Quality** | `good_qty` and `scrap_qty` on orders and operations |

---

## REST API Endpoints

### Orders

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/orders` | List orders (filter by `status`, `line_instance_id`, `product_id`) |
| `GET` | `/api/orders/{id}` | Get single order |
| `POST` | `/api/orders` | Create a manual order |
| `PUT` | `/api/orders/{id}/status` | Change order status |
| `DELETE` | `/api/orders/{id}` | Cancel an order |

### Operations

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/operations` | List operations (filter by `order_id`, `machine_id`, `status`, `line_instance_id`) |
| `GET` | `/api/operations/{id}` | Get single operation |
| `GET` | `/api/orders/{id}/operations` | Get all operations for an order |

### Create Order Request

```json
{
  "line_instance_id": "line-automotive-welding-1",
  "line_template": "automotive-welding",
  "product_id": "FRAME-WELD-A",
  "planned_qty": 100,
  "priority": 75
}
```

### Recipes

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/recipes` | All recipes grouped by template |
| `GET` | `/api/lines/{id}/recipes` | Recipes for a specific line |

---

## Event System

The ERP and MES communicate via an internal event bus:

| Event | Publisher | Subscribers | Payload |
|-------|-----------|-------------|---------|
| `order.created` | ERP | MES (auto-release) | Order object |
| `order.released` | ERP | — | Order object |
| `order.progress` | MES | WebSocket hub, Webhook | `{order_id, good_qty, scrap_qty, planned_qty}` |
| `order.closed` | MES | WebSocket hub, Webhook | Order object |
| `machine.cycle_complete` | Engine | MES (progress tracking), Webhook | `{machine_id, good: bool}` |
| `machine.state_change` | Machine | WebSocket hub, Webhook | `{previous_state, new_state}` |
