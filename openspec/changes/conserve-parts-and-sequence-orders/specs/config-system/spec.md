## MODIFIED Requirements

### Requirement: Global configuration settings
`config/global.yaml` SHALL define simulator-wide settings including: time_scale, random_factor, tick_rate_ms, log level, API host/port, ERP mode and intervals, MES dispatch interval and strategy, and `deadlock_threshold_ms`. The database path SHALL be set by `SIMULATOR_DB_PATH`.

#### Scenario: Default global config
- **WHEN** no overrides are applied
- **THEN** defaults are: time_scale=1.0, random_factor=0.1, tick_rate_ms=100, log_level=info, api_host=0.0.0.0, api_port=8081, database path `./data/simulator.db`, and deadlock_threshold_ms derived as the longest `auto_restore_time_ms` plus 60 s, at least 120 s

#### Scenario: ERP configuration in global
- **WHEN** `global.yaml` sets `erp.mode: auto` and `erp.auto_order_interval_ms: 30000`
- **THEN** ERP generates orders automatically every 30 seconds

#### Scenario: Deadlock threshold validated
- **WHEN** `deadlock_threshold_ms` is not greater than the largest `auto_restore_time_ms` of any active machine
- **THEN** startup fails with a validation error naming the machine and both values

#### Scenario: Database survives rebuild
- **WHEN** the container is rebuilt with `./data` mounted
- **THEN** orders and operations from before the rebuild are still present
