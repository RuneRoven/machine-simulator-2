## ADDED Requirements

### Requirement: Recipe definition within line templates
Each line template SHALL define one or more product recipes in its YAML file under the `recipes` key. Each recipe SHALL specify a `product_id`, `description`, `planned_qty_range` (min/max), and optional per-stage `overrides` for machine parameters.

#### Scenario: Line template with multiple recipes
- **WHEN** the `automotive-welding` template defines recipes for `DOOR-PANEL-A` and `FENDER-B`
- **THEN** both products are available for orders on any instance of that line type

#### Scenario: Recipe with per-stage overrides
- **WHEN** recipe `DOOR-PANEL-A` specifies `overrides: { 1: { cycle_time_ms: 4500 }, 3: { scrap_rate: 0.02 } }`
- **THEN** stage 1 uses 4500ms cycle time and stage 3 uses 2% scrap rate when processing this product, while other stages use the line template defaults

#### Scenario: Recipe with planned quantity range
- **WHEN** recipe `FENDER-B` specifies `planned_qty_range: [30, 100]`
- **THEN** auto-generated orders for this product have a planned quantity randomly selected between 30 and 100 (inclusive)

### Requirement: Recipe overrides applied during production
When MES assigns an order for a specific product to a line instance, the system SHALL apply that recipe's per-stage overrides to the corresponding machines. When the order completes or is cancelled, machines SHALL revert to their line template defaults.

#### Scenario: Apply recipe overrides on order start
- **WHEN** MES starts order for product `DOOR-PANEL-A` on `automotive-welding-1`
- **THEN** machine `automotive-welding-1-stage-1` cycle time changes to 4500ms and `automotive-welding-1-stage-3` scrap rate changes to 0.02

#### Scenario: Revert overrides on order completion
- **WHEN** the order for `DOOR-PANEL-A` on `automotive-welding-1` closes
- **THEN** all machines on that line instance revert to their line template default cycle times and scrap rates

#### Scenario: Override only specified parameters
- **WHEN** a recipe override for stage 2 specifies only `cycle_time_ms: 6000`
- **THEN** stage 2 cycle time changes to 6000ms but scrap rate and breakdown probability remain at line template defaults

### Requirement: Recipe parameters available via API
The REST API SHALL expose recipe information for each active line instance, including the full recipe list and current active recipe (if an order is in progress).

#### Scenario: List recipes for a line instance
- **WHEN** a client sends `GET /api/lines/automotive-welding-1/recipes`
- **THEN** the response includes all recipes defined in the `automotive-welding` template with their product IDs, descriptions, and override summaries

#### Scenario: Show active recipe
- **WHEN** a client sends `GET /api/lines/automotive-welding-1` while an order for `DOOR-PANEL-A` is in progress
- **THEN** the response includes the active product ID and the effective machine parameters (with overrides applied)

### Requirement: Each line template has at least 2 recipes
Every line template in the catalog SHALL define a minimum of 2 product recipes to demonstrate product variety and recipe switching.

#### Scenario: Template with only 1 recipe
- **WHEN** a line template YAML has only 1 recipe entry
- **THEN** config validation fails at startup with an error indicating the minimum recipe count
