## ADDED Requirements

### Requirement: Machine type registry
The system SHALL provide a global machine type registry where each type registers itself via an `init()` function. Creating a machine instance SHALL use the registry by type name.

#### Scenario: Register a machine type
- **WHEN** the `metal_forming` package's `init()` runs
- **THEN** `machine.Register("metal_forming", factory)` adds the type to the registry

#### Scenario: Create machine from registry
- **WHEN** the system calls `machine.Create("metal_forming", cfg)`
- **THEN** the registry returns a new MetalFormingMachine instance configured with `cfg`

#### Scenario: Unknown machine type
- **WHEN** the system calls `machine.Create("nonexistent", cfg)`
- **THEN** an error is returned indicating the type is not registered

### Requirement: Machine interface compliance
Every machine type SHALL implement the full Machine interface: identity (ID, Type, Role, LineInstanceID), lifecycle (Start, Stop, SendCommand), state (GetState, GetBaseData), tags (GetTags, GetTagDefinitions), production (AssignOrder, ClearOrder, ApplyRecipeOverrides), and line integration (SetOnPartComplete, CanAcceptPart, IsBlockedByBuffer, SetBlockedByBuffer).

#### Scenario: BaseMachine provides default implementation
- **WHEN** a new machine type embeds `*BaseMachine`
- **THEN** it inherits default implementations for all interface methods except `GetTags()`, `GetTagDefinitions()`, and `Start()`

#### Scenario: Type-specific tag definitions
- **WHEN** a machine type implements `GetTagDefinitions()`
- **THEN** it returns tag definitions specific to that machine type (e.g., metal_forming returns press_force, hydraulic_pressure, tool_wear)

### Requirement: Automotive domain machine types
The system SHALL include machine types for automotive manufacturing: `metal_forming` (stamping press), `spot_welder`, `robot_welder`, `robot_pick_place`, and `painting_booth`.

#### Scenario: Metal forming machine
- **WHEN** a `metal_forming` machine runs
- **THEN** it simulates press force (kN), hydraulic pressure (bar), die temperature (C), stroke count, and tool wear (%)

#### Scenario: Painting booth machine
- **WHEN** a `painting_booth` machine runs
- **THEN** it simulates booth temperature, humidity, paint flow rate, film thickness, and cure time

### Requirement: Electronics domain machine types
The system SHALL include machine types for electronics manufacturing: `smt_placement` (SMT pick-and-place), `reflow_oven`, `wave_solder`, and `aoi_inspection` (automated optical inspection).

#### Scenario: SMT placement machine
- **WHEN** an `smt_placement` machine runs
- **THEN** it simulates placement rate (components/hour), nozzle vacuum pressure, feeder status, placement accuracy (um), and head position (X/Y)

#### Scenario: Reflow oven machine
- **WHEN** a `reflow_oven` machine runs
- **THEN** it simulates zone temperatures (preheat, soak, reflow, cooling), conveyor speed, and oxygen level (ppm)

### Requirement: Food & Beverage domain machine types
The system SHALL include machine types for food manufacturing: `filling_machine`, `sealing_machine`, and `labeling_machine`.

#### Scenario: Filling machine
- **WHEN** a `filling_machine` runs
- **THEN** it simulates fill volume (ml), fill rate, nozzle pressure, product temperature, and tank level (%)

#### Scenario: Sealing machine
- **WHEN** a `sealing_machine` runs
- **THEN** it simulates seal temperature, seal pressure, seal integrity check, and conveyor speed

### Requirement: Pharma domain machine types
The system SHALL include machine types for pharmaceutical manufacturing: `reactor_vessel`, `pharma_filler`, and `capping_machine`.

#### Scenario: Reactor vessel
- **WHEN** a `reactor_vessel` runs
- **THEN** it simulates vessel temperature, pressure, agitator speed (RPM), pH level, and batch volume

#### Scenario: Pharma filler
- **WHEN** a `pharma_filler` runs
- **THEN** it simulates fill volume, fill accuracy, particulate count, and clean room pressure differential

### Requirement: Window manufacturing domain machine types
The system SHALL include machine types for window manufacturing: `profile_cutter` (saw/cutter), `corner_welder` (frame welding), and `glass_setter`.

#### Scenario: Profile cutter
- **WHEN** a `profile_cutter` runs
- **THEN** it simulates blade speed (RPM), cut length (mm), feed rate, blade temperature, and cut angle accuracy

#### Scenario: Glass setter
- **WHEN** a `glass_setter` runs
- **THEN** it simulates suction cup vacuum, glass thickness measurement, sealant flow rate, and positioning accuracy

### Requirement: Furniture domain machine types
The system SHALL include machine types for furniture manufacturing: `cnc_router`, `edge_bander`, and `assembly_press`.

#### Scenario: CNC router
- **WHEN** a `cnc_router` runs
- **THEN** it simulates spindle speed (RPM), XY position, cut depth, dust extraction airflow, and tool wear

#### Scenario: Edge bander
- **WHEN** an `edge_bander` runs
- **THEN** it simulates glue temperature, feed speed, trimmer position, edge tape tension, and press roller force

### Requirement: Metal parts domain machine types
The system SHALL include machine types for metal fabrication: `laser_cutter`, `press_brake`, and `deburring_machine`.

#### Scenario: Laser cutter
- **WHEN** a `laser_cutter` runs
- **THEN** it simulates laser power (W), cutting speed, assist gas pressure, focal point position, and kerf width

#### Scenario: Press brake
- **WHEN** a `press_brake` runs
- **THEN** it simulates ram force (tons), bend angle, back gauge position, stroke length, and die temperature

### Requirement: Plastic parts domain machine types
The system SHALL include machine types for plastic manufacturing: `injection_molding`, `extruder`, and `trimming_press`.

#### Scenario: Injection molding machine
- **WHEN** an `injection_molding` machine runs
- **THEN** it simulates barrel temperatures (4 zones), injection pressure, mold temperature, clamp force, cooling time, and shot weight

#### Scenario: Extruder
- **WHEN** an `extruder` runs
- **THEN** it simulates barrel zone temperatures, screw speed (RPM), melt pressure, die pressure, and haul-off speed

### Requirement: Realistic tag value simulation
Each machine type SHALL simulate tag values with realistic ranges, gradual changes (no discontinuous jumps during normal operation), and correlation between related tags (e.g., higher spindle speed correlates with higher temperature).

#### Scenario: Gradual temperature change
- **WHEN** a machine starts from cold
- **THEN** temperature tags ramp up gradually over multiple cycles rather than jumping to operating temperature instantly

#### Scenario: Correlated tags
- **WHEN** a `cnc_milling` machine increases spindle speed
- **THEN** the spindle temperature tag also trends upward with a delay

### Requirement: Minimum 15 tags per machine type
Each machine type SHALL define a minimum of 15 tag definitions covering process parameters, quality metrics, and machine health indicators.

#### Scenario: Tag count check
- **WHEN** `GetTagDefinitions()` is called on any machine type
- **THEN** at least 15 tag definitions are returned
