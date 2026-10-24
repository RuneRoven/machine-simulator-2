## MODIFIED Requirements

### Requirement: ERP order lifecycle
Orders SHALL progress through states: CREATED → RELEASED → IN_PROGRESS → CLOSED. Orders MAY be CANCELLED at any point before CLOSED. At most one order per line instance SHALL be IN_PROGRESS at any time.

#### Scenario: Order creation
- **WHEN** ERP creates a new order (auto or via API)
- **THEN** the order is stored with status CREATED, a product_id, line_instance_id, and planned_qty

#### Scenario: Order released to MES
- **WHEN** ERP releases an order
- **THEN** the order status changes to RELEASED and it joins the line's queue

#### Scenario: Order completion
- **WHEN** the last stage has delivered good parts >= planned_qty
- **THEN** the order status changes to CLOSED with a closed_at timestamp, leftover work in progress is carried over or scrapped as WIP, and the next queued order of the line is dispatched

#### Scenario: Order cancellation
- **WHEN** a user cancels an order via API
- **THEN** the order status changes to CANCELLED, its work in progress is counted as `wip_scrapped`, recipe overrides are reverted, and the next queued order is dispatched

### Requirement: MES dispatches operations per line stage
When a line has no IN_PROGRESS order, the MES SHALL dispatch the first RELEASED order of that line's queue (ordered by priority descending, then creation time ascending), creating one operation per machine stage. Operations are numbered sequentially (OP10, OP20, etc.). Dispatch SHALL be idempotent: an order is dispatched at most once.

#### Scenario: Create operations for a 4-stage line
- **WHEN** MES dispatches an order for `automotive-welding-1` (4 stages)
- **THEN** 4 operations are created: OP10 (stage 1), OP20 (stage 2), OP30 (stage 3), OP40 (stage 4)

#### Scenario: Busy line keeps the order queued
- **WHEN** an order is RELEASED for `automotive-welding-1` while another order is IN_PROGRESS on that line
- **THEN** the new order stays RELEASED and is dispatched after the running order closes or is cancelled

#### Scenario: No double dispatch
- **WHEN** release handling and the periodic dispatch loop run at the same time for the same order
- **THEN** exactly one set of operations is created and the order is assigned to the machines once

#### Scenario: Recipe overrides applied at dispatch
- **WHEN** MES dispatches an order for product `DOOR-PANEL-A`
- **THEN** MES calls `ApplyRecipeOverrides` on each machine in the line instance with the product's per-stage overrides

### Requirement: MES tracks order progress
MES SHALL monitor machine cycle completions and update operation quantities (good_qty, scrap_qty) in real time. An event SHALL be published when order progress changes.

#### Scenario: Operation quantity update
- **WHEN** a machine at stage 2 completes a cycle producing a good part
- **THEN** the corresponding operation's good_qty increments by 1

#### Scenario: Order progress event
- **WHEN** an operation's good_qty changes
- **THEN** an `order.progress` event is published with the order ID, line_instance_id, started_qty, current good_qty, scrap_qty, and planned_qty

## ADDED Requirements

### Requirement: Restore orders after restart
On startup the MES SHALL restore at most one IN_PROGRESS order per line instance: the one with the latest started_at. Other IN_PROGRESS orders of that line SHALL be set to RELEASED so they are queued, not lost.

#### Scenario: Several orders found in progress
- **WHEN** the database holds three IN_PROGRESS orders for one line at startup
- **THEN** the most recently started one is restored and the other two become RELEASED and are logged
