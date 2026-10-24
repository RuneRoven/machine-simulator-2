# Machine Simulator 2

A configurable manufacturing factory simulator that models production lines with realistic machine behavior, order management, and industrial protocol support (OPC-UA and Modbus TCP).

## Overview

Machine Simulator 2 creates virtual factory environments with:

- **27 machine types** across 8 industrial domains (automotive, electronics, food & beverage, pharma, window manufacturing, furniture, metal parts, plastic parts)
- **Production lines** — sequential machines connected by inter-stage buffers
- **Order management** — ERP auto-generates orders, MES dispatches them to lines
- **OPC-UA servers** — one per machine with numeric node IDs (`ns=1;i=<NodeID>`)
- **Modbus TCP gateway** — single server mapping all machines to unit IDs and holding registers
- **REST API** — full control over lines, machines, orders, simulation parameters
- **WebSocket events** — real-time state changes, cycle completions, order progress
- **Webhook push** — configurable HTTP POST notifications to external systems (Power BI, data pipelines)
- **Web dashboard** — browser UI for monitoring and control

## Prerequisites

- **Go 1.24+** (for building from source)
- **Docker** and **Docker Compose** (for containerized deployment)

## Quick Start

### Docker (recommended)

```bash
docker compose up --build
```

The simulator starts with the `demo-mixed` profile (one line each from automotive, electronics, food & beverage, and pharma domains).

Access:
- **Web dashboard**: http://localhost:8081
- **REST API**: http://localhost:8081/api
- **OPC-UA**: ports 4840+ (one per machine)
- **Modbus TCP**: port 502

### From Source

```bash
go build -o simulator ./cmd/simulator
./simulator --profile demo-mixed
```

## Selecting a Profile

Profiles define which production lines to instantiate. Three built-in profiles are included:

| Profile | Description | Lines |
|---------|-------------|-------|
| `demo-mixed` | One line per domain (default) | automotive-welding, electronics-smt, food-beverage-filling, pharma-batch |
| `automotive-focus` | Automotive-heavy factory | 2x automotive-welding, automotive-assembly, metal-parts-fabrication |
| `full-catalog` | All 10 line templates | One instance of every line template |

Set the profile via command line or environment variable:

```bash
# Command line
./simulator --profile full-catalog

# Environment variable
SIMULATOR_PROFILE=full-catalog docker compose up

# Docker Compose
SIMULATOR_PROFILE=automotive-focus docker compose up --build
```

## Project Structure

```
machine-simulator-2/
├── cmd/simulator/          # Application entry point
├── internal/
│   ├── api/                # REST API handlers + WebSocket
│   ├── app/                # Application wiring
│   ├── config/             # Configuration loading
│   ├── erp/                # ERP simulator (auto order generation)
│   ├── event/              # Event bus
│   ├── line/               # Production line management
│   ├── machine/            # Machine interface + base implementation
│   │   └── types/          # 27 machine type implementations
│   ├── mes/                # MES simulator (order dispatch)
│   ├── protocol/           # Protocol adapters
│   │   ├── opcua/          # OPC-UA adapter
│   │   └── modbus/         # Modbus TCP adapter
│   ├── storage/            # SQLite storage for orders
│   └── web/                # Web dashboard
├── config/
│   ├── global.yaml         # Global settings
│   ├── adapters/           # OPC-UA and Modbus config
│   ├── lines/              # Line templates by domain
│   └── profiles/           # Factory profiles
└── docs/                   # Documentation
```

## Documentation

- [Configuration Guide](configuration.md) — global settings, line templates, profiles, adapter configs
- [API Reference](api-reference.md) — REST endpoints, WebSocket events, curl examples
- [ERP & MES System](erp-mes-system.md) — order lifecycle, data structures, dispatch strategies, OEE inputs
- [Line Jobs and Part Flow](line-job-control.md) — job lifecycle, part ledger, external order source, line OPC UA server
- [Tag Catalog](tag-catalog.md) — complete tag list for all 27 machine types with OPC-UA node IDs

## Key Concepts

### Machine States

Every machine operates in one of five states:

| State | Description |
|-------|-------------|
| `IDLE` | Ready, waiting for work |
| `RUNNING` | Actively processing a cycle |
| `STOPPED` | Manually stopped |
| `FAULT` | Breakdown (auto-restores after configured time) |
| `MAINTENANCE` | In maintenance mode |

### Tags

Each machine exposes **tags** (sensor values/metrics) via OPC-UA and the REST API. There are 6 **base tags** common to every machine (state, cycle_count, good_count, scrap_count, cycle_time_ms, blocked_by_buffer) plus 15+ **type-specific tags** per machine type. See the [Tag Catalog](tag-catalog.md) for the full list.

### OPC-UA Addressing

Tags are addressed using numeric node IDs: `ns=1;i=<NodeID>`

- Base tags: `ns=1;i=1` (state) through `ns=1;i=6` (blocked_by_buffer)
- Type-specific tags: `ns=1;i=7` and above

Each machine gets its own OPC-UA server on a unique port starting from `base_port` (default 4840).

### Buffers

Machines within a line are connected by buffers. When a downstream machine's input buffer is full, the upstream machine becomes blocked (`blocked_by_buffer = true`). Buffer capacities are configurable per line template.

### Orders and Recipes

The ERP simulator auto-generates orders based on the recipes defined in each line template. The MES dispatcher assigns orders to lines using round-robin or priority strategy. Orders progress through states: `CREATED` → `RELEASED` → `IN_PROGRESS` → `CLOSED`.
