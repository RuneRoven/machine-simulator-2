# Configuration Guide

This document is the complete reference for configuring the machine simulator. It covers every configuration file, field, and override mechanism.

## 1. Configuration Overview

### File Hierarchy

All configuration lives under the `config/` directory:

```
config/
  global.yaml                          # Simulator-wide settings
  adapters/
    opcua.yaml                         # OPC-UA protocol adapter
    modbus.yaml                        # Modbus TCP protocol adapter
  lines/                               # Line templates, organized by domain
    automotive/
      welding-line.yaml
      assembly-line.yaml
    electronics/
      smt-line.yaml
      through-hole-line.yaml
    food-beverage/
      filling-line.yaml
    pharma/
      batch-line.yaml
    window/
      frame-line.yaml
    furniture/
      assembly-line.yaml
    metal-parts/
      fabrication-line.yaml
    plastic-parts/
      molding-line.yaml
  profiles/                            # Factory profiles (select active lines)
    demo-mixed.yaml
    automotive-focus.yaml
    full-catalog.yaml
```

### Loading Order

Configuration is assembled in a strict order. Each step can override values set by previous steps:

1. **Built-in defaults** -- Hardcoded in `internal/config/defaults.go`. Every setting has a default so the simulator can start with minimal configuration.
2. **`config/global.yaml`** -- Overrides built-in defaults for simulator-wide settings. Optional; if the file is missing, defaults are used.
3. **Line templates** (`config/lines/**/*.yaml`) -- All YAML files under `config/lines/` are loaded into a template registry keyed by their `name` field. The subdirectory name is used as the `domain` if the template does not set one explicitly.
4. **Profile** (`config/profiles/<name>.yaml`) -- The selected profile determines which line templates are instantiated and how many instances of each. Profile-level overrides are applied to global settings at this point.
5. **Adapter configs** (`config/adapters/*.yaml`) -- OPC-UA and Modbus adapter settings are loaded, overriding their built-in defaults. Optional; missing files leave defaults in place.
6. **Environment variable overrides** -- Any `SIMULATOR_*` environment variable that is set takes final precedence over all file-based configuration.

After loading, the configuration is validated. Validation checks include: at least one active line, no duplicate machine IDs, at least two recipes per template, non-empty machine types, and value-range constraints on numeric fields.

---

## 2. Global Settings (`global.yaml`)

File: `config/global.yaml`

```yaml
time_scale: 1.0
random_factor: 0.1
tick_rate_ms: 100
log_level: "info"
api:
  host: "0.0.0.0"
  port: 8081
erp:
  mode: "auto"
  auto_order_interval_ms: 30000
  max_backlog: 5
mes:
  dispatch_interval_ms: 5000
  dispatch_strategy: "round-robin"
```

### Top-Level Fields

| Field | Type | Default | Valid Range | Description |
|---|---|---|---|---|
| `time_scale` | float | `1.0` | >= 0 | Simulation speed multiplier. `1.0` is real-time, `2.0` is double speed, `0.5` is half speed. A value of `0` pauses time progression. |
| `random_factor` | float | `0.1` | 0.0 -- 1.0 | Controls the amount of randomness injected into cycle times and other stochastic parameters. `0.0` means fully deterministic; `1.0` means maximum variance. |
| `tick_rate_ms` | int | `100` | > 0 | Simulation tick interval in milliseconds. Lower values increase simulation resolution at the cost of higher CPU usage. |
| `log_level` | string | `"info"` | `"debug"`, `"info"`, `"warn"`, `"error"` | Minimum log level for console output. |

### API Settings (`api`)

| Field | Type | Default | Description |
|---|---|---|---|
| `host` | string | `"0.0.0.0"` | Bind address for the HTTP API server. Use `"0.0.0.0"` to listen on all interfaces or `"127.0.0.1"` for localhost only. |
| `port` | int | `8081` | TCP port for the HTTP API server. |

### ERP Settings (`erp`)

| Field | Type | Default | Valid Values | Description |
|---|---|---|---|---|
| `mode` | string | `"auto"` | `"auto"`, `"manual"`, `"disabled"` | Controls how production orders are generated. `auto` generates orders on a timer. `manual` requires orders to be submitted via the API. `disabled` turns off ERP order generation entirely. |
| `auto_order_interval_ms` | int | `30000` | > 0 | Interval in milliseconds between automatic order generation (only used when `mode` is `"auto"`). |
| `max_backlog` | int | `5` | >= 0 | Maximum number of unstarted orders allowed in the backlog. When this limit is reached, no new orders are generated until existing orders are dispatched. |

### MES Settings (`mes`)

| Field | Type | Default | Valid Values | Description |
|---|---|---|---|---|
| `dispatch_interval_ms` | int | `1000` (built-in default) | > 0 | How often the MES checks for pending orders and dispatches them to lines, in milliseconds. Note: the `global.yaml` shipped with the project sets this to `5000`; the built-in default is `1000`. |
| `dispatch_strategy` | string | `"round-robin"` | `"round-robin"`, `"priority"` | Strategy for assigning orders to lines. `round-robin` distributes evenly across eligible lines. `priority` prefers lines with shorter queues or higher throughput. |

---

## 3. Line Templates

Line templates are YAML files under `config/lines/`, organized into subdirectories by domain. Each template defines a production line's machine stages, inter-stage buffers, and product recipes.

All templates are loaded at startup regardless of which profile is active. The profile determines which templates are instantiated.

### Full Annotated Example

Below is a fully annotated line template using `config/lines/automotive/welding-line.yaml` as the basis:

```yaml
# Template identity
name: "automotive-welding"              # Unique template name, referenced by profiles.
domain: "automotive"                    # Industry domain. If omitted, derived from the
                                        # parent directory name (e.g., "automotive").
description: "Automotive body welding line for car and truck frames"

# Machine stages -- ordered list of processing steps in the line.
# Parts flow sequentially from stage 1 -> 2 -> ... -> N.
machines:
  - stage: 1                            # Stage number (sequential, starting at 1).
    type: "metal_forming"               # Machine type identifier (must be non-empty).
    role: "Stamp and form sheet metal blanks into frame components"
                                        # Human-readable description of what this stage does.
    defaults:                           # Default operating parameters for this stage.
      cycle_time_ms: 12000              # Time to process one part, in milliseconds.
                                        # Falls back to 5000 ms if set to 0 or omitted.
      scrap_rate: 0.03                  # Probability (0.0--1.0) that a finished part is
                                        # rejected as scrap. Default: 0.02.
      breakdown_probability: 0.005      # Probability (0.0--1.0) of a breakdown per tick.
                                        # Default: 0.001.
      auto_restore_time_ms: 45000       # Time before a broken-down machine automatically
                                        # restores itself, in milliseconds.
                                        # Falls back to 60000 ms if set to 0 or omitted.

  - stage: 2
    type: "spot_welder"
    role: "Spot-weld formed panels at structural joints"
    defaults:
      cycle_time_ms: 8000
      scrap_rate: 0.02
      breakdown_probability: 0.004
      auto_restore_time_ms: 30000

  - stage: 3
    type: "robot_welder"
    role: "Seam-weld frame assembly for structural integrity"
    defaults:
      cycle_time_ms: 15000
      scrap_rate: 0.04
      breakdown_probability: 0.006
      auto_restore_time_ms: 60000

  - stage: 4
    type: "robot_pick_place"
    role: "Unload and palletize finished welded frames"
    defaults:
      cycle_time_ms: 10000
      scrap_rate: 0.01
      breakdown_probability: 0.003
      auto_restore_time_ms: 20000

# Buffer configuration -- buffers sit between consecutive stages.
buffers:
  default_capacity: 8                   # Default buffer capacity (number of parts).
                                        # If set to 0 or omitted, falls back to 10.
  overrides:                            # Per-buffer capacity overrides (optional).
    []                                  # An empty list means all buffers use the default.
    # Example override:
    # - from_stage: 2                   # Upstream stage number.
    #   to_stage: 3                     # Downstream stage number.
    #   capacity: 15                    # Override capacity for this specific buffer.

# Recipes -- each recipe represents a product variant that can be manufactured
# on this line. A template must define at least 2 recipes (enforced by validation).
recipes:
  - product_id: "FRAME-WELD-A"         # Unique product identifier.
    description: "Car body frame - sedan/hatchback platform"
    planned_qty_range: [20, 60]         # [min, max] range for order quantities.
                                        # The ERP picks a random value in this range.
    overrides:                          # Per-stage overrides for this specific product.
      3:                                # Key is the stage number.
        cycle_time_ms: 14000            # Override cycle time for stage 3 when making
                                        # this product. Fields not listed here keep
                                        # the stage defaults.

  - product_id: "FRAME-WELD-B"
    description: "Truck body frame - heavy-duty platform"
    planned_qty_range: [10, 30]
    overrides:
      1:                                # Stage 1 takes longer for truck frames.
        cycle_time_ms: 15000
      3:                                # Stage 3 also takes longer and has higher scrap.
        cycle_time_ms: 18000
        scrap_rate: 0.05
```

### Machine Defaults Fallbacks

If a machine stage omits a field or sets it to zero, the following built-in fallbacks are used:

| Field | Fallback Value |
|---|---|
| `cycle_time_ms` | `5000` |
| `scrap_rate` | `0.02` |
| `breakdown_probability` | `0.001` |
| `auto_restore_time_ms` | `60000` |

### Recipe Stage Override Fields

Each recipe override entry (keyed by stage number) supports the following optional fields:

| Field | Type | Description |
|---|---|---|
| `cycle_time_ms` | int | Override cycle time for this product at this stage. |
| `scrap_rate` | float | Override scrap rate for this product at this stage. |
| `breakdown_probability` | float | Override breakdown probability for this product at this stage. |

Fields that are not specified in the override retain the stage's default values.

### Available Line Templates

| Template Name | Domain | File |
|---|---|---|
| `automotive-welding` | automotive | `config/lines/automotive/welding-line.yaml` |
| `automotive-assembly` | automotive | `config/lines/automotive/assembly-line.yaml` |
| `electronics-smt` | electronics | `config/lines/electronics/smt-line.yaml` |
| `electronics-through-hole` | electronics | `config/lines/electronics/through-hole-line.yaml` |
| `food-beverage-filling` | food-beverage | `config/lines/food-beverage/filling-line.yaml` |
| `pharma-batch` | pharma | `config/lines/pharma/batch-line.yaml` |
| `window-frame` | window | `config/lines/window/frame-line.yaml` |
| `furniture-assembly` | furniture | `config/lines/furniture/assembly-line.yaml` |
| `metal-parts-fabrication` | metal-parts | `config/lines/metal-parts/fabrication-line.yaml` |
| `plastic-parts-molding` | plastic-parts | `config/lines/plastic-parts/molding-line.yaml` |

---

## 4. Profiles

A profile selects which line templates to instantiate and how many copies of each. Profiles are stored in `config/profiles/` and selected at startup (e.g., via a command-line flag).

### Profile Structure

```yaml
name: "demo-mixed"                          # Profile name.
description: "Mixed demo with one line per domain"

lines:                                      # List of line template references.
  - template: "automotive-welding"          # Must match a template's `name` field.
    instances: 1                            # Number of independent instances to create.
  - template: "electronics-smt"
    instances: 1

overrides:                                  # Optional global setting overrides.
  global:
    time_scale: 2.0                         # Example: run this profile at 2x speed.
    random_factor: 0.05                     # Example: reduce randomness.
```

### How Instance IDs Are Generated

When a profile requests N instances of a template, each instance receives an ID in the format:

```
<template-name>-<N>
```

For example, if the profile requests 2 instances of `automotive-welding`:

| Instance | Instance ID |
|---|---|
| 1 | `automotive-welding-1` |
| 2 | `automotive-welding-2` |

Machine IDs within each instance follow the pattern `<instance-id>-stage-<stage-number>`:

- `automotive-welding-1-stage-1`
- `automotive-welding-1-stage-2`
- `automotive-welding-2-stage-1`
- ...

Buffer IDs follow the pattern `<instance-id>-buffer-<from-stage>-<to-stage>`:

- `automotive-welding-1-buffer-1-2`
- `automotive-welding-1-buffer-2-3`
- ...

### Profile Overrides

The `overrides.global` map allows a profile to override global settings without modifying `global.yaml`. Currently supported override keys:

| Key | Type | Description |
|---|---|---|
| `time_scale` | float | Override the global `time_scale`. |
| `random_factor` | float | Override the global `random_factor`. |

Profile overrides are applied after `global.yaml` is loaded but before environment variable overrides. This means environment variables always take final precedence.

### Available Profiles

#### `demo-mixed`

A balanced demo with one line from each of four domains.

```yaml
lines:
  - template: "automotive-welding"     (1 instance)
  - template: "electronics-smt"        (1 instance)
  - template: "food-beverage-filling"  (1 instance)
  - template: "pharma-batch"           (1 instance)
```

#### `automotive-focus`

An automotive-focused factory with multiple welding lines and supporting fabrication.

```yaml
lines:
  - template: "automotive-welding"     (2 instances)
  - template: "automotive-assembly"    (1 instance)
  - template: "metal-parts-fabrication" (1 instance)
```

#### `full-catalog`

One instance of every available line template (10 lines total).

```yaml
lines:
  - template: "automotive-welding"       (1 instance)
  - template: "automotive-assembly"      (1 instance)
  - template: "electronics-smt"          (1 instance)
  - template: "electronics-through-hole" (1 instance)
  - template: "food-beverage-filling"    (1 instance)
  - template: "pharma-batch"            (1 instance)
  - template: "window-frame"            (1 instance)
  - template: "furniture-assembly"       (1 instance)
  - template: "metal-parts-fabrication"  (1 instance)
  - template: "plastic-parts-molding"    (1 instance)
```

---

## 5. OPC-UA Adapter Configuration

File: `config/adapters/opcua.yaml`

The OPC-UA adapter exposes each simulated machine as an OPC-UA server. Each machine gets its own OPC-UA endpoint on a separate port.

```yaml
enabled: true
base_port: 4840
update_rate_ms: 1000
max_sessions: 50
hostname: "0.0.0.0"
```

| Field | Type | Default (built-in) | Default (yaml) | Description |
|---|---|---|---|---|
| `enabled` | bool | `true` | `true` | Whether the OPC-UA adapter is active. Set to `false` to disable all OPC-UA endpoints. |
| `base_port` | int | `4840` | `4840` | Starting port number. The first machine gets this port, the second gets `base_port + 1`, and so on. Port 4840 is the IANA-registered default for OPC-UA. |
| `update_rate_ms` | int | `100` | `1000` | How often machine state is pushed to OPC-UA clients, in milliseconds. The built-in default is `100` ms; the shipped `opcua.yaml` sets it to `1000` ms. |
| `max_sessions` | int | `100` | `50` | Maximum number of concurrent OPC-UA client sessions across all endpoints. The built-in default is `100`; the shipped `opcua.yaml` sets it to `50`. |
| `hostname` | string | `"localhost"` | `"0.0.0.0"` | Hostname or IP address for OPC-UA server bindings. Use `"0.0.0.0"` to accept connections from any interface. |

### Port Assignment

Ports are assigned sequentially from `base_port`, one port per machine across all line instances. For example, with `base_port: 4840` and a profile that creates 8 total machines, ports 4840 through 4847 are used.

---

## 6. Modbus Adapter Configuration

File: `config/adapters/modbus.yaml`

The Modbus adapter exposes all simulated machines through a single Modbus TCP gateway. Each machine is addressed by a unique Modbus unit ID.

```yaml
enabled: true
port: 502
max_clients: 10
update_interval_ms: 500
```

| Field | Type | Default (built-in) | Default (yaml) | Description |
|---|---|---|---|---|
| `enabled` | bool | `true` | `true` | Whether the Modbus TCP adapter is active. Set to `false` to disable the Modbus gateway. |
| `port` | int | `502` | `502` | TCP port for the Modbus gateway. Port 502 is the IANA-registered default for Modbus TCP. |
| `max_clients` | int | `10` | `10` | Maximum number of concurrent Modbus TCP client connections. |
| `update_interval_ms` | int | `1000` | `500` | How often machine state is written to Modbus registers, in milliseconds. The built-in default is `1000` ms; the shipped `modbus.yaml` sets it to `500` ms. |

### Unit ID Assignment

Unlike OPC-UA (which uses one port per machine), Modbus uses a single gateway port with unit IDs to distinguish machines. Each machine across all line instances is assigned a unique unit ID.

---

## 7. Environment Variable Overrides

Environment variables provide the highest-priority override mechanism. They are applied last in the loading order, after all file-based configuration and profile overrides.

Set any of the following `SIMULATOR_*` variables to override the corresponding setting:

### Global Settings

| Environment Variable | Overrides | Type |
|---|---|---|
| `SIMULATOR_TIME_SCALE` | `time_scale` | float |
| `SIMULATOR_RANDOM_FACTOR` | `random_factor` | float |
| `SIMULATOR_TICK_RATE_MS` | `tick_rate_ms` | int |
| `SIMULATOR_LOG_LEVEL` | `log_level` | string |

### API Settings

| Environment Variable | Overrides | Type |
|---|---|---|
| `SIMULATOR_API_HOST` | `api.host` | string |
| `SIMULATOR_API_PORT` | `api.port` | int |

### ERP Settings

| Environment Variable | Overrides | Type |
|---|---|---|
| `SIMULATOR_ERP_MODE` | `erp.mode` | string |
| `SIMULATOR_ERP_AUTO_ORDER_INTERVAL_MS` | `erp.auto_order_interval_ms` | int |
| `SIMULATOR_ERP_MAX_BACKLOG` | `erp.max_backlog` | int |

### MES Settings

| Environment Variable | Overrides | Type |
|---|---|---|
| `SIMULATOR_MES_DISPATCH_INTERVAL_MS` | `mes.dispatch_interval_ms` | int |
| `SIMULATOR_MES_DISPATCH_STRATEGY` | `mes.dispatch_strategy` | string |

### OPC-UA Adapter

| Environment Variable | Overrides | Type |
|---|---|---|
| `SIMULATOR_OPCUA_ENABLED` | `opcua.enabled` | bool (`"true"` or `"1"` to enable) |
| `SIMULATOR_OPCUA_BASE_PORT` | `opcua.base_port` | int |
| `SIMULATOR_OPCUA_UPDATE_RATE_MS` | `opcua.update_rate_ms` | int |
| `SIMULATOR_OPCUA_MAX_SESSIONS` | `opcua.max_sessions` | int |
| `SIMULATOR_OPCUA_HOSTNAME` | `opcua.hostname` | string |

### Modbus Adapter

| Environment Variable | Overrides | Type |
|---|---|---|
| `SIMULATOR_MODBUS_ENABLED` | `modbus.enabled` | bool (`"true"` or `"1"` to enable) |
| `SIMULATOR_MODBUS_PORT` | `modbus.port` | int |
| `SIMULATOR_MODBUS_MAX_CLIENTS` | `modbus.max_clients` | int |
| `SIMULATOR_MODBUS_UPDATE_INTERVAL_MS` | `modbus.update_interval_ms` | int |

**Note:** The `env` struct tags in `internal/config/types.go` document the intended mapping for all environment variables. However, only the variables listed in the `applyEnvOverrides` function in `internal/config/loader.go` are actually read at runtime. The following variables are declared in struct tags but **not currently implemented** in the loader: `SIMULATOR_TICK_RATE_MS`, `SIMULATOR_LOG_LEVEL`, `SIMULATOR_ERP_MAX_BACKLOG`, `SIMULATOR_MES_DISPATCH_INTERVAL_MS`, `SIMULATOR_OPCUA_BASE_PORT`, `SIMULATOR_OPCUA_UPDATE_RATE_MS`, `SIMULATOR_OPCUA_MAX_SESSIONS`, `SIMULATOR_OPCUA_HOSTNAME`, `SIMULATOR_MODBUS_PORT`, `SIMULATOR_MODBUS_MAX_CLIENTS`, `SIMULATOR_MODBUS_UPDATE_INTERVAL_MS`.

---

## 8. Creating Custom Profiles

Follow these steps to create a new factory profile.

### Step 1: Choose Your Line Templates

Review the available templates in `config/lines/`. Each template represents a type of production line. You can list them with:

```bash
ls config/lines/**/*.yaml
```

Refer to the [Available Line Templates](#available-line-templates) table above for the template names.

### Step 2: Create the Profile File

Create a new YAML file in `config/profiles/`. The filename (without extension) becomes the profile name used at startup.

```bash
touch config/profiles/my-custom.yaml
```

### Step 3: Define the Profile

Write the profile YAML. At minimum, you need a `name`, `description`, and at least one line entry:

```yaml
name: "my-custom"
description: "Custom factory for testing purposes"

lines:
  - template: "automotive-welding"
    instances: 3                        # Three parallel welding lines.
  - template: "electronics-smt"
    instances: 2                        # Two SMT lines.
  - template: "pharma-batch"
    instances: 1                        # One pharmaceutical batch line.

overrides: {}                           # No global overrides (uses global.yaml as-is).
```

### Step 4: Add Global Overrides (Optional)

If this profile needs different global settings, add them under `overrides.global`:

```yaml
overrides:
  global:
    time_scale: 5.0                     # Run at 5x speed for this profile.
    random_factor: 0.0                  # Deterministic mode for repeatable testing.
```

Currently, the profile override mechanism supports `time_scale` and `random_factor`. For other global settings, use environment variables.

### Step 5: Validate

Start the simulator with your new profile to confirm it loads without errors. The loader validates that:

- All referenced templates exist.
- Each template has at least 2 recipes.
- All machine types are non-empty.
- No duplicate machine IDs are generated across instances.
- Numeric values are within valid ranges.

### Step 6: Create Custom Line Templates (Optional)

If none of the existing templates fit your needs, create a new line template:

1. Create a new subdirectory under `config/lines/` for your domain (or use an existing one):
   ```bash
   mkdir -p config/lines/custom-domain
   ```

2. Create a template YAML file:
   ```bash
   touch config/lines/custom-domain/my-line.yaml
   ```

3. Define the template following the structure documented in [Line Templates](#3-line-templates). Requirements:
   - `name` must be unique across all templates.
   - `machines` must have at least one stage with a non-empty `type`.
   - `recipes` must contain at least 2 entries.
   - Stage numbers should be sequential starting from 1.

4. Reference the new template by its `name` field in your profile's `lines` list.
