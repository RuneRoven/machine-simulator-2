## ADDED Requirements

### Requirement: Order source mode
The simulator SHALL support `orders.source` with values `internal` (default) and `external`. In `external` mode the built-in ERP SHALL NOT create orders, the built-in MES SHALL NOT dispatch or restore orders, and order webhooks SHALL NOT be sent.

#### Scenario: Default keeps today's behaviour
- **WHEN** `orders.source` is not set
- **THEN** the built-in ERP and MES run as before

#### Scenario: External mode leaves lines idle
- **WHEN** the simulator starts with `orders.source: external`
- **THEN** no orders are created and every line stays idle with job state NO_JOB

#### Scenario: Order creation refused in external mode
- **WHEN** `POST /api/orders` is called in external mode
- **THEN** the response is 409 and names the line job API

### Requirement: Line job lifecycle
Each line instance SHALL hold at most one job `{order_id, operation_id, product_id, target_qty}` with states NO_JOB, LOADED, RUNNING, INTERRUPTED and TARGET_REACHED, changed only by the commands load, start, interrupt and finish. Illegal transitions SHALL be rejected with a reason and leave the state unchanged.

#### Scenario: Load applies the recipe
- **WHEN** a job for product `DOOR-PANEL-A` is loaded on `automotive-welding-1`
- **THEN** the job state is LOADED and the product's recipe overrides are applied to each stage

#### Scenario: Unknown product rejected
- **WHEN** a job is loaded with a product that is not in the line's recipes
- **THEN** the command is rejected, the result names the valid products, and the state is unchanged

#### Scenario: Start without a job rejected
- **WHEN** start is sent while the state is NO_JOB
- **THEN** the command is rejected and nothing starts

#### Scenario: Interrupt and resume
- **WHEN** a RUNNING job is interrupted and later started again
- **THEN** stage 1 starts no parts while INTERRUPTED, parts in process finish into buffers, and production continues from the same counters after start

### Requirement: Stage-1 release by job
Stage 1 SHALL start a part only while the job is RUNNING and, when `target_qty > 0`, `job_good + parts_in_process < target_qty`. When the target is met and no parts of the job remain in process, the job state SHALL become TARGET_REACHED and stay there until finish.

#### Scenario: Batch counter stops the line
- **WHEN** a job with target 50 has 48 good and 2 in process
- **THEN** stage 1 starts no further part, and after both parts are delivered good the state is TARGET_REACHED

#### Scenario: Open-ended job
- **WHEN** a job has `target_qty` 0
- **THEN** stage 1 keeps starting parts until interrupt or finish

### Requirement: Finish accounts for leftover parts
On finish, the job's parts still in the line SHALL be kept for a following job of the same product (`wip_carried_over`) or counted as `wip_scrapped` when a job for another product is loaded.

#### Scenario: Short close
- **WHEN** a job with target 100 is finished at 80 good with 4 parts in process and the next job is another product
- **THEN** the 4 parts are counted as `wip_scrapped` when the next job is loaded and the ledger balances
