# Machine Simulator 2

A configurable manufacturing factory simulator that models production lines with realistic machine behavior, order management, and industrial protocol support (OPC-UA and Modbus TCP).

## Quick Start

```bash
# Docker (recommended)
docker compose up --build

# From source
go build -o simulator ./cmd/simulator
./simulator --profile demo-mixed
```

Access the simulator:
- **Web dashboard**: http://localhost:8081
- **REST API**: http://localhost:8081/api
- **OPC-UA**: ports 4840+ (one per machine)
- **Modbus TCP**: port 502

## Features

- **27 machine types** across 8 industrial domains
- **Production lines** with inter-stage buffers
- **Order management** — ERP auto-generates orders, MES dispatches to lines
- **OPC-UA servers** — one per machine with numeric node IDs (`ns=1;i=<NodeID>`)
- **Modbus TCP gateway** — single server mapping all machines to holding registers
- **REST API** — full control over lines, machines, orders, simulation parameters
- **WebSocket events** — real-time state changes and order progress
- **Webhook push** — configurable HTTP POST notifications to external systems
- **Web dashboard** — browser UI for monitoring and control

## Industrial Protocol Reference

The simulator exposes machine data via two industrial protocols. Both are read-only and updated on a configurable polling interval.

### Machine Addressing

With the default factory configuration (4 lines, 15 machines):

| Machine ID | Type | OPC-UA Port | Modbus Unit ID |
|-----------|------|-------------|----------------|
| automotive-assembly-1-stage-1 | robot_pick_place | 4840 | 1 |
| automotive-assembly-1-stage-2 | assembly_press | 4841 | 2 |
| automotive-assembly-1-stage-3 | spot_welder | 4842 | 3 |
| automotive-assembly-1-stage-4 | painting_booth | 4843 | 4 |
| automotive-welding-1-stage-1 | metal_forming | 4844 | 5 |
| automotive-welding-1-stage-2 | spot_welder | 4845 | 6 |
| automotive-welding-1-stage-3 | robot_welder | 4846 | 7 |
| automotive-welding-1-stage-4 | robot_pick_place | 4847 | 8 |
| electronics-smt-1-stage-1 | smt_placement | 4848 | 9 |
| electronics-smt-1-stage-2 | reflow_oven | 4849 | 10 |
| electronics-smt-1-stage-3 | aoi_inspection | 4850 | 11 |
| electronics-smt-1-stage-4 | labeling_machine | 4851 | 12 |
| electronics-through-hole-1-stage-1 | robot_pick_place | 4852 | 13 |
| electronics-through-hole-1-stage-2 | wave_solder | 4853 | 14 |
| electronics-through-hole-1-stage-3 | aoi_inspection | 4854 | 15 |

### Base Tags (all machines)

Every machine exposes these 6 tags regardless of type. Machine-type-specific tags (NodeID 7+) are listed in the [Tag Catalog](docs/tag-catalog.md).

| Tag | OPC-UA NodeID | Modbus Register | DataType | Description |
|-----|---------------|-----------------|----------|-------------|
| `state` | `ns=1;i=1` | 40001-40002 | String | Machine state (IDLE, RUNNING, STOPPED, FAULT, MAINTENANCE) |
| `cycle_count` | `ns=1;i=2` | 40003-40004 | Int64 | Total cycles completed |
| `good_count` | `ns=1;i=3` | 40005-40006 | Int64 | Good parts produced |
| `scrap_count` | `ns=1;i=4` | 40007-40008 | Int64 | Scrap parts produced |
| `cycle_time_ms` | `ns=1;i=5` | 40009-40010 | Int32 | Current cycle time in milliseconds |
| `blocked_by_buffer` | `ns=1;i=6` | 40011-40012 | Boolean | Whether machine is blocked by a full downstream buffer |

Machine-type tags start at NodeID 7 / register 40013. For example, a `robot_pick_place` machine has `joint1_angle` at `ns=1;i=7` (OPC-UA) or register 40013-40014 (Modbus). See the [Tag Catalog](docs/tag-catalog.md) for the complete list per machine type.

### OPC-UA

- **One server per machine**, each on its own TCP port (base port 4840, incrementing)
- Connect to: `opc.tcp://<host>:<port>`
- Tags are variable nodes under the Objects folder: `ns=1;i=<NodeID>`
- NodeIDs 1-6 are the base tags; 7+ are machine-type-specific
- Native data types (String, Double, Int64, Int32, Boolean) — no precision loss

### Modbus TCP

- **Single TCP gateway** on port 502 with all machines multiplexed via Unit ID
- Holding registers only (function code 0x03), read-only
- Each tag occupies **2 consecutive registers** (big-endian float32: high word first, low word second)
- All values are converted to float32 — this means:
  - **`state`**: always reads as 0.0 (string cannot be represented). Use OPC-UA or the REST API for state.
  - **Int64 counters**: truncated to float32 precision (~7 significant digits)
- The Modbus register formula: tag at OPC-UA NodeID _N_ maps to Modbus registers **40000 + (N\*2 - 1)** and **40000 + (N\*2)**
- The full runtime register map is available via `GET /api/modbus/register-map`

## Documentation

- [Project Overview](docs/README.md) — architecture, prerequisites, profiles, project structure
- [Configuration Guide](docs/configuration.md) — global settings, line templates, profiles, adapter configs
- [API Reference](docs/api-reference.md) — REST endpoints, WebSocket events, curl examples
- [ERP & MES System](docs/erp-mes-system.md) — order lifecycle, data structures, dispatch strategies, OEE inputs
- [Tag Catalog](docs/tag-catalog.md) — complete tag list for all 27 machine types with OPC-UA node IDs

## Prerequisites

- **Go 1.24+** (building from source)
- **Docker** and **Docker Compose** (containerized deployment)

## License

See [LICENSE](LICENSE) for details.
