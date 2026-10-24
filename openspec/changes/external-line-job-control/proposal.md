## Why

In a real plant the machine does not own the order book. Orders come from the ERP, the MES turns them into operations per workplace, an operator sees the work list on a terminal, sets up the machine and starts the operation, and the MES reports quantities back to the ERP. machine-simulator-2 currently plays ERP, MES and machine at once: it invents orders, dispatches them to lines and reports them by webhook. That makes it impossible to demonstrate the integration layer (UMH) doing what it does at customers: reading and writing ERP and MES systems over their APIs and connecting them to the shop floor.

The ERP and MES move to a separate simulator (SAP-style OData ERP, MPDV-Hydra-style MES, new repository). This change turns machine-simulator-2 into the shop-floor layer only: lines that run a job when one is loaded and started from outside, and that report per-job counters over OPC UA and REST.

Depends on `conserve-parts-and-sequence-orders` (part ledger, stage-1 gating, no buffer clearing).

## What Changes

- New global setting `orders.source`: `internal` (today's built-in ERP + MES, default, unchanged behaviour) or `external` (built-in ERP auto-ordering, MES dispatch and order webhooks are off; lines idle until a job is loaded).
- **Line job**: each line instance has at most one job `{order_id, operation_id, product_id, target_qty}` and a job state machine `NO_JOB → LOADED → RUNNING ⇄ INTERRUPTED → TARGET_REACHED → (FINISH) NO_JOB`. Loading a job applies the product recipe; unknown products are rejected.
- Stage 1 starts parts only while the job is RUNNING and (if `target_qty > 0`) `delivered_good + parts_in_process < target_qty` (the PLC batch counter). At the target the line drains and the job goes to TARGET_REACHED; it stays there until FINISH, as on a real machine waiting for the operator to report.
- FINISH and INTERRUPT handle leftover parts with the ledger rules of the dependency change (carry over to a next job of the same product, otherwise `wip_scrapped` when the next job loads).
- **Line OPC UA server** ("line PLC"), one per line instance on its own port range: writable job nodes (`job_order_id`, `job_operation_id`, `job_product_id`, `job_target_qty`, `job_command`) and read-only nodes (`job_state`, `job_good`, `job_scrap`, `job_started`, `job_command_result`, line ledger counters). Writes are applied through a namespace write hook; today writes are accepted but overwritten on the next refresh.
- REST equivalents: `GET/PUT /api/lines/{id}/job`, `POST /api/lines/{id}/job/{start|interrupt|finish}`.
- Line control (part routing, starting downstream stages) moves out of `internal/mes` into `internal/line`, so lines run without the built-in MES.

## Capabilities

### New Capabilities
- `line-job-control`: job model, state machine, commands, stage-1 gating by job, external order-source mode.

### Modified Capabilities
- `protocol-adapters`: line-level OPC UA server with writable job nodes.
- `rest-api`: line job endpoints.

## Impact

- **Code**: new `internal/line/job.go`, line control moved from `internal/mes/mes.go` to `internal/line`, `internal/protocol/opcua` (line server + write hook), `internal/app/app.go` (wire by `orders.source`), `internal/config` (`orders.source`, `opcua.line_base_port`), `internal/api/handlers.go`.
- **Clients**: in `external` mode no order webhooks are sent; order data must come from the ERP/MES simulator through UMH. Machine servers and NodeIDs are unchanged. New line servers on `line_base_port` (default 4900).
- **Default behaviour**: unchanged (`orders.source: internal`).
