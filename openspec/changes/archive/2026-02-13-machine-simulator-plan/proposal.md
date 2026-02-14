## Why

The existing machine simulator at `/Users/daniel/github/machine-simulator` supports only 2 hardcoded production lines with 9 machines and no concept of products or recipes. It cannot represent different manufacturing domains or allow operators to choose which lines to run. A clean rewrite is needed to create a configurable, multi-domain factory simulator where users select from a catalog of 10-12 pre-configured line types across 8 industry domains, run multiple instances of any line, and manage product recipes per line type - all while keeping the proven Go + SQLite + htmx stack.

## What Changes

- **Clean rewrite** of the entire simulator (new codebase, cherry-pick patterns from original)
- **Line catalog**: 10-12 pre-configured line templates spanning 8 industry domains (automotive, electronics, food & beverage, pharma/chemical, window manufacturing, furniture, metal parts, plastic parts)
- **Profile system**: Named YAML profiles that define which lines to activate, how many instances of each, enabling quick factory configurations (e.g., `profiles/demo-mixed.yaml`, `profiles/automotive-focus.yaml`)
- **Multi-instance lines**: Any line type can run N copies simultaneously with unique IDs and independent state
- **Recipe management**: Each line type has a fixed set of products it can manufacture, with per-product cycle times, quality targets, and material specifications
- **Adaptive ERP/MES**: Order generation and dispatch automatically adapt to whichever lines and recipes are currently active
- **Pluggable protocol adapters**: Interface-based connectivity layer so OPC-UA, Modbus, and future protocols (MQTT, Sparkplug B, S7) can be added without changing core code
- **Multi-file YAML configuration**: Separate config files per line type, per profile, and for global settings - replacing the monolithic `default.yaml`
- **New machine types**: Domain-specific machines for all 8 industries (SMT pick-and-place, filling machines, reactors, saws, CNC routers, extruders, etc.)

## Capabilities

### New Capabilities

- `line-catalog`: Library of 10-12 pre-configured production line templates across 8 industry domains. Each template defines a sequential machine arrangement, buffer configuration, supported products, and default operating parameters. Domains: automotive (welding line, assembly line), electronics (SMT line, through-hole line), food & beverage (filling/packaging line), pharma (batch processing line), window manufacturing (frame line), furniture (assembly line), metal parts (fabrication line), plastic parts (molding line).

- `profile-system`: Named YAML profiles that declare which line templates to instantiate and how many of each. Profiles compose a complete factory configuration from the line catalog. Enables switching between factory setups without editing core config. The simulator loads a profile at startup to determine its active lines.

- `recipe-management`: Product recipe system where each line type has a fixed set of manufacturable products. Recipes define per-product parameters: cycle time overrides, quality targets (scrap rate, tolerance), material requirements, and operation-specific settings per machine stage. ERP generates orders for products that match the currently active lines.

- `protocol-adapters`: Plugin interface for industrial connectivity protocols. Built-in adapters for OPC-UA (per-machine server) and Modbus TCP (gateway). Interface contract allows adding MQTT, Sparkplug B, S7, ADS, or custom protocols without modifying the simulation core. Each adapter registers/deregisters dynamically based on configuration.

- `simulation-engine`: Core simulation engine managing machines, production lines, FIFO buffers, time scaling, breakdown/fault modeling, and scrap simulation. Each machine runs as an independent goroutine with a tick-based state machine. Supports configurable time acceleration (1x-100x) and random variance. Carries forward proven patterns from the original simulator.

- `erp-mes-system`: Combined ERP and MES simulation that dynamically adapts to the active factory configuration. ERP generates orders targeting active lines with products from their recipe catalogs. MES dispatches operations across line instances using configurable strategies (round-robin, least-busy, priority). Handles order lifecycle: CREATED -> RELEASED -> IN_PROGRESS -> CLOSED/CANCELLED.

- `config-system`: Multi-file YAML configuration architecture. Global settings in `config/global.yaml`, line templates in `config/lines/<domain>/<line-type>.yaml`, profiles in `config/profiles/<name>.yaml`, protocol adapter configs in `config/adapters/<protocol>.yaml`. Environment variable overrides and CLI flags for quick adjustments.

- `web-dashboard`: Server-rendered htmx web UI for real-time monitoring and control. Dashboard showing active lines with buffer levels, machine states, order progress, and recipe information. Controls for machine commands, simulation speed, and runtime settings. WebSocket-powered live updates.

- `rest-api`: REST API for external integration covering orders, machines, lines, recipes, simulation control, settings, and KPIs. WebSocket endpoint for real-time state streaming.

- `machine-types`: Expanded library of machine type implementations covering all 8 industry domains. Each machine type defines its sensor tags, cycle phases, breakdown behavior, and realistic value ranges. Estimated 15-20 machine types total (up from 7 in the original).

### Modified Capabilities

_(None - this is a clean rewrite with no existing specs to modify.)_

## Impact

- **Codebase**: Entirely new Go project at `/Users/daniel/github/machine-simulator-2`. No changes to the original simulator.
- **Architecture**: Modular package structure designed around the line catalog and recipe concepts. Clear separation between simulation core, protocol adapters, persistence, and presentation.
- **Dependencies**: Same core Go dependencies (chi, gopcua, modbus, zerolog, sqlite). May add new deps for additional machine type simulations.
- **Deployment**: Docker support with profile selection via environment variable. Single binary with embedded static assets.
- **Database**: SQLite with expanded schema for recipes, line instances, and product tracking.
- **Configuration**: Breaking change from single-file to multi-file YAML. Not backwards-compatible with original config format (intentional for clean rewrite).
- **API**: New API design informed by original but not backwards-compatible. Expanded to cover recipes, profiles, and multi-instance lines.
