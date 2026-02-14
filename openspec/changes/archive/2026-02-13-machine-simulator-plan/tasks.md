## 1. Project Scaffolding

- [x] 1.1 Initialize Go module (`go mod init`), create directory structure per design (cmd/, internal/, config/)
- [x] 1.2 Add core dependencies: chi, gopcua, modbus, zerolog, sqlite, yaml.v3, env, gorilla/websocket, uuid
- [x] 1.3 Create `cmd/simulator/main.go` stub with flag parsing (`--profile`) and signal handling

## 2. Event Bus

- [x] 2.1 Implement `internal/event/bus.go`: Bus struct with Subscribe, Publish, thread-safe handler map
- [x] 2.2 Define event type constants: machine.state_change, machine.cycle_complete, line.part_moved, order.created, order.closed, buffer.full, buffer.empty
- [x] 2.3 Write unit tests for bus: subscribe, publish, fan-out to multiple handlers, concurrent publish safety

## 3. Config System

- [x] 3.1 Define config structs in `internal/config/types.go`: GlobalConfig, LineTemplateConfig, ProfileConfig, AdapterConfig, MachineStageConfig, RecipeConfig, BufferConfig
- [x] 3.2 Implement built-in defaults in `internal/config/defaults.go`
- [x] 3.3 Implement multi-file YAML loader in `internal/config/loader.go`: load global.yaml, scan config/lines/ for templates, load profile, load adapter configs
- [x] 3.4 Implement config precedence: built-in defaults → global.yaml → profile overrides → environment variables → CLI flags
- [x] 3.5 Implement config validation: unknown machine types, duplicate IDs, invalid ranges, profile-template consistency, min 2 recipes per template
- [x] 3.6 Write unit tests for config loading and validation (valid config, missing profile, unknown template, invalid values)

## 4. Machine Core

- [x] 4.1 Define Machine interface in `internal/machine/machine.go` with all methods: ID, Type, Role, LineInstanceID, Start, Stop, SendCommand, GetState, GetBaseData, GetTags, GetTagDefinitions, AssignOrder, ClearOrder, GetCurrentOrder, ApplyRecipeOverrides, UpdateConfig, GetSettings, SetOnPartComplete, CanAcceptPart, IsBlockedByBuffer, SetBlockedByBuffer
- [x] 4.2 Define supporting types: State enum (IDLE, RUNNING, STOPPED, FAULT, MAINTENANCE), Command enum, ValidTransitions map, BaseData, MachineSettings, MachineInstanceConfig, RecipeOverrides
- [x] 4.3 Implement BaseMachine struct with full default implementation: state machine, tick loop goroutine, cycle simulation, breakdown/fault logic, auto-recovery, scrap simulation, recipe override save/restore
- [x] 4.4 Implement TagDefinition and tag helpers in `internal/machine/tags.go`
- [x] 4.5 Implement machine type registry in `internal/machine/registry.go`: Register(), Create(), ListTypes()
- [x] 4.6 Write unit tests for BaseMachine: state transitions, cycle completion, breakdown/recovery, scrap rate, recipe override apply/revert, blocked by buffer

## 5. Simulation Engine

- [x] 5.1 Implement `internal/simulation/engine.go`: Engine struct with machine registration, Start/Stop lifecycle, context-based cancellation
- [x] 5.2 Implement time scaling: SetTimeScale propagates to all machines, affects cycle times
- [x] 5.3 Implement random factor: SetRandomFactor propagates to all machines
- [x] 5.4 Wire engine to event bus: publish machine.state_change and machine.cycle_complete events
- [x] 5.5 Write unit tests for engine lifecycle and time scaling

## 6. Line System

- [x] 6.1 Implement Part struct and Buffer (FIFO queue) in `internal/line/line.go`: Add, Remove, Peek, IsFull, IsEmpty, Count, Clear, with mutex protection
- [x] 6.2 Implement Line struct: ordered machine references, buffer map between stages, line instance ID, template reference
- [x] 6.3 Implement LineTemplate loading in `internal/line/template.go`: parse YAML, resolve machine stages, create buffer configs, load embedded recipes
- [x] 6.4 Implement line instantiation: given a template and instance number, create Line with unique machine IDs (`<instance>-stage-<N>`) and buffers
- [x] 6.5 Implement LineManager in `internal/line/manager.go`: manage all active lines, route parts between machines via buffers, handle upstream blocking on full buffer, downstream wait on empty buffer
- [x] 6.6 Implement deadlock detection and recovery in LineManager: detect all-stopped + non-empty buffers, clear buffers, reset machines
- [x] 6.7 Wire LineManager to event bus: publish line.part_moved, buffer.full, buffer.empty events
- [x] 6.8 Write unit tests for Buffer (FIFO, capacity, full/empty), line instantiation, and part flow

## 7. Recipe System

- [x] 7.1 Implement `internal/recipe/recipe.go`: Recipe struct (product_id, description, planned_qty_range, per-stage overrides), RecipeCatalog for lookup by line template + product_id
- [x] 7.2 Implement recipe catalog population from loaded line templates at startup
- [x] 7.3 Write unit tests for recipe lookup and override merging

## 8. Storage Layer

- [x] 8.1 Implement `internal/storage/db.go`: SQLite connection with WAL mode, schema migration (create tables on first run)
- [x] 8.2 Create schema: orders table (id, product_id, line_instance_id, line_template, status, planned_qty, good_qty, scrap_qty, priority, timestamps), operations table (id, order_id, machine_id, stage, status, good_qty, scrap_qty, timestamps)
- [x] 8.3 Implement OrderRepository in `internal/storage/orders.go`: Create, GetByID, List (with filters: status, line_instance_id, product_id, limit, offset), UpdateStatus, UpdateQuantities
- [x] 8.4 Implement OperationRepository in `internal/storage/operations.go`: Create, GetByID, ListByOrderID, UpdateStatus, UpdateQuantities
- [x] 8.5 Write unit tests for repositories with in-memory SQLite

## 9. ERP System

- [x] 9.1 Implement `internal/erp/erp.go`: ERP struct with RegisterProduct, auto-order generation loop, manual order creation
- [x] 9.2 Implement adaptive product discovery: register products from active line instances and their recipe catalogs
- [x] 9.3 Implement auto-order generation: random line instance selection, random product from recipe, planned_qty from recipe range, time-scale-adjusted interval
- [x] 9.4 Implement backlog control: per-line-instance max pending orders, skip lines at capacity
- [x] 9.5 Implement order lifecycle management: CREATED → RELEASED → IN_PROGRESS → CLOSED/CANCELLED
- [x] 9.6 Wire ERP to event bus: publish order.created, order.released, order.closed events
- [x] 9.7 Write unit tests for order generation, backlog control, and lifecycle transitions

## 10. MES System

- [x] 10.1 Implement `internal/mes/mes.go`: MES struct with dispatch loop, operation tracking, progress monitoring
- [x] 10.2 Implement dispatch: receive RELEASED orders, create operations per line stage (OP10, OP20...), apply recipe overrides to machines
- [x] 10.3 Implement dispatch strategies: round-robin, least-busy, priority — selectable via config
- [x] 10.4 Implement operation tracking: subscribe to machine.cycle_complete events, update operation good_qty/scrap_qty, advance to next stage when complete
- [x] 10.5 Implement order completion: when last operation completes, close order, revert recipe overrides on machines
- [x] 10.6 Implement order cancellation: stop active operations, revert recipe overrides, update order status
- [x] 10.7 Wire MES to event bus: publish order.progress events
- [x] 10.8 Write unit tests for dispatch, operation tracking, and order completion

## 11. Machine Types — Automotive

- [x] 11.1 Implement `metal_forming` machine type: press force, hydraulic pressure, die temperature, stroke count, tool wear + 10 more tags, self-registering init()
- [x] 11.2 Implement `spot_welder` machine type: weld current, electrode force, nugget size, tip wear + tags
- [x] 11.3 Implement `robot_welder` machine type: joint temps, arc voltage/current, wire feed, gas flow + tags
- [x] 11.4 Implement `robot_pick_place` machine type: 6 joint angles, TCP position, gripper force + tags
- [x] 11.5 Implement `painting_booth` machine type: booth temp, humidity, paint flow, film thickness, cure time + tags

## 12. Machine Types — Electronics

- [x] 12.1 Implement `smt_placement` machine type: placement rate, nozzle vacuum, feeder status, accuracy, head XY + tags
- [x] 12.2 Implement `reflow_oven` machine type: zone temps (preheat/soak/reflow/cooling), conveyor speed, O2 level + tags
- [x] 12.3 Implement `wave_solder` machine type: solder pot temp, wave height, flux density, conveyor speed + tags
- [x] 12.4 Implement `aoi_inspection` machine type: camera resolution, inspection speed, defect count, false positive rate + tags

## 13. Machine Types — Food & Beverage

- [x] 13.1 Implement `filling_machine` machine type: fill volume, fill rate, nozzle pressure, product temp, tank level + tags
- [x] 13.2 Implement `sealing_machine` machine type: seal temp, seal pressure, seal integrity, conveyor speed + tags
- [x] 13.3 Implement `labeling_machine` machine type: label position accuracy, applicator speed, label tension, printer DPI + tags

## 14. Machine Types — Pharma

- [x] 14.1 Implement `reactor_vessel` machine type: vessel temp, pressure, agitator RPM, pH, batch volume + tags
- [x] 14.2 Implement `pharma_filler` machine type: fill volume, fill accuracy, particulate count, clean room dP + tags
- [x] 14.3 Implement `capping_machine` machine type: torque, cap position accuracy, reject rate, feed rate + tags

## 15. Machine Types — Window Manufacturing

- [x] 15.1 Implement `profile_cutter` machine type: blade RPM, cut length, feed rate, blade temp, angle accuracy + tags
- [x] 15.2 Implement `corner_welder` machine type: weld temp, weld time, clamp force, joint strength + tags
- [x] 15.3 Implement `glass_setter` machine type: suction vacuum, glass thickness, sealant flow, positioning accuracy + tags

## 16. Machine Types — Furniture

- [x] 16.1 Implement `cnc_router` machine type: spindle RPM, XY position, cut depth, dust extraction, tool wear + tags
- [x] 16.2 Implement `edge_bander` machine type: glue temp, feed speed, trimmer position, tape tension, roller force + tags
- [x] 16.3 Implement `assembly_press` machine type: press force, alignment accuracy, dwell time, cycle pressure + tags

## 17. Machine Types — Metal Parts

- [x] 17.1 Implement `laser_cutter` machine type: laser power, cutting speed, assist gas pressure, focal point, kerf width + tags
- [x] 17.2 Implement `press_brake` machine type: ram force, bend angle, back gauge position, stroke length, die temp + tags
- [x] 17.3 Implement `deburring_machine` machine type: brush speed, contact pressure, feed rate, surface finish measurement + tags

## 18. Machine Types — Plastic Parts

- [x] 18.1 Implement `injection_molding` machine type: barrel temps (4 zones), injection pressure, mold temp, clamp force, cooling time, shot weight + tags
- [x] 18.2 Implement `extruder` machine type: barrel zone temps, screw RPM, melt pressure, die pressure, haul-off speed + tags
- [x] 18.3 Implement `trimming_press` machine type: press force, trim accuracy, feed rate, die clearance + tags

## 19. Protocol Adapter Interface

- [x] 19.1 Define `protocol.Adapter` interface in `internal/protocol/adapter.go`: Name, Start, Stop, RegisterMachine, DeregisterMachine
- [x] 19.2 Define AdapterMachineConfig struct: Port, UnitID, Metadata

## 20. OPC-UA Adapter

- [x] 20.1 Implement OPC-UA adapter in `internal/protocol/opcua/`: per-machine server creation, tag node mapping from GetTagDefinitions, tag cache with configurable update rate
- [x] 20.2 Implement OPC-UA port auto-assignment from base_port config
- [x] 20.3 Implement RegisterMachine/DeregisterMachine: create/destroy OPC-UA server for a machine
- [x] 20.4 Load OPC-UA config from `config/adapters/opcua.yaml`
- [x] 20.5 Write integration test: start adapter, register a machine, verify OPC-UA client can read tags

## 21. Modbus TCP Adapter

- [x] 21.1 Implement Modbus adapter in `internal/protocol/modbus/`: single gateway server, dynamic slave registration
- [x] 21.2 Implement dynamic register mapping: map machine tags to sequential holding registers per unit ID
- [x] 21.3 Implement RegisterMachine/DeregisterMachine: add/remove slave with auto-assigned unit ID
- [x] 21.4 Expose register map via API endpoint (GET /api/modbus/register-map)
- [x] 21.5 Load Modbus config from `config/adapters/modbus.yaml`

## 22. REST API

- [x] 22.1 Set up chi router with CORS, logging middleware in `internal/api/handlers.go`
- [x] 22.2 Implement line endpoints: GET /api/lines, GET /api/lines/{id}, GET /api/lines/{id}/buffers, POST /api/lines/{id}/start, POST /api/lines/{id}/stop
- [x] 22.3 Implement machine endpoints: GET /api/machines, GET /api/machines/{id}, GET /api/machines/{id}/tags, POST /api/machines/{id}/command
- [x] 22.4 Implement order endpoints: GET /api/orders (with filters), GET /api/orders/{id}, POST /api/orders, PUT /api/orders/{id}/status, DELETE /api/orders/{id}
- [x] 22.5 Implement recipe endpoints: GET /api/lines/{id}/recipes, GET /api/recipes
- [x] 22.6 Implement simulation control endpoints: GET /api/simulation, PUT /api/simulation/timescale, PUT /api/simulation/random-factor
- [x] 22.7 Implement settings endpoints: GET /api/settings/machines, PUT /api/settings/machines/{id}, GET /api/settings/buffers, PUT /api/settings/buffers/{id}
- [x] 22.8 Implement health endpoint: GET /health

## 23. WebSocket

- [x] 23.1 Implement WebSocket hub in `internal/api/`: connection management, broadcast to all clients
- [x] 23.2 Wire event bus to WebSocket hub: subscribe to machine, order, and buffer events, broadcast as JSON messages
- [x] 23.3 Implement WebSocket endpoint: ws://localhost:8081/ws

## 24. Web Dashboard

- [x] 24.1 Create base HTML layout template with htmx, navigation (Dashboard, Lines, Machines, Orders, Simulation), and WebSocket connection script
- [x] 24.2 Implement dashboard overview page (`/`): line instance cards with machine states, buffer levels, active order, active product
- [x] 24.3 Implement line detail page (`/lines/{id}`): machine sequence visualization, buffer fill bars, active recipe display, order progress
- [x] 24.4 Implement machine detail page (`/machines/{id}`): all tags with live values, state indicator, command buttons (START, STOP, RESET, FAULT), current order info
- [x] 24.5 Implement order management page (`/orders`): order list with status/line/product filters, manual order creation form with line instance and product dropdowns
- [x] 24.6 Implement simulation control page (`/simulation`): time scale slider, random factor slider, active profile display, line/machine counts
- [x] 24.7 Add htmx polling/WebSocket triggers for real-time updates on all pages
- [x] 24.8 Create static CSS for layout, line visualization, buffer indicators, and machine state colors

## 25. Config YAML Files — Line Templates

- [x] 25.1 Create `config/lines/automotive/welding-line.yaml`: 4 stages (metal_forming → spot_welder → robot_welder → robot_pick_place), 2+ recipes
- [x] 25.2 Create `config/lines/automotive/assembly-line.yaml`: 4 stages (robot_pick_place → assembly_press → spot_welder → painting_booth), 2+ recipes
- [x] 25.3 Create `config/lines/electronics/smt-line.yaml`: 4 stages (smt_placement → reflow_oven → aoi_inspection → labeling_machine), 2+ recipes
- [x] 25.4 Create `config/lines/electronics/through-hole-line.yaml`: 3 stages (robot_pick_place → wave_solder → aoi_inspection), 2+ recipes
- [x] 25.5 Create `config/lines/food-beverage/filling-line.yaml`: 3 stages (filling_machine → sealing_machine → labeling_machine), 2+ recipes
- [x] 25.6 Create `config/lines/pharma/batch-line.yaml`: 3 stages (reactor_vessel → pharma_filler → capping_machine), 2+ recipes
- [x] 25.7 Create `config/lines/window/frame-line.yaml`: 3 stages (profile_cutter → corner_welder → glass_setter), 2+ recipes
- [x] 25.8 Create `config/lines/furniture/assembly-line.yaml`: 3 stages (cnc_router → edge_bander → assembly_press), 2+ recipes
- [x] 25.9 Create `config/lines/metal-parts/fabrication-line.yaml`: 3 stages (laser_cutter → press_brake → deburring_machine), 2+ recipes
- [x] 25.10 Create `config/lines/plastic-parts/molding-line.yaml`: 3 stages (extruder → injection_molding → trimming_press), 2+ recipes

## 26. Config YAML Files — Profiles & Global

- [x] 26.1 Create `config/global.yaml`: time_scale, random_factor, tick_rate, log level, API host/port, ERP mode/interval, MES dispatch interval/strategy
- [x] 26.2 Create `config/profiles/demo-mixed.yaml`: 1x automotive-welding, 1x electronics-smt, 1x food-filling, 1x pharma-batch
- [x] 26.3 Create `config/profiles/automotive-focus.yaml`: 2x automotive-welding, 1x automotive-assembly, 1x metal-parts-fabrication
- [x] 26.4 Create `config/profiles/full-catalog.yaml`: 1x of every line template
- [x] 26.5 Create `config/adapters/opcua.yaml`: enabled, base_port, update_rate_ms, max_sessions, hostname
- [x] 26.6 Create `config/adapters/modbus.yaml`: enabled, port, max_clients, update_interval_ms

## 27. Composition Root & Wiring

- [x] 27.1 Implement `internal/app/app.go` App.New(): create event bus, load config, create DB, create engine, instantiate lines from profile, create machines via registry, create line manager, create ERP with registered products, create MES, create protocol adapters, create API/web handlers
- [x] 27.2 Implement App.Run(): start all components in dependency order, wait for shutdown signal (SIGINT/SIGTERM), stop in reverse order with 30s timeout
- [x] 27.3 Add blank import of `internal/machine/types` package in main.go to trigger machine type registration
- [x] 27.4 End-to-end smoke test: start simulator with demo-mixed profile, verify all lines running, create an order via API, observe it flow through a line

## 28. Docker & Deployment

- [x] 28.1 Create multi-stage Dockerfile: Go 1.23 build stage → alpine runtime, embed config/ and static assets
- [x] 28.2 Create docker-compose.yaml with profile selection via SIMULATOR_PROFILE environment variable
- [x] 28.3 Add health check to Dockerfile using GET /health
