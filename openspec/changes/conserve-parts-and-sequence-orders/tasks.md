## 1. Tests first (failing)

- [x] 1.1 Engine test: run a 4-stage line through 5 orders with faults and downstream scrap; assert the conservation invariant (decision 1) holds after every order and at the end
- [x] 1.2 MES test: release two orders for the same line; assert only one is IN_PROGRESS and the second starts after the first closes
- [x] 1.3 MES test: concurrent `releaseAndDispatch` and `dispatchLoop` never dispatch the same order twice (run with `-race`)
- [x] 1.4 Restore test: three IN_PROGRESS orders on one line in the DB at startup → newest restored, two RELEASED
- [x] 1.5 Deadlock test: a stage FAULT lasting `auto_restore_time_ms` does not trigger recovery; a true stall does, and stage 1 restarts
- [x] 1.6 Tag test: base tags 1001–1005 exist on every machine type with correct values for first, middle and last stage

## 2. Line ledger and buffers (`internal/line`)

- [x] 2.1 Add per-line ledger counters (started, delivered_good, scrapped per stage, wip_carried_over, wip_scrapped) and `PartsInProcess(orderID)`
- [x] 2.2 Make `HandlePartComplete` check the `Add` result and never drop a part (block upstream instead)
- [x] 2.3 Remove `Buffer.Clear()` calls from deadlock recovery; add `ScrapWIP(lineID, reason)` and `CarryOverWIP(lineID, newOrderID)`
- [x] 2.4 Stage-1 gating: `CanStartPart(lineID)` = delivered_good + parts_in_process < planned_qty; machines call it before starting a cycle at stage 1
- [x] 2.5 Track `starved` per machine (set when waiting on an empty in-buffer)

## 3. MES order handling (`internal/mes`)

- [x] 3.1 Per-line mutex and a single `tryDispatch(lineID)` used by release, completion and `dispatchLoop`
- [x] 3.2 Queue order: priority DESC, created_at ASC; skip lines with an IN_PROGRESS order
- [x] 3.3 `dispatchOrder`: no buffer clearing, no machine reset of in-flight parts; carry over or scrap leftover WIP per decision 3
- [x] 3.4 `completeOrder` / `CancelOrder`: account for remaining WIP instead of clearing
- [x] 3.5 `restoreInProgressOrders`: keep newest per line, re-queue others, log them
- [x] 3.6 `order.progress` event and webhook payload: add `line_instance_id`, `started_qty`

## 4. Deadlock detection (`internal/line`, `internal/config`)

- [x] 4.1 Add `deadlock_threshold_ms` to global config (default 120000); validate it exceeds the max `auto_restore_time_ms` at startup
- [x] 4.2 Exclude lines with a FAULT machine or one inside its auto-restore window; use ledger change as the progress signal
- [x] 4.3 Recovery: START stopped machines including stage 1; publish `line.deadlock_recovered`

## 5. Tags and protocols (`internal/machine`, `internal/protocol`)

- [x] 5.1 Add flow tags 1001–1005 (`FlowTags()`, values in `BaseTagValues()`), appended last by `AllTagDefinitions()`
- [x] 5.2 OPC UA and Modbus adapters use `AllTagDefinitions()` (Modbus appends after existing registers; checked by test, not against a Modbus client)

## 6. API, config, storage

- [x] 6.1 `GET /api/lines/{id}` and `/buffers`: add ledger counters, in/out counts per buffer
- [x] 6.2 `POST /api/orders`: validate line instance exists and product is in its recipes (400 with valid products otherwise)
- [x] 6.3 `SIMULATOR_DB_PATH` env (default `./data/simulator.db`); document in `docs/`
- [x] 6.4 Update `docs/` tag catalog (nodes 1001–1005) and API reference

## 7. Verification

- [x] 7.1 `go test -mod=vendor -race ./internal/...` green
- [ ] 7.2 Run the four demo lines for 2 h at time_scale 10: per line, first-stage good minus last-stage cycles equals parts in process + scrapped downstream + wip_scrapped (report the numbers) — partial: 2 min at time_scale 20 on the demo-mixed profile, ledger balanced on all 4 lines, max 1 IN_PROGRESS per line, 8/8 closed orders good = planned
- [ ] 7.3 Deploy to the sales-demo host behind a tag; confirm in the UMH demo: at most one IN_PROGRESS order per line, buffer tags arriving, Plant pages unchanged or better
