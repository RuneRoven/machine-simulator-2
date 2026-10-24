## Context

machine-simulator-2 runs one goroutine per machine (100 ms ticker); stage completions flow synchronously through the event bus (`machine.cycle_complete` → `mes.handleCycleComplete` → `line.Manager.HandlePartComplete` / `TryPullFromBuffer`). `line.Buffer` is a FIFO of `*Part{ID, OrderID, OperationID, IsGood}`. Orders are created by the ERP (auto every 30 s, or `POST /api/orders`) and dispatched by the MES, which today resets the whole line on every dispatch.

Consumers (UMH umh-core flows in the sales demo) read machine tags over OPC UA and order events over webhooks. They compute OEE, WIP and order progress from these, so any part that disappears or any overlapping order shows up as wrong numbers on customer-facing dashboards.

## Goals / Non-Goals

**Goals:**
- Every part that enters a line is accounted for: it is in a buffer, in a machine, delivered as good at the last stage, scrapped at a stage, carried over, or scrapped as WIP.
- At most one IN_PROGRESS order per line; released orders queue.
- No buffer clearing as a side effect of dispatch, completion or fault recovery.
- Buffer levels and starved state are readable over OPC UA / Modbus without breaking existing node IDs.
- State survives a container rebuild.

**Non-Goals:**
- Shift calendar, setup/changeover, maintenance windows, quality from process values, serials, material lots (phase 1, separate change).
- External ERP integration, due dates, customers (phase 2/3, separate changes; this change only keeps `POST /api/orders` usable for an external dispatcher).
- Preemption / rush orders that interrupt a running order (later; needs an explicit INTERRUPTED state).

## Decisions

### 1. Conservation ledger per line
Keep per-line counters in `line.Manager`: `started` (stage-1 starts for the current order), `delivered_good` (last stage good), `scrapped` (per stage), `wip_carried_over`, `wip_scrapped`. Invariant checked in tests and logged when violated: `started + carried_in = delivered_good + scrapped + in_buffers + in_machines + carried_out + wip_scrapped`. Machine counters (`cycle_count`, `good_count`, `scrap_count`) stay cumulative and monotonic for OPC UA clients.

### 2. Stage-1 release limited by demand
Stage 1 may start a cycle only while `delivered_good + parts_in_process < planned_qty`, where `parts_in_process` counts parts in buffers and machines that belong to the order. Downstream scrap therefore triggers exactly as many re-starts as needed. Alternative considered: a fixed scrap allowance at release — rejected, it either over-produces or leaves the order short.

### 3. Leftover WIP at order end
When the last stage reaches `planned_qty` the order closes. Parts still in the line belong to that order; they are re-tagged to the next queued order if it is for the same product (`wip_carried_over`), otherwise the line drains them as scrap at dispatch of the next product (`wip_scrapped`, counted at the stage where they sit). With decision 2 the leftover is at most the parts in process when the last good part arrives, typically a few per buffer. Cancelling an order scraps its WIP the same way.

### 4. One order per line, queue in the MES
`dispatchOrder` is only called for lines with no IN_PROGRESS order. The queue is the RELEASED orders of the line ordered by `priority DESC, created_at ASC` (due date added by the ERP change later). `releaseAndDispatch` only releases; dispatch happens in one place (on release, on order completion, and in `dispatchLoop` as a safety net) under a per-line mutex, making it idempotent. The ERP backlog limit keeps the queue bounded.

### 5. Startup restore
`restoreInProgressOrders` keeps, per line, the IN_PROGRESS order with the latest `started_at` and sets any others to RELEASED (they re-queue; they are not lost). A one-off log line lists the orders that were re-queued.

### 6. Deadlock detection redefined
A line is stalled only if: its job is RUNNING or INTERRUPTED, there is work, no machine is FAULT or MAINTENANCE, and no part has moved for `deadlock_threshold_ms` (unset: longest `auto_restore_time_ms` + 60 s, at least 120 s; an explicit value must exceed every `auto_restore_time_ms`, validated at startup). A real pharma machine restores after 120 s, which is why a fixed 120 s default failed validation. Recovery sends START to stopped machines including stage 1; buffers are never cleared. A `line.deadlock_recovered` event is still published.

### 7. Tags for buffers in a reserved range (as built: `FlowTags()`, appended last by `AllTagDefinitions()` so Modbus addresses do not move)
NodeIDs 1–6 are base tags and 7+ are type-specific, so adding base tags at 7 would renumber every machine type and break clients. New base tags use the reserved range starting at 1001: `starved` (1001, Boolean), `in_buffer_level` (1002, Int32), `in_buffer_capacity` (1003, Int32), `out_buffer_level` (1004, Int32), `out_buffer_capacity` (1005, Int32). Stage 1 has no in-buffer and the last stage no out-buffer: their values are 0 / capacity 0. Alternative considered: a separate OPC UA server per line with buffer nodes — cleaner model but a second addressing scheme for clients; can be added later.

### 8. Database path
`storage.NewDB` uses `SIMULATOR_DB_PATH` (default `./data/simulator.db`); `docker-compose.yaml` already mounts `./data`.

### 9. As built: one flow per line owns every part
`internal/line/flow.go` holds, per line, the part each machine is working on, finished parts waiting for space (`held`), the buffers, the job and the ledger, all under one mutex. A machine only cycles while it holds a part (`starved` otherwise) and is `blocked` only while it holds a finished part it cannot place. The built-in MES no longer routes parts: it loads one line job per order, counts from `line.part_booked` events (each part carries its own order ID) and copies order quantities from the job, so concurrent machine events cannot lose updates. Carry-over is capped at what the next job needs; the excess is purged as WIP scrap.

## Risks / Trade-offs

- [Lower throughput] One order at a time means a line idles between orders while the next order's stage 1 starts → acceptable and realistic; the ERP interval/backlog can be tuned.
- [Existing deployments] Clients that relied on overlapping IN_PROGRESS orders (none known; the UMH demo works around them) → re-queue on restore avoids losing orders.
- [Behaviour change visible on dashboards] OEE and order figures in the UMH demo will shift (fewer open orders, balanced counters) → announce with the release.
- [Ledger overhead] Per-cycle bookkeeping under the line mutex → trivial at 100 ms ticks.

## Open Questions

1. Leftover WIP for a different product: scrap it (realistic changeover purge, current proposal) or let the line finish it into finished goods without an order?
2. Should stage 1 also stop at end of shift once the shift calendar exists (phase 1), leaving WIP in buffers overnight? Proposed: yes, but out of scope here.
3. NodeID range 1001+ for new base tags: acceptable for UMH bridge templates, or prefer a separate line server?
