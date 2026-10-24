## ADDED Requirements

### Requirement: Line OPC UA server with writable job nodes
The OPC UA adapter SHALL start one server per line instance on `line_base_port + index` (default 4900) exposing the job nodes defined in the design (NodeIDs 1–16). Nodes 1–5 SHALL be writable setpoints that the server never overwrites; writes to other nodes SHALL be refused with `BadUserAccessDenied`. A load SHALL use the setpoints as they were when `job_command` was written. Every write to `job_command` SHALL be handled, even when the value is unchanged, and the node SHALL keep the written value so a read-back check succeeds; `job_command_result` SHALL hold `OK` or the rejection reason.

#### Scenario: Load and start over OPC UA
- **WHEN** a client writes order, operation, product and target to nodes 1–4, then `job_command=1`, then `job_command=2`
- **THEN** `job_state` reads RUNNING, `job_command` reads 2 and `job_command_result` reads OK

#### Scenario: Read-only node write refused
- **WHEN** a client writes to `job_good`
- **THEN** the write returns `BadUserAccessDenied` and the value is unchanged

#### Scenario: Machine servers unchanged
- **WHEN** line servers are enabled
- **THEN** machine servers keep their ports and NodeIDs
