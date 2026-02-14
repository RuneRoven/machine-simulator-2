## ADDED Requirements

### Requirement: Multi-file YAML configuration
The system SHALL load configuration from multiple YAML files organized by concern: `config/global.yaml` for simulator-wide settings, `config/lines/<domain>/<template>.yaml` for line templates, `config/profiles/<name>.yaml` for factory profiles, and `config/adapters/<protocol>.yaml` for protocol adapters.

#### Scenario: All config directories exist
- **WHEN** the simulator starts
- **THEN** the config loader reads from `config/global.yaml`, `config/lines/`, `config/profiles/`, and `config/adapters/`

#### Scenario: Line template auto-discovery
- **WHEN** the config loader scans `config/lines/`
- **THEN** all `*.yaml` files in subdirectories are loaded as line templates, using the subdirectory name as the domain

### Requirement: Configuration precedence
Configuration values SHALL be resolved in this order (highest precedence first): CLI flags → environment variables → profile overrides → global.yaml → built-in defaults.

#### Scenario: Environment variable overrides YAML
- **WHEN** `global.yaml` sets `time_scale: 1.0` and environment variable `SIMULATOR_TIME_SCALE=10.0` is set
- **THEN** the effective time scale is 10.0

#### Scenario: CLI flag overrides everything
- **WHEN** `--time-scale 50` is provided as a CLI flag
- **THEN** the effective time scale is 50 regardless of YAML or environment variable values

#### Scenario: Profile overrides global defaults
- **WHEN** `global.yaml` sets `random_factor: 0.1` and the active profile sets `overrides.global.random_factor: 0.2`
- **THEN** the effective random factor is 0.2

### Requirement: Global configuration settings
`config/global.yaml` SHALL define simulator-wide settings including: time_scale, random_factor, tick_rate_ms, log level, API host/port, ERP mode and intervals, MES dispatch interval and strategy.

#### Scenario: Default global config
- **WHEN** no overrides are applied
- **THEN** defaults are: time_scale=1.0, random_factor=0.1, tick_rate_ms=100, log_level=info, api_host=0.0.0.0, api_port=8081

#### Scenario: ERP configuration in global
- **WHEN** `global.yaml` sets `erp.mode: auto` and `erp.auto_order_interval_ms: 30000`
- **THEN** ERP generates orders automatically every 30 seconds

### Requirement: Configuration validation at startup
The system SHALL validate the complete assembled configuration before starting the simulation. Validation SHALL check for: unknown machine types, duplicate IDs, missing required fields, invalid value ranges, and profile-template consistency.

#### Scenario: Duplicate machine ID detected
- **WHEN** two line instances would produce machines with the same ID
- **THEN** startup fails with an error identifying the duplicate IDs and the line instances that caused them

#### Scenario: Invalid scrap rate value
- **WHEN** a line template specifies `scrap_rate: 1.5` (above 1.0)
- **THEN** startup fails with a validation error for that field

#### Scenario: Missing global.yaml
- **WHEN** `config/global.yaml` does not exist
- **THEN** the system uses built-in defaults for all global settings and logs a warning

### Requirement: Environment variable naming convention
Environment variables SHALL follow the pattern `SIMULATOR_<SECTION>_<KEY>` using uppercase with underscores. Nested keys use additional underscores.

#### Scenario: Nested config via environment
- **WHEN** `SIMULATOR_ERP_AUTO_ORDER_INTERVAL_MS=15000` is set
- **THEN** the ERP auto-order interval is 15000ms

#### Scenario: Boolean environment variable
- **WHEN** `SIMULATOR_MODBUS_ENABLED=false` is set
- **THEN** the Modbus adapter is disabled regardless of YAML config
