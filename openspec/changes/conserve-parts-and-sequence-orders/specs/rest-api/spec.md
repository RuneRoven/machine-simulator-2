## MODIFIED Requirements

### Requirement: Line instance endpoints
The REST API SHALL provide endpoints for listing and inspecting active line instances, including the line's part ledger.

#### Scenario: List all active line instances
- **WHEN** `GET /api/lines`
- **THEN** response includes all active line instances with their name, template, domain, machine count, buffer count, and current status

#### Scenario: Get line instance detail
- **WHEN** `GET /api/lines/{instanceId}`
- **THEN** response includes the full line state: machine IDs and states, buffer IDs with fill levels, active order (if any), queued orders, active product, available recipes, and the ledger (`started`, `delivered_good`, `scrapped`, `parts_in_process`, `wip_carried_over`, `wip_scrapped`)

#### Scenario: List line buffers
- **WHEN** `GET /api/lines/{instanceId}/buffers`
- **THEN** response includes each buffer with its upstream/downstream machine IDs, current count, max capacity, in-count, out-count and throughput counter

#### Scenario: Start all machines on a line
- **WHEN** `POST /api/lines/{instanceId}/start`
- **THEN** all machines on that line instance receive a START command

#### Scenario: Stop all machines on a line
- **WHEN** `POST /api/lines/{instanceId}/stop`
- **THEN** all machines on that line instance receive a STOP command

### Requirement: Order endpoints
The REST API SHALL provide CRUD operations for orders with filtering capabilities.

#### Scenario: List orders with filters
- **WHEN** `GET /api/orders?status=IN_PROGRESS&line_instance=automotive-welding-1`
- **THEN** response includes only orders matching both filters, with pagination support (limit/offset)

#### Scenario: Create order
- **WHEN** `POST /api/orders` with `{ "line_instance_id": "...", "product_id": "...", "planned_qty": 100 }`
- **THEN** the order is created, validated (line instance must exist, product must be in the line's recipe list, planned_qty > 0), and returned with status CREATED

#### Scenario: Create order with invalid product
- **WHEN** `POST /api/orders` specifies a product_id not in the target line's recipe list
- **THEN** response is 400 with an error message listing valid products for that line

#### Scenario: Create order for unknown line
- **WHEN** `POST /api/orders` specifies a line_instance_id that is not active
- **THEN** response is 400 with an error message listing the active line instances

#### Scenario: Get order detail
- **WHEN** `GET /api/orders/{id}`
- **THEN** response includes the order with all fields plus its operations list and started_qty

#### Scenario: Cancel order
- **WHEN** `DELETE /api/orders/{id}` (or `PUT /api/orders/{id}/status` with `CANCELLED`)
- **THEN** the order is cancelled if not already CLOSED, and its work in progress is counted as `wip_scrapped`
