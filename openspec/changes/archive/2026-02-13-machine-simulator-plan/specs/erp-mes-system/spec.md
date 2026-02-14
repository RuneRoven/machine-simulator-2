## ADDED Requirements

### Requirement: ERP adapts to active line configuration
The ERP simulator SHALL discover all active line instances and their product recipes at startup. Auto-generated orders SHALL only target products that can be manufactured on currently active lines.

#### Scenario: ERP registers products from active lines
- **WHEN** the profile activates `automotive-welding-1` (recipes: DOOR-PANEL-A, FENDER-B) and `electronics-smt-1` (recipes: PCB-MAIN, PCB-SENSOR)
- **THEN** the ERP has 4 products available for order generation, each tied to its line instance

#### Scenario: Auto-order generation
- **WHEN** ERP is in auto mode with a 30-second interval
- **THEN** every 30 seconds (adjusted by time scale), ERP creates a new order by selecting a random active line instance and a random product from its recipe list

#### Scenario: Order planned quantity from recipe range
- **WHEN** ERP creates an order for product `DOOR-PANEL-A` which has `planned_qty_range: [50, 200]`
- **THEN** the order's planned_qty is a random value between 50 and 200

### Requirement: ERP order lifecycle
Orders SHALL progress through states: CREATED → RELEASED → IN_PROGRESS → CLOSED. Orders MAY be CANCELLED at any point before CLOSED.

#### Scenario: Order creation
- **WHEN** ERP creates a new order (auto or via API)
- **THEN** the order is stored with status CREATED, a product_id, line_instance_id, and planned_qty

#### Scenario: Order released to MES
- **WHEN** ERP releases an order
- **THEN** the order status changes to RELEASED and becomes available for MES dispatch

#### Scenario: Order completion
- **WHEN** all operations for an order are complete and good_qty >= planned_qty
- **THEN** the order status changes to CLOSED with a closed_at timestamp

#### Scenario: Order cancellation
- **WHEN** a user cancels an order via API
- **THEN** the order status changes to CANCELLED, active operations are stopped, and recipe overrides are reverted on affected machines

### Requirement: ERP backlog control
The ERP SHALL enforce a maximum pending order backlog per line instance. When a line's backlog reaches the limit, ERP SHALL skip that line for auto-order generation until orders complete.

#### Scenario: Backlog limit reached
- **WHEN** `automotive-welding-1` has 5 pending orders and `max_backlog` is 5
- **THEN** ERP does not create new orders targeting `automotive-welding-1` until at least one order completes

#### Scenario: Backlog limit not global
- **WHEN** `automotive-welding-1` is at max backlog but `electronics-smt-1` has capacity
- **THEN** ERP can still generate orders for `electronics-smt-1`

### Requirement: MES dispatches operations per line stage
When MES receives a RELEASED order, it SHALL create one operation per machine stage in the target line instance. Operations are numbered sequentially (OP10, OP20, etc.) and executed in order.

#### Scenario: Create operations for a 4-stage line
- **WHEN** MES dispatches an order for `automotive-welding-1` (4 stages)
- **THEN** 4 operations are created: OP10 (stage 1), OP20 (stage 2), OP30 (stage 3), OP40 (stage 4)

#### Scenario: Operations execute sequentially
- **WHEN** OP10 completes (first stage)
- **THEN** OP20 begins (second stage machine starts processing parts from the buffer)

#### Scenario: Recipe overrides applied at dispatch
- **WHEN** MES dispatches an order for product `DOOR-PANEL-A`
- **THEN** MES calls `ApplyRecipeOverrides` on each machine in the line instance with the product's per-stage overrides

### Requirement: MES dispatch strategies
MES SHALL support configurable dispatch strategies for selecting which line instance receives an order when multiple instances of the same template are active: round-robin, least-busy, and priority.

#### Scenario: Round-robin dispatch
- **WHEN** dispatch strategy is round-robin and `automotive-welding-1` received the last order
- **THEN** the next order for automotive-welding goes to `automotive-welding-2`

#### Scenario: Least-busy dispatch
- **WHEN** dispatch strategy is least-busy and `automotive-welding-1` has 3 pending orders while `automotive-welding-2` has 1
- **THEN** the next order goes to `automotive-welding-2`

### Requirement: MES tracks order progress
MES SHALL monitor machine cycle completions and update operation quantities (good_qty, scrap_qty) in real time. An event SHALL be published when order progress changes.

#### Scenario: Operation quantity update
- **WHEN** a machine at stage 2 completes a cycle producing a good part
- **THEN** the corresponding operation's good_qty increments by 1

#### Scenario: Order progress event
- **WHEN** an operation's good_qty changes
- **THEN** an `order.progress` event is published with the order ID, current good_qty, scrap_qty, and planned_qty

### Requirement: Manual order creation via API
Users SHALL be able to create orders manually via the REST API, specifying the target line instance and product ID. Manual orders follow the same lifecycle as auto-generated orders.

#### Scenario: Create manual order
- **WHEN** a user sends `POST /api/orders` with `{ "line_instance_id": "automotive-welding-1", "product_id": "DOOR-PANEL-A", "planned_qty": 100 }`
- **THEN** an order is created with the specified parameters and status CREATED
