# Line Jobs and Part Flow

Each line instance behaves like a line PLC: it runs one **job** at a time, releases parts at stage 1 up to the job's target, and accounts for every part it starts. An MES (the built-in one, or an external one over OPC UA or REST) loads and starts jobs.

## Order source

`global.yaml`:

```yaml
orders:
  source: internal   # internal | external   (env SIMULATOR_ORDERS_SOURCE)
```

| | `internal` (default) | `external` |
|---|---|---|
| Orders created by | built-in ERP (auto) or `POST /api/orders` | nobody: an outside ERP/MES owns orders |
| Jobs loaded by | built-in MES, one order per line, queue by priority then age | an outside MES via the line OPC UA server or `PUT /api/lines/{id}/job` |
| Order webhooks | sent | not sent (`order.*` removed from `webhook.event_types`) |
| `POST /api/orders` | creates an order | 409 |

## Job lifecycle

```
NO_JOB --load--> LOADED --start--> RUNNING --interrupt--> INTERRUPTED --start--> RUNNING
RUNNING --target reached and line drained--> TARGET_REACHED
any state with a job --finish--> NO_JOB
```

- **load**: allowed in `NO_JOB` or `LOADED` (replaces the loaded job). Applies the product recipe to every stage. Unknown products are rejected and the state does not change.
- **start**: starts or resumes. Starting a running job does nothing.
- **interrupt**: stage 1 stops releasing parts; parts already in the line finish and wait in buffers.
- **finish**: ends the job, also before the target (short close). The line stays as it is until the next load.
- `target_qty` 0 means open-ended: stage 1 keeps releasing until interrupt or finish.

## Part flow and ledger

- Stage 1 starts a part only while the job is `RUNNING` and `good + parts_in_process < target_qty`. Scrap downstream therefore causes exactly one more release per scrapped part.
- A machine works only while it holds a part. When its output buffer is full it keeps the finished part and is blocked; nothing is dropped.
- Every line keeps a ledger since start-up. It always balances:

  `started = delivered_good + scrapped + wip_scrapped + in_buffers + in_machines`

- **Leftover parts at the next load**: parts of the same product are carried over to the new job (`wip_carried_over`), up to what the job still needs. Parts of another product, and any excess, are removed and counted as `wip_scrapped` (changeover purge). Cancelling a running internal order scraps its work in progress.
- **Stall recovery**: if a running job shows no part movement for `deadlock_threshold_ms` (simulated time) while no machine is in `FAULT` or `MAINTENANCE`, idle or stopped machines that hold parts are restarted. Buffers are never cleared. Unset, the threshold is the longest `auto_restore_time_ms` plus 60 s, at least 120 s.

## Line OPC UA server

One server per line instance on `adapters.opcua.line_base_port + index` (default 4900, env `SIMULATOR_OPCUA_LINE_BASE_PORT`), namespace `urn:machine-simulator:line:<instance>`, addressed as `ns=1;i=<NodeID>`. Started only with `orders.source: external`.

| NodeID | Name | Type | Access | Meaning |
|---|---|---|---|---|
| 1 | `job_order_id` | String | RW | setpoint for the next load |
| 2 | `job_operation_id` | String | RW | setpoint |
| 3 | `job_product_id` | String | RW | setpoint |
| 4 | `job_target_qty` | Int32 | RW | setpoint, 0 = open-ended |
| 5 | `job_command` | Int32 | RW | 1 load, 2 start, 3 interrupt, 4 finish |
| 6 | `job_state` | String | R | `NO_JOB`, `LOADED`, `RUNNING`, `INTERRUPTED`, `TARGET_REACHED` |
| 7 | `job_command_result` | String | R | `OK`, `rejected: …` or `error: …` |
| 8 | `job_good` | Int32 | R | good parts delivered at the last stage |
| 9 | `job_scrap` | Int32 | R | scrapped parts of the job, all stages |
| 10 | `job_started` | Int32 | R | parts released at stage 1 |
| 11 | `parts_in_process` | Int32 | R | parts of the job in machines and buffers |
| 12 | `wip_scrapped` | Int32 | R | line total |
| 13–16 | `active_order_id`, `active_operation_id`, `active_product_id`, `active_target_qty` | | R | the job the line actually runs |

Rules for clients:

- Write the setpoints 1–4, then `job_command = 1`. The setpoints are copied at the moment of the command write, so a later setpoint write cannot change a queued load.
- Every write to `job_command` runs the command, also when the value is unchanged. The node keeps the written value, so a read-back check (the benthos-umh `opcua` output handshake) passes.
- Writes to read-only nodes return `BadUserAccessDenied`.
- Read the outcome from `job_command_result` and `job_state`.

benthos-umh example (load and start in two messages):

```yaml
output:
  opcua:
    endpoint: "opc.tcp://simulator:4900"
    securityMode: None
    securityPolicy: None
    nodeMappings:
      - {nodeId: "ns=1;i=1", valueFrom: "order", dataType: "String"}
      - {nodeId: "ns=1;i=2", valueFrom: "op", dataType: "String"}
      - {nodeId: "ns=1;i=3", valueFrom: "product", dataType: "String"}
      - {nodeId: "ns=1;i=4", valueFrom: "target", dataType: "Int32"}
      - {nodeId: "ns=1;i=5", valueFrom: "cmd", dataType: "Int32"}
```

## REST

| Method | Path | Body / result |
|---|---|---|
| `GET` | `/api/lines/{id}/job` | job status |
| `PUT` | `/api/lines/{id}/job` | `{"order_id","operation_id","product_id","target_qty"}` → job status |
| `POST` | `/api/lines/{id}/job/start` | job status |
| `POST` | `/api/lines/{id}/job/interrupt` | job status |
| `POST` | `/api/lines/{id}/job/finish` | `{"job": <now NO_JOB>, "finished": <final status>}` |

Errors: 404 unknown line, 409 illegal transition, 400 unknown product or invalid job. The body is `{"error": "...", "job": <current status>}`.

`GET /api/lines` and `GET /api/lines/{id}` include `job` and `ledger`; `GET /api/lines/{id}/buffers` includes `in_count` and `out_count`.

## Events

| Event | Data |
|---|---|
| `line.job_changed` | job status after every command and on `TARGET_REACHED` |
| `line.part_booked` | `{machine_id, order_id, stage, good, last_stage}` per finished cycle, attributed to the part's own order |
| `line.deadlock_recovered` | line ID |
