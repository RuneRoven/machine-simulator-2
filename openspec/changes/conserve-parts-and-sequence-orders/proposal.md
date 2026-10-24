## Why

The simulated plant does not behave like a real one, and downstream systems notice. In the UMH sales demo, the first station's good-part counter exceeds the last station's cycle counter by about 1,000 parts per day per line, several orders sit IN_PROGRESS on the same line at once (some for 16+ hours), and lines stall after ordinary faults. Root causes in the code:

- `Buffer.Clear()` discards parts without accounting in `mes.dispatchOrder`, `completeOrder`, `CancelOrder` and `line.DetectAndRecoverDeadlock`.
- Stage 1 produces from nothing, with no limit tied to the order quantity.
- `mes.dispatchOrder` never checks whether the line is busy: a new order takes over the machines and the previous order stays IN_PROGRESS forever. `releaseAndDispatch` and `dispatchLoop` can dispatch the same order twice; `restoreInProgressOrders` re-assigns every IN_PROGRESS order after a restart.
- Deadlock detection (10 strikes × 5 s = 50 s) fires during normal faults (`auto_restore_time_ms` 45–60 s), clears buffers, and resets machines to IDLE; nothing restarts stage 1.
- Buffer levels exist internally but are not published, so a client cannot measure WIP.
- `simulator.db` lives in `/app`, so orders are lost on container rebuild.

This change makes material flow and order handling correct. It is phase 0 of a larger plan (shift calendar, setup states, quality from process values, serials, and an external SAP-style ERP simulator that feeds orders through UMH), and it is the precondition for all of it.

## What Changes

- Parts are conserved: buffers are never emptied silently. Parts left over when an order ends are carried to the next order of the same product on that line, or counted as `wip_scrapped` when the product changes or the order is cancelled.
- Stage 1 starts a part only while `good at last stage + parts in process < planned_qty`, so each order releases exactly what it needs (re-starting as downstream scrap occurs).
- **BREAKING (behaviour)**: one order runs per line at a time. Released orders wait in a per-line queue (by priority, then creation time) and are dispatched only when the line has no IN_PROGRESS order. Double dispatch is removed. On startup, only the most recently started IN_PROGRESS order per line is restored; any others go back to RELEASED.
- Deadlock detection ignores lines with a FAULT machine or a machine still inside its auto-restore window; the threshold becomes configurable and defaults above the longest auto-restore time. Recovery restarts the line (stage 1 included) instead of clearing buffers.
- Buffer state is published as machine tags in a new reserved base range (NodeIDs 1001+): `starved`, `in_buffer_level`, `in_buffer_capacity`, `out_buffer_level`, `out_buffer_capacity`. Existing NodeIDs 1–6 and type-specific IDs (7+) do not change.
- Line-level counters `wip_scrapped`, `wip_carried_over` and `parts_in_process` are exposed in `GET /api/lines/{id}` and the buffer endpoint (with in/out counts).
- The order webhook and `order.progress` event gain `line_instance_id` and `started_qty`.
- `simulator.db` path is configurable (`SIMULATOR_DB_PATH`, default `./data/simulator.db`) so it survives container rebuilds.
- `POST /api/orders` validates line and product as the spec already requires (the code currently does not).

## Capabilities

### New Capabilities
- (none)

### Modified Capabilities
- `simulation-engine`: FIFO buffers conserve parts; stage-1 release limited by the order; deadlock detection and recovery redefined.
- `erp-mes-system`: one order per line with a queue; dispatch idempotent; restore on restart; progress event fields; completion and cancellation account for remaining WIP.
- `machine-types`: reserved base tag range 1001+ for buffer and flow tags.
- `rest-api`: line and buffer endpoints expose conservation counters; order creation validated.
- `config-system`: database path and deadlock threshold settings.

## Impact

- **Code**: `internal/mes/mes.go` (dispatch, queue, completion, cancel, restore), `internal/line/line.go` (buffer accounting, deadlock detection, stage-1 gating, counters), `internal/machine/machine.go` and `tags.go` (starved flag, base tags 1001+), `internal/app/app.go` (DB path), `internal/config/types.go` + `config/global.yaml` (new settings), `internal/api/handlers.go` (validation, line counters), `internal/webhook/webhook.go` (payload fields).
- **APIs**: additive fields on `GET /api/lines/{id}`, `/api/lines/{id}/buffers`, order objects and webhooks; `POST /api/orders` may now return 400 for unknown line/product.
- **Protocol clients**: new OPC UA / Modbus nodes 1001–1005 on every machine; existing nodes unchanged.
- **Behaviour**: throughput per line is limited to one order at a time; the ERP backlog fills up instead of preempting. Downstream dashboards (UMH demo) will see fewer, non-overlapping IN_PROGRESS orders and counters that balance.
- **Docs**: `docs/` tag catalog and API reference updated.
