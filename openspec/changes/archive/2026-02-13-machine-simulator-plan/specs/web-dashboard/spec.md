## ADDED Requirements

### Requirement: Server-rendered htmx web UI
The web dashboard SHALL be built with server-rendered HTML templates and htmx for dynamic updates. No JavaScript framework build step SHALL be required.

#### Scenario: Dashboard loads without JS build
- **WHEN** the user navigates to `http://localhost:8081/`
- **THEN** the page renders fully with server-side templates, and htmx handles dynamic updates via HTML-over-the-wire

### Requirement: Dashboard overview page
The root page SHALL display an overview of all active line instances with their current status: machine states, buffer levels, active order, and product being manufactured.

#### Scenario: Dashboard shows active lines
- **WHEN** the profile activates 4 line instances
- **THEN** the dashboard displays 4 line cards, each showing the line name, domain, machine count, and overall status (running/idle/faulted)

#### Scenario: Line card shows buffer levels
- **WHEN** a line instance has 3 buffers
- **THEN** the line card displays each buffer's current fill level as a visual indicator (e.g., progress bar) with count/capacity

#### Scenario: Real-time updates via WebSocket
- **WHEN** a machine changes state
- **THEN** the dashboard updates within 1 second without a full page reload

### Requirement: Line detail page
Each line instance SHALL have a detail page showing the full machine sequence, buffer states, active order progress, and current recipe information.

#### Scenario: Navigate to line detail
- **WHEN** the user clicks on a line card for `automotive-welding-1`
- **THEN** the page shows all machines in stage order with their state, cycle count, good/scrap counts, and the buffers between them

#### Scenario: Active recipe displayed
- **WHEN** the line is processing an order for product `DOOR-PANEL-A`
- **THEN** the line detail page shows the product ID, recipe description, order progress (good_qty / planned_qty), and which per-stage overrides are active

### Requirement: Machine detail page
Each machine SHALL have a detail page showing all tags, current values, state history, and controls.

#### Scenario: View machine tags
- **WHEN** the user navigates to `/machines/<id>`
- **THEN** all tag definitions are displayed with current values, updated in real time

#### Scenario: Send machine command
- **WHEN** the user clicks START, STOP, RESET, or FAULT button on a machine detail page
- **THEN** the command is sent to the machine and the state updates in the UI

### Requirement: Order management page
The web UI SHALL provide a page to view all orders with filtering by status and line instance, and a form to create manual orders.

#### Scenario: Filter orders by line instance
- **WHEN** the user selects `automotive-welding-1` from the line filter
- **THEN** only orders targeting that line instance are displayed

#### Scenario: Create manual order via UI
- **WHEN** the user fills in the order form with a line instance and product (from a dropdown of available recipes) and submits
- **THEN** a new order is created and appears in the order list

### Requirement: Simulation control page
The web UI SHALL provide controls for adjusting the simulation time scale and random factor at runtime.

#### Scenario: Change time scale via UI
- **WHEN** the user adjusts the time scale slider to 10x and confirms
- **THEN** the simulation time scale changes to 10x and all machines adjust their effective cycle times

### Requirement: Profile and line info display
The web UI SHALL display the active profile name and the list of available recipes per line instance, so users understand the current factory configuration.

#### Scenario: Show active profile
- **WHEN** the user views the dashboard
- **THEN** the active profile name is displayed in the header or sidebar

#### Scenario: Show available products per line
- **WHEN** the user views a line detail page
- **THEN** the available recipes for that line type are listed with product IDs and descriptions
