## Context

This is a clean rewrite of the factory simulator at `/Users/daniel/github/machine-simulator`. The original is a ~13,300 LOC Go project with 7 machine types, 9 machine instances across 2 lines, OPC-UA + Modbus connectivity, ERP/MES simulation, and an htmx web UI. It works well but is limited to a fixed set of machines and lines with no concept of products, recipes, or configurable factory layouts.

The new simulator must support 10-12 pre-configured line templates across 8 industry domains, runtime profile selection, multi-instance lines, per-line-type product recipes, and a pluggable protocol adapter system — while keeping the proven Go + SQLite + htmx stack.

The original codebase provides strong patterns to carry forward: goroutine-per-machine concurrency, callback-based events, composition over inheritance, context-based lifecycle, and interface-driven machine abstraction. Key areas to redesign: machine type registration (hardcoded factory switch → registry), protocol coupling (direct machine-to-server wiring → adapter interface), configuration (single file → multi-file composition), and main.go wiring (500-line function → composition root).

## Goals / Non-Goals

**Goals:**

- Configurable factory layouts via named profiles (select 3-4 lines from a catalog of 10-12)
- Multi-instance support for any line type (run N copies independently)
- Product recipe system tied to line types (per-product cycle times, quality targets, materials)
- Pluggable protocol adapter interface (OPC-UA and Modbus built-in, extensible)
- Adaptive ERP/MES that discovers active lines and their recipes at startup
- Multi-file YAML configuration with clear separation of concerns
- Expanded machine type library (~15-20 types across 8 industry domains)
- Clean, testable architecture with a composition root pattern

**Non-Goals:**

- Non-sequential line topologies (no parallel branches, merge points, or DAG routing)
- Runtime line reconfiguration (lines are fixed at startup via profile)
- High test coverage targets (minimal viable: critical paths only)
- Migration from or backwards compatibility with the original simulator
- Multi-plant / distributed simulation
- Persistent recipe editing through the UI (recipes are YAML-defined)
- OPC-UA write operations or command execution via protocols

## Decisions

### 1. Project structure

```
machine-simulator-2/
├── cmd/simulator/
│   └── main.go                     # Thin entry point → composition root
├── internal/
│   ├── app/
│   │   └── app.go                  # Composition root: builds and wires all components
│   ├── machine/
│   │   ├── machine.go              # Machine interface + BaseMachine
│   │   ├── registry.go             # Machine type registry (register + create by name)
│   │   ├── tags.go                 # TagDefinition, tag helpers
│   │   └── types/                  # One file per machine type
│   │       ├── metal_forming.go
│   │       ├── injection_molding.go
│   │       ├── smt_placement.go
│   │       ├── filling_machine.go
│   │       └── ...                 # ~15-20 types
│   ├── line/
│   │   ├── line.go                 # Line struct, buffer, part flow
│   │   ├── manager.go              # LineManager: coordinates all active lines
│   │   └── template.go             # LineTemplate: loaded from catalog YAML
│   ├── recipe/
│   │   └── recipe.go               # Recipe struct, lookup by line type + product
│   ├── simulation/
│   │   └── engine.go               # Core engine: time scaling, machine orchestration
│   ├── erp/
│   │   └── erp.go                  # ERP order generation, adapts to active recipes
│   ├── mes/
│   │   └── mes.go                  # MES dispatch, operation tracking
│   ├── protocol/
│   │   ├── adapter.go              # ProtocolAdapter interface
│   │   ├── opcua/                  # OPC-UA adapter implementation
│   │   └── modbus/                 # Modbus adapter implementation
│   ├── storage/
│   │   ├── db.go                   # SQLite setup, migrations
│   │   ├── orders.go               # Order repository
│   │   └── operations.go           # Operation repository
│   ├── config/
│   │   ├── loader.go               # Multi-file YAML loader + validation
│   │   ├── types.go                # Config structs (global, machine, line, profile)
│   │   └── defaults.go             # Default values
│   ├── event/
│   │   └── bus.go                  # Typed event bus (replaces raw callbacks)
│   ├── api/
│   │   └── handlers.go             # REST API + WebSocket
│   └── web/
│       ├── handlers.go             # htmx page handlers
│       ├── templates/              # HTML templates
│       └── static/                 # CSS, JS
├── config/
│   ├── global.yaml                 # Simulator-wide settings
│   ├── lines/                      # Line catalog (one YAML per line type)
│   │   ├── automotive/
│   │   │   ├── welding-line.yaml
│   │   │   └── assembly-line.yaml
│   │   ├── electronics/
│   │   │   ├── smt-line.yaml
│   │   │   └── through-hole-line.yaml
│   │   ├── food-beverage/
│   │   │   └── filling-line.yaml
│   │   ├── pharma/
│   │   │   └── batch-line.yaml
│   │   ├── window/
│   │   │   └── frame-line.yaml
│   │   ├── furniture/
│   │   │   └── assembly-line.yaml
│   │   ├── metal-parts/
│   │   │   └── fabrication-line.yaml
│   │   └── plastic-parts/
│   │       └── molding-line.yaml
│   ├── profiles/                   # Named factory profiles
│   │   ├── demo-mixed.yaml
│   │   ├── automotive-focus.yaml
│   │   └── full-catalog.yaml
│   └── adapters/                   # Protocol adapter configs
│       ├── opcua.yaml
│       └── modbus.yaml
├── Dockerfile
├── docker-compose.yaml
├── go.mod
└── go.sum
```

**Rationale**: The original's flat `internal/` with all machine types in one package works at 7 types but won't at 15-20. Separating `internal/machine/types/` keeps the machine interface clean. The `internal/app/` composition root replaces the 500-line `main.go`. Config files are organized by domain for discoverability.

**Alternative considered**: Monorepo with separate binaries per domain — rejected because a single binary with profile selection is simpler to deploy and matches the user's intent.

### 2. Machine type registry

Replace the hardcoded factory switch with a self-registering registry:

```go
// internal/machine/registry.go
type MachineFactory func(cfg MachineInstanceConfig) Machine

var registry = map[string]MachineFactory{}

func Register(typeName string, factory MachineFactory) {
    registry[typeName] = factory
}

func Create(typeName string, cfg MachineInstanceConfig) (Machine, error) {
    factory, ok := registry[typeName]
    if !ok {
        return nil, fmt.Errorf("unknown machine type: %s", typeName)
    }
    return factory(cfg), nil
}
```

Each machine type self-registers via `init()`:

```go
// internal/machine/types/metal_forming.go
func init() {
    machine.Register("metal_forming", func(cfg machine.MachineInstanceConfig) machine.Machine {
        return NewMetalFormingMachine(cfg)
    })
}
```

**Rationale**: Adding a new machine type only requires creating one file. No changes to factory switches or constant lists. The original required edits in 3+ places per new type.

**Alternative considered**: Interface-based plugin system with shared libraries — rejected as over-engineering for a single-binary application. `init()` registration is idiomatic Go.

### 3. Line catalog and templates

Each line type is a YAML file that defines the machine sequence, buffer config, and available recipes:

```yaml
# config/lines/automotive/welding-line.yaml
name: automotive-welding
domain: automotive
description: "Multi-stage automotive welding line"
machines:
  - stage: 1
    type: metal_forming
    role: "Stamping press"
    defaults:
      cycle_time_ms: 5000
      scrap_rate: 0.02
      breakdown_probability: 0.001
  - stage: 2
    type: spot_welder
    role: "Body panel welding"
    defaults:
      cycle_time_ms: 3000
      scrap_rate: 0.01
  - stage: 3
    type: robot_welder
    role: "Frame welding"
    defaults:
      cycle_time_ms: 8000
      scrap_rate: 0.015
  - stage: 4
    type: robot_pick_place
    role: "Assembly placement"
    defaults:
      cycle_time_ms: 2500
      scrap_rate: 0.005

buffers:
  default_capacity: 10
  overrides:
    - from_stage: 1
      to_stage: 2
      capacity: 15

recipes:
  - product_id: "DOOR-PANEL-A"
    description: "Front door panel assembly"
    planned_qty_range: [50, 200]
    overrides:
      1: { cycle_time_ms: 4500 }
      3: { cycle_time_ms: 7000, scrap_rate: 0.02 }
  - product_id: "FENDER-B"
    description: "Rear fender"
    planned_qty_range: [30, 100]
    overrides:
      1: { cycle_time_ms: 6000 }
```

At startup, the template is loaded and instantiated N times (per profile). Each instance gets a unique ID: `automotive-welding-1`, `automotive-welding-2`, etc. Instance machines get IDs like `automotive-welding-1-stage-1`.

**Rationale**: YAML is human-readable and versionable. Keeping recipes inside the line template enforces the "recipe per line type" constraint. Stage-based overrides let products customize individual machines without duplicating the full config.

**Alternative considered**: Separate recipe YAML files referenced by line templates — rejected because it adds indirection. Recipes are inherently tied to line types, so co-location is clearer.

### 4. Profile system

Profiles are YAML files that declare which lines to activate:

```yaml
# config/profiles/demo-mixed.yaml
name: demo-mixed
description: "Mixed manufacturing demo with 4 lines"
lines:
  - template: automotive-welding
    instances: 1
  - template: electronics-smt
    instances: 1
  - template: food-filling
    instances: 1
  - template: pharma-batch
    instances: 1

overrides:
  global:
    time_scale: 1.0
    random_factor: 0.15
```

The simulator loads a profile at startup via `--profile demo-mixed` or `SIMULATOR_PROFILE=demo-mixed`. The config loader:
1. Reads `config/global.yaml` for base settings
2. Reads the profile YAML to determine active lines
3. For each line entry, loads the template from `config/lines/<domain>/<template>.yaml`
4. Instantiates N copies of each template
5. Loads protocol adapter configs from `config/adapters/`
6. Applies environment variable overrides last

**Rationale**: Profiles decouple "what lines exist" from "which lines are active." Users can ship 10+ line templates but only activate 3-4 per run. Named profiles make common setups repeatable.

### 5. Protocol adapter interface

```go
// internal/protocol/adapter.go
type Adapter interface {
    // Name returns the adapter identifier (e.g., "opcua", "modbus")
    Name() string
    // Start begins serving protocol connections
    Start(ctx context.Context) error
    // Stop gracefully shuts down
    Stop() error
    // RegisterMachine exposes a machine's tags via this protocol
    RegisterMachine(m machine.Machine, cfg AdapterMachineConfig) error
    // DeregisterMachine removes a machine from this protocol
    DeregisterMachine(machineID string) error
}

type AdapterMachineConfig struct {
    Port     int               // For per-machine servers (OPC-UA)
    UnitID   uint8             // For shared servers (Modbus)
    Metadata map[string]string // Protocol-specific settings
}
```

Built-in adapters:
- **OPC-UA** (`internal/protocol/opcua/`): One server per machine, same as original. Reads tags via `machine.GetTags()`.
- **Modbus** (`internal/protocol/modbus/`): Single gateway server. Maps machine tags to register addresses dynamically (unlike original's hardcoded slaves).

The composition root iterates active lines, and for each machine calls `adapter.RegisterMachine()` on all enabled adapters. Adding MQTT or Sparkplug B later means implementing `Adapter` and adding a config file — no changes to simulation core.

**Rationale**: The original has separate, incompatible wiring for OPC-UA (per-machine server manager) and Modbus (global server with hardcoded slaves). A unified interface means consistent registration and lifecycle management.

**Alternative considered**: Event-driven adapter (adapters subscribe to machine events) — rejected because industrial protocols are predominantly poll-based (clients read tags), not push-based. The current `GetTags()` poll model fits naturally.

### 6. Event bus (replaces raw callbacks)

```go
// internal/event/bus.go
type Event struct {
    Type      string
    MachineID string
    LineID    string
    Data      interface{}
}

type Handler func(Event)

type Bus struct {
    mu       sync.RWMutex
    handlers map[string][]Handler
}

func (b *Bus) Subscribe(eventType string, handler Handler)
func (b *Bus) Publish(event Event)
```

Event types: `machine.state_change`, `machine.cycle_complete`, `line.part_moved`, `order.created`, `order.closed`, `buffer.full`, `buffer.empty`.

**Rationale**: The original uses 6+ separate callback setters (`SetOnPartComplete`, `SetOnStateChange`, `SetOnOrderCreated`, etc.) wired manually in main.go. A central bus simplifies cross-component communication and makes it trivial to add new subscribers (webhooks, analytics, logging) without modifying source components.

**Alternative considered**: Go channels per event type — rejected because fan-out (multiple subscribers) requires manual multiplexing. The bus handles fan-out natively.

### 7. ERP/MES adaptation to active config

At startup, ERP and MES receive the list of active line instances and their recipes:

```go
// ERP discovers what it can order
erp := erp.New(erp.Config{
    Mode:     "auto",
    Interval: 30 * time.Second,
})
for _, instance := range activeLineInstances {
    for _, recipe := range instance.Template.Recipes {
        erp.RegisterProduct(instance.ID, recipe)
    }
}
```

When ERP generates an auto-order, it:
1. Picks a random active line instance
2. Picks a random product from that instance's recipe list
3. Creates an order with `planned_qty` from the recipe's `planned_qty_range`
4. Tags the order with the line instance ID and product ID

MES then dispatches to the correct line instance, applying recipe overrides (per-stage cycle times, scrap rates) to the machines for that order.

**Rationale**: This keeps ERP/MES logic generic. They don't know about "automotive" vs "pharma" — they only know line instances and their registered products. Adding a new domain requires zero ERP/MES code changes.

### 8. Database schema

SQLite with expanded tables:

```sql
-- Orders now track product and line instance
CREATE TABLE orders (
    id TEXT PRIMARY KEY,
    product_id TEXT NOT NULL,
    line_instance_id TEXT NOT NULL,
    line_template TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'CREATED',
    planned_qty INTEGER NOT NULL,
    good_qty INTEGER NOT NULL DEFAULT 0,
    scrap_qty INTEGER NOT NULL DEFAULT 0,
    priority INTEGER NOT NULL DEFAULT 50,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    started_at TIMESTAMP,
    closed_at TIMESTAMP
);

-- Operations track per-stage work
CREATE TABLE operations (
    id TEXT PRIMARY KEY,
    order_id TEXT NOT NULL REFERENCES orders(id),
    machine_id TEXT NOT NULL,
    stage INTEGER NOT NULL,
    status TEXT NOT NULL DEFAULT 'PENDING',
    good_qty INTEGER NOT NULL DEFAULT 0,
    scrap_qty INTEGER NOT NULL DEFAULT 0,
    started_at TIMESTAMP,
    completed_at TIMESTAMP
);
```

**Rationale**: Adding `product_id` and `line_instance_id` to orders enables filtering by product and line in the API/UI. The `line_template` field allows analytics across instances of the same type. Schema stays simple — no recipe tables needed since recipes are config-defined, not data.

### 9. Machine interface refinements

Carry forward the 22-method interface with targeted changes:

```go
type Machine interface {
    // Identity
    ID() string
    Type() string
    Role() string       // NEW: human-readable role (e.g., "Stamping press")
    LineInstanceID() string  // NEW: which line instance this belongs to

    // Lifecycle
    Start(ctx context.Context)
    Stop()
    SendCommand(cmd Command) error

    // State
    GetState() State
    GetBaseData() BaseData

    // Tags (unchanged)
    GetTags() map[string]interface{}
    GetTagDefinitions() []TagDefinition

    // Production
    AssignOrder(orderID, operationID, productID string)  // CHANGED: added productID
    ClearOrder()
    GetCurrentOrder() (orderID, operationID string)

    // Runtime config
    ApplyRecipeOverrides(overrides RecipeOverrides)  // NEW: per-product tuning
    UpdateConfig(settings MachineSettings)
    GetSettings() MachineSettings

    // Line integration (unchanged)
    SetOnPartComplete(fn func(machineID string, good bool))
    CanAcceptPart() bool
    IsBlockedByBuffer() bool
    SetBlockedByBuffer(blocked bool)
}
```

`ApplyRecipeOverrides` lets MES adjust cycle time and scrap rate when starting an order for a specific product, then reset to defaults when the order completes.

### 10. Composition root

```go
// cmd/simulator/main.go
func main() {
    flag.Parse()
    cfg, err := config.Load(profileName)
    // ...
    app, err := app.New(cfg)
    // ...
    app.Run(ctx)
}

// internal/app/app.go
type App struct { /* all components */ }

func New(cfg *config.Config) (*App, error) {
    // 1. Create event bus
    // 2. Create database + repositories
    // 3. Create simulation engine
    // 4. Load line templates, instantiate per profile
    // 5. Create machines via registry, register with engine
    // 6. Create line manager, wire buffers
    // 7. Create ERP, register products from active recipes
    // 8. Create MES
    // 9. Create protocol adapters, register machines
    // 10. Create API + Web handlers
    // return wired App
}

func (a *App) Run(ctx context.Context) error {
    // Start all components in order
    // Wait for shutdown signal
    // Stop all components in reverse order
}
```

**Rationale**: Extracts the 500-line `main.go` into a testable composition root. `App.New()` can be called from tests with different configs. `App.Run()` manages the full lifecycle.

## Risks / Trade-offs

**[Registry init() ordering]** → Go `init()` execution order is deterministic within a package but not across packages. The `types/` package uses a single `init()` per file which is fine, but we must ensure `internal/machine/types` is imported (blank import `_ "machine-simulator-2/internal/machine/types"`) in the composition root. Mitigation: add a `types.RegisterAll()` function as a safety net.

**[YAML config sprawl]** → 10-12 line templates + profiles + adapter configs = 15+ YAML files. Users may find it hard to discover what's configurable. Mitigation: include a `config/README.md` with the file map, and the web UI shows the active profile/lines.

**[Recipe overrides complexity]** → Per-stage per-product overrides could create subtle bugs if a recipe override changes a machine's scrap rate but the machine doesn't reset after the order. Mitigation: `ApplyRecipeOverrides` saves previous values and `ClearOrder` restores them. Unit test this path.

**[~15-20 machine types to implement]** → Each machine type needs realistic tag definitions and cycle simulation logic. This is the bulk of the implementation effort. Mitigation: start with 4-5 types covering the core domains, add the rest incrementally. Each type is self-contained (one file, self-registering) so they can be added independently.

**[SQLite at scale]** → With 3-4 lines × 4-5 machines × 100x time scale, order throughput could be high. SQLite WAL mode handles concurrent reads well but writes are serialized. Mitigation: batch inserts for operations, and the original simulator proved SQLite handles this load. If it becomes an issue, the repository interface allows swapping to PostgreSQL later.

**[Modbus dynamic register mapping]** → The original hardcodes register layouts. Dynamic mapping (machine tags → Modbus registers) needs a consistent scheme so external clients know where to read. Mitigation: define a standard register layout per machine type in the line template YAML, or auto-assign registers sequentially and expose the mapping via API.

## Open Questions

- **Exact line catalog**: The 10-12 line types need final definition — which machine types compose each line, realistic cycle times, and 2-3 product recipes per line. This will be detailed in the specs artifact.
- **OPC-UA port allocation**: With multi-instance lines, port assignment needs a strategy. Options: auto-increment from a base port, or explicit port ranges per profile. Leaning toward auto-increment with a configurable base.
- **Web UI line visualization**: With 3-4 active lines of 4-5 machines each, the dashboard needs a layout that scales. The original's single-page view may not fit. Consider tabs per line or a grid layout.
