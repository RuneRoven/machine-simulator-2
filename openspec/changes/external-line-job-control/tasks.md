## 1. Tests first (failing)

- [x] 1.1 Job controller: state machine table test (every command × every state → new state or rejection)
- [x] 1.2 Stage-1 gating: target 20 → exactly 20 good delivered, TARGET_REACHED, ledger balances (with downstream scrap)
- [x] 1.3 Interrupt/resume: no stage-1 starts while INTERRUPTED; counters continue after start
- [x] 1.4 Product change after short finish: leftover counted as `wip_scrapped` at next load
- [x] 1.5 External mode: no orders created and no webhook sent — checked in a smoke run (409 on `POST /api/orders`, ERP/MES not started, `order.*` filtered from webhook types), no unit test
- [x] 1.6 OPC UA: client writes nodes 1–5 → job RUNNING; write to node 8 → BadUserAccessDenied (through the namespace write path the Write service uses; the gopcua client package is not vendored)

## 2. Line control refactor

- [x] 2.1 Move `handleCycleComplete` part routing and `tryStartDownstream` from `internal/mes` to `line.Controller`
- [x] 2.2 Built-in MES calls the controller (load/start/finish) instead of touching machines and buffers directly; existing MES tests stay green

## 3. Job controller (`internal/line/flow.go`)

- [x] 3.1 Job struct, states, commands, per-line mutex, rejection reasons
- [x] 3.2 Recipe apply on load; changeover purge marking
- [x] 3.3 Stage-1 gate uses job state and target; TARGET_REACHED after drain
- [x] 3.4 Publish `line.job_changed` events (for WebSocket UI)

## 4. Protocols and API

- [x] 4.1 OPC UA line server per instance (`opcua.line_base_port`, default 4900) with nodes 1–16
- [x] 4.2 Namespace wrapper intercepting `SetAttribute` for nodes 1–5
- [x] 4.3 REST job endpoints; `POST /api/orders` returns 409 in external mode
- [x] 4.4 Check benthos-umh `opcua_plugin/write.go`: data types and write semantics match (String, Int32)

## 5. Config and wiring

- [x] 5.1 `orders.source` in global config + env override; validation
- [x] 5.2 `app.go`: skip ERP loop, MES dispatch/restore and webhook in external mode
- [x] 5.3 docker-compose: expose 4900–4903

## 6. Verification

- [x] 6.1 `go test -mod=vendor -race ./internal/...` green
- [x] 6.2 Manual: drove `automotive-welding-1` through load → start → TARGET_REACHED (5 good, 1 scrap) with the benthos-umh `opcua` output, handshake on (local benthos-umh build of 2026-08-25)
