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
