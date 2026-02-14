## ADDED Requirements

### Requirement: Line template definition
The system SHALL provide a library of pre-configured production line templates, each defined as a YAML file in `config/lines/<domain>/<template>.yaml`. Each template SHALL declare an ordered sequence of machine stages, buffer configuration between stages, supported product recipes, and default operating parameters.

#### Scenario: Load a line template from YAML
- **WHEN** the config loader reads `config/lines/automotive/welding-line.yaml`
- **THEN** the system creates a LineTemplate with the name, domain, description, ordered machine stages (each with type, role, and default settings), buffer configuration, and recipe list

#### Scenario: Template defines machine stages in order
- **WHEN** a line template specifies stages 1 through N
- **THEN** each stage SHALL have a `type` (matching a registered machine type), a `role` (human-readable label), and `defaults` (cycle_time_ms, scrap_rate, breakdown_probability)

#### Scenario: Template defines buffer configuration
- **WHEN** a line template specifies `buffers.default_capacity` and optional `overrides`
- **THEN** buffers between consecutive stages use the default capacity unless an override specifies a different capacity for a specific stage pair

### Requirement: Line catalog covers 8 industry domains
The system SHALL include line templates spanning 8 industry domains: automotive, electronics, food & beverage, pharma/chemical, window manufacturing, furniture, metal parts, and plastic parts. Some domains SHALL have 2 template variants for a total of 10-12 templates.

#### Scenario: Automotive domain has two line types
- **WHEN** the user lists available line templates
- **THEN** the automotive domain includes at least a welding line and an assembly line

#### Scenario: Electronics domain has two line types
- **WHEN** the user lists available line templates
- **THEN** the electronics domain includes at least an SMT line and a through-hole assembly line

#### Scenario: Single-variant domains
- **WHEN** the user lists available line templates
- **THEN** food & beverage, pharma, window, furniture, metal parts, and plastic parts each have at least one line template

### Requirement: Line template instantiation
The system SHALL create concrete line instances from templates. Each instance receives a unique ID formed as `<template-name>-<instance-number>` (e.g., `automotive-welding-1`). Machines within an instance SHALL have IDs of the form `<instance-id>-stage-<N>`.

#### Scenario: Instantiate a line template once
- **WHEN** the profile requests 1 instance of `automotive-welding`
- **THEN** the system creates line instance `automotive-welding-1` with machines `automotive-welding-1-stage-1` through `automotive-welding-1-stage-N` and FIFO buffers between consecutive stages

#### Scenario: Instantiate a line template multiple times
- **WHEN** the profile requests 3 instances of `electronics-smt`
- **THEN** the system creates `electronics-smt-1`, `electronics-smt-2`, and `electronics-smt-3`, each with fully independent machines, buffers, and state

### Requirement: Line templates are self-contained
Each line template YAML file SHALL contain all information needed to instantiate the line, including machine types, buffer sizes, and product recipes. No external references beyond registered machine type names SHALL be required.

#### Scenario: Template with unknown machine type
- **WHEN** a line template references a machine type that is not in the registry
- **THEN** the system SHALL fail at startup with a clear error message naming the unknown type and the template file
