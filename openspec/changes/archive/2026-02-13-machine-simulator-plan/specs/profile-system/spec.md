## ADDED Requirements

### Requirement: Profile YAML definition
The system SHALL support named profile files at `config/profiles/<name>.yaml`. Each profile SHALL declare which line templates to activate and how many instances of each. Profiles MAY include global setting overrides.

#### Scenario: Profile activates multiple line types
- **WHEN** the profile `demo-mixed.yaml` specifies `automotive-welding: 1`, `electronics-smt: 1`, `food-filling: 1`, `pharma-batch: 1`
- **THEN** the system instantiates exactly those 4 lines with 1 instance each

#### Scenario: Profile with multiple instances of one type
- **WHEN** the profile specifies `automotive-welding: 2`
- **THEN** the system creates `automotive-welding-1` and `automotive-welding-2` as independent line instances

#### Scenario: Profile with global overrides
- **WHEN** the profile includes `overrides.global.time_scale: 10.0`
- **THEN** the simulator starts with a time scale of 10.0 instead of the default from `global.yaml`

### Requirement: Profile selection at startup
The simulator SHALL accept a profile name via `--profile <name>` CLI flag or `SIMULATOR_PROFILE` environment variable. The CLI flag SHALL take precedence over the environment variable.

#### Scenario: Select profile via CLI flag
- **WHEN** the user runs `./simulator --profile demo-mixed`
- **THEN** the system loads `config/profiles/demo-mixed.yaml` and activates the lines declared in it

#### Scenario: Select profile via environment variable
- **WHEN** `SIMULATOR_PROFILE=automotive-focus` is set and no `--profile` flag is provided
- **THEN** the system loads `config/profiles/automotive-focus.yaml`

#### Scenario: CLI flag overrides environment variable
- **WHEN** `SIMULATOR_PROFILE=demo-mixed` is set AND `--profile automotive-focus` is provided
- **THEN** the system loads `automotive-focus.yaml` (CLI wins)

#### Scenario: No profile specified
- **WHEN** neither `--profile` nor `SIMULATOR_PROFILE` is provided
- **THEN** the system SHALL fail at startup with a message listing available profiles

### Requirement: Profile validation at startup
The system SHALL validate the profile before starting simulation. Validation SHALL fail fast with clear error messages.

#### Scenario: Profile references unknown template
- **WHEN** a profile references line template `nonexistent-line`
- **THEN** the system fails at startup with an error naming the unknown template and available templates

#### Scenario: Profile requests zero instances
- **WHEN** a profile specifies `automotive-welding: 0`
- **THEN** the system treats that line as not activated (skips it)

#### Scenario: Profile is empty
- **WHEN** a profile has no line entries
- **THEN** the system fails at startup with an error indicating no lines are configured

### Requirement: Default profiles shipped with the simulator
The system SHALL ship with at least 3 pre-built profiles: a mixed-domain demo profile (3-4 different domains), a single-domain focused profile, and a full-catalog profile that activates one instance of every line template.

#### Scenario: Demo mixed profile exists
- **WHEN** the user lists `config/profiles/`
- **THEN** a `demo-mixed.yaml` profile is present activating 3-4 lines from different domains

#### Scenario: Full catalog profile exists
- **WHEN** the user activates `full-catalog` profile
- **THEN** one instance of every available line template is started
