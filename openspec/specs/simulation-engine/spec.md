## ADDED Requirements

### Requirement: Machine state machine
Each machine SHALL implement a state machine with 5 states: IDLE, RUNNING, STOPPED, FAULT, and MAINTENANCE. State transitions SHALL be validated — only permitted transitions are allowed.

#### Scenario: Valid state transitions
- **WHEN** a machine is in IDLE state and receives a START command
- **THEN** the machine transitions to RUNNING

#### Scenario: Invalid state transition
- **WHEN** a machine is in FAULT state and receives a START command
- **THEN** the command is rejected with an error (machine must RESET first to return to IDLE)

#### Scenario: All valid transitions
- **WHEN** state transitions are attempted
- **THEN** the following transitions are permitted: IDLE→RUNNING, RUNNING→IDLE, RUNNING→STOPPED, RUNNING→FAULT, RUNNING→MAINTENANCE, STOPPED→IDLE, FAULT→IDLE, MAINTENANCE→IDLE

### Requirement: Tick-based cycle simulation
Each machine SHALL run in its own goroutine with a tick-based loop (default 100ms tick rate). During RUNNING state, the machine SHALL simulate production cycles with configurable cycle time and stochastic variance.

#### Scenario: Cycle completion
- **WHEN** a machine in RUNNING state accumulates ticks equal to its effective cycle time
- **THEN** a part is produced and the cycle counter increments

#### Scenario: Time scaling affects cycle time
- **WHEN** time scale is set to 10x and base cycle time is 5000ms
- **THEN** effective cycle time is 500ms (5000 / 10)

#### Scenario: Random variance on cycle time
- **WHEN** random factor is 0.15 and base cycle time is 5000ms
- **THEN** each cycle's actual duration varies randomly within ±15% of the base (4250ms to 5750ms before time scaling)

### Requirement: Breakdown and fault simulation
Machines SHALL simulate random breakdowns during production. Each cycle completion has a configurable probability of triggering a FAULT state. Faulted machines SHALL auto-recover after a configurable delay.

#### Scenario: Random breakdown occurs
- **WHEN** a machine completes a cycle and the random check (against `breakdown_probability`) triggers
- **THEN** the machine transitions to FAULT state and stops producing

#### Scenario: Auto-recovery from fault
- **WHEN** a machine has been in FAULT state for longer than `auto_restore_time_ms`
- **THEN** the machine transitions back to IDLE and can be restarted

#### Scenario: Breakdown probability of zero
- **WHEN** `breakdown_probability` is set to 0.0
- **THEN** the machine never randomly enters FAULT state

### Requirement: Scrap simulation
Each completed part SHALL be marked as good or scrap based on the machine's `scrap_rate`. Scrap parts are counted but do not advance through the production line.

#### Scenario: Good part produced
- **WHEN** a part is completed and the random check passes (part is good)
- **THEN** the part is marked as good, `good_qty` increments, and the part moves to the downstream buffer

#### Scenario: Scrap part produced
- **WHEN** a part is completed and the random check fails (part is scrap)
- **THEN** the part is discarded, `scrap_qty` increments, and no part enters the downstream buffer

### Requirement: FIFO buffers between machines
Consecutive machines in a line SHALL be connected by FIFO buffers with configurable capacity. An upstream machine SHALL be blocked from completing a cycle when the downstream buffer is full. A downstream machine SHALL wait when the upstream buffer is empty.

#### Scenario: Part flows through buffer
- **WHEN** machine at stage 1 produces a good part and the buffer to stage 2 is not full
- **THEN** the part is added to the buffer and machine at stage 2 can pull it when ready

#### Scenario: Upstream blocked by full buffer
- **WHEN** the buffer between stage 1 and stage 2 reaches max capacity
- **THEN** stage 1 machine is blocked and cannot complete its current cycle until space opens

#### Scenario: Downstream waits on empty buffer
- **WHEN** the buffer between stage 1 and stage 2 is empty and stage 2 finishes its current part
- **THEN** stage 2 transitions to IDLE and waits for a part to appear in the buffer

### Requirement: Time scaling
The simulation engine SHALL support a global time scale factor (1x to 100x) that accelerates all machine cycle times, fault recovery times, and ERP/MES intervals proportionally.

#### Scenario: Change time scale at runtime
- **WHEN** the time scale is changed from 1x to 50x via API
- **THEN** all active machines adjust their effective cycle times immediately (ongoing cycles are not interrupted but new cycles use the updated scale)

#### Scenario: Time scale affects all timed operations
- **WHEN** time scale is 10x
- **THEN** cycle times, auto-restore times, and ERP order generation intervals are all divided by 10

### Requirement: Engine lifecycle management
The simulation engine SHALL support Start and Stop operations. Start SHALL launch all registered machines. Stop SHALL gracefully shut down all machines using context cancellation.

#### Scenario: Start simulation
- **WHEN** `engine.Start(ctx)` is called
- **THEN** all registered machines start their goroutines and begin responding to commands

#### Scenario: Stop simulation
- **WHEN** `engine.Stop()` is called
- **THEN** all machine goroutines receive context cancellation and shut down within 5 seconds

### Requirement: Deadlock detection and recovery
The line manager SHALL detect deadlocked lines where all machines are STOPPED, buffers contain parts, and no machine is running or faulted. Recovery SHALL clear buffers and reset stopped machines to IDLE.

#### Scenario: Deadlock detected
- **WHEN** all machines on a line instance are in STOPPED state and at least one buffer is non-empty
- **THEN** the system detects a deadlock condition

#### Scenario: Automatic deadlock recovery
- **WHEN** a deadlock is detected on a line instance
- **THEN** all buffers on that line are cleared, all STOPPED machines are reset to IDLE, and a `line.deadlock_recovered` event is published
