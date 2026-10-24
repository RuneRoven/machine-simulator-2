## Context

Target architecture (sales demo):

```
ERP sim (SAP OData, :8100) ⇄ UMH http_client ⇄ MES sim (Hydra-style REST, :8200)
                                   │                       ▲
                     UNS / DB → operator dashboard → operator action (outbox)
                                   │
                     UMH opcua output (write job) ──► machine-simulator-2 line server
                     UMH opcua input (read tags)  ◄── machine + line servers
```

The operator, not the MES, starts work: the dashboard shows the MES work list per workplace, the operator starts an operation, UMH books the start in the MES and loads the job onto the line. When the line reaches the target, the operator finishes the operation and UMH books the quantities in the MES; the MES confirmation goes to the ERP through UMH. machine-simulator-2 only has to behave like line PLCs.

## Goals / Non-Goals

**Goals:** lines run only on an externally loaded and started job; job state and counters readable and writable over OPC UA and REST; the built-in ERP/MES can be switched off without changing machine tags.

**Non-Goals:** the ERP and MES simulators themselves (new repository), operator logins and booking logic (MES), shift calendar and setup times (phase 1 change; a SETUP job state is reserved for it).

## Decisions

### 1. Order source switch, default unchanged
`orders.source: internal | external`. In `external` mode the app does not start `erp.autoOrderLoop`, `mes.dispatchLoop`, order restore or the order webhook. Order endpoints stay readable but `POST /api/orders` returns 409 with a hint to use the job API. Default `internal` keeps standalone users unaffected.

### 2. Line control moves to `internal/line`
`handleCycleComplete`, `tryStartDownstream` and buffer pulls are line physics, not MES logic. They move to `line.Controller`, subscribed to `machine.cycle_complete`. The built-in MES keeps order bookkeeping and calls the controller to load/start jobs, so both modes use the same line code path.

### 3. Job state machine
```
NO_JOB --load--> LOADED --start--> RUNNING --interrupt--> INTERRUPTED --start--> RUNNING
RUNNING --target reached & drained--> TARGET_REACHED
LOADED|RUNNING|INTERRUPTED|TARGET_REACHED --finish--> NO_JOB
```
- `load` is only allowed in NO_JOB (or LOADED, to replace). Loading a different product while parts of another product are in the line marks them for `wip_scrapped` at start (changeover purge).
- `interrupt` stops stage 1 from starting parts; parts in process finish and park in buffers (realistic pause, keeps the ledger simple).
- `finish` before the target is allowed (operator closes short); remaining WIP follows the ledger rules.
- `job_good`/`job_scrap` count last-stage good and all-stage scrap for the current job; `job_started` counts stage-1 starts.

### 4. OPC UA line server
One server per line instance on `line_base_port + i` (default 4900), namespace `urn:machine-simulator:line:<instance>`. NodeIDs:

| NodeID | Name | Type | Access |
|---|---|---|---|
| 1 | job_order_id | String | RW |
| 2 | job_operation_id | String | RW |
| 3 | job_product_id | String | RW |
| 4 | job_target_qty | Int32 | RW |
| 5 | job_command | Int32 | RW (1 load, 2 start, 3 interrupt, 4 finish; keeps the written value, every write is handled) |
| 6 | job_state | String | R |
| 7 | job_command_result | String | R (`OK`, or the rejection reason) |
| 8 | job_good | Int32 | R |
| 9 | job_scrap | Int32 | R |
| 10 | job_started | Int32 | R |
| 11 | parts_in_process | Int32 | R |
| 12 | wip_scrapped | Int32 | R |
| 13–16 | active_order_id, active_operation_id, active_product_id, active_target_qty | | R (the job the line actually runs; 1–4 are only setpoints) |

As built: gopcua has no write callback and does not synchronise node values. The line server wraps its `NodeNameSpace` and swaps the wrapper into the server's namespace slice (`srv.Namespaces()` returns the server's own slice); the wrapper's `SetAttribute` serialises client writes with the server's own updates and records every accepted write. A `job_command` write snapshots the setpoints and is queued (capacity 64) for a handler goroutine. Read-only nodes have AccessLevel CurrentRead, so client writes get `BadUserAccessDenied` from the library. Parameters are latched only on `load`, so a client writes 1–4 then `job_command=1`, then `job_command=2`. `job_command` is not reset after handling: benthos-umh's opcua output verifies writes by read-back (`handshake`, `opcua_plugin/write.go`) and a reset would fail it. Each write is handled, so the same command twice (two loads) works. Alternative considered: OPC UA methods (`LoadJob(...)`) — cleaner, but the opcua output writes variables, not methods [Certain, from `write.go` fields nodeId/dataType].

### 5. REST
`GET /api/lines/{id}/job`, `PUT /api/lines/{id}/job` (load), `POST /api/lines/{id}/job/start|interrupt|finish`. Same controller as OPC UA, same rejection reasons (409 for illegal transitions, 400 for unknown product).

## Risks / Trade-offs

- [Two write paths race] OPC UA and REST hit the same controller under one mutex → last command wins, both return the resulting state.
- [Silent writes today] Clients that write machine nodes today see no effect; this change does not alter machine nodes.
- [Idle lines in the demo] In external mode nothing runs unless UMH and the operator bot load jobs → the ERP/MES simulator ships an operator bot.

## Open Questions

1. Modbus mirror of the line server: skip unless a customer case needs it.
