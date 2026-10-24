## MODIFIED Requirements

### Requirement: FIFO buffers between machines
Consecutive machines in a line SHALL be connected by FIFO buffers with configurable capacity. An upstream machine SHALL be blocked from completing a cycle when the downstream buffer is full. A downstream machine SHALL wait when the upstream buffer is empty. Buffers SHALL NOT discard parts: a part leaves a buffer only when a downstream machine pulls it, when it is carried over to a following order, or when it is scrapped as work in progress, and each of these SHALL be counted.

#### Scenario: Part flows through buffer
- **WHEN** machine at stage 1 produces a good part and the buffer to stage 2 is not full
- **THEN** the part is added to the buffer and machine at stage 2 can pull it when ready

#### Scenario: Upstream blocked by full buffer
- **WHEN** the buffer between stage 1 and stage 2 reaches max capacity
- **THEN** stage 1 machine is blocked and cannot complete its current cycle until space opens

#### Scenario: Downstream waits on empty buffer
- **WHEN** the buffer between stage 1 and stage 2 is empty and stage 2 finishes its current part
- **THEN** stage 2 waits for a part, its `starved` flag is set, and it resumes when a part appears

#### Scenario: No part is lost on a full buffer
- **WHEN** a machine completes a good part and its out-buffer is full
- **THEN** the part stays at the machine (the machine is blocked) and is not discarded

### Requirement: Deadlock detection and recovery
The line manager SHALL detect a stalled line: a job is RUNNING or INTERRUPTED, there is work (a part in a machine or stage 1 may release), no machine is FAULT or MAINTENANCE (a fault restores itself), and no part has moved for `deadlock_threshold_ms` of simulated time. Recovery SHALL restart idle or stopped machines that hold parts, including stage 1, and SHALL NOT clear buffers.

#### Scenario: Normal fault is not a deadlock
- **WHEN** a machine on a line is FAULT for up to its `auto_restore_time_ms`
- **THEN** no deadlock is detected and no recovery runs

#### Scenario: Stalled line recovered without losing parts
- **WHEN** a line meets the stall condition for `deadlock_threshold_ms`
- **THEN** all STOPPED or IDLE machines that hold a part are started (stage 1 included), buffer contents are unchanged, and a `line.deadlock_recovered` event is published

## ADDED Requirements

### Requirement: Part conservation per line
Each line SHALL keep a ledger of parts for the current order: started at stage 1, delivered good at the last stage, scrapped per stage, carried over, and scrapped as work in progress. At any time `started + carried_in` SHALL equal `delivered_good + scrapped + in_buffers + in_machines + carried_out + wip_scrapped`.

#### Scenario: Ledger balances after an order
- **WHEN** an order completes on a 4-stage line after faults and downstream scrap
- **THEN** the ledger equation holds and every part started is accounted for

#### Scenario: Leftover parts of the same product are carried over
- **WHEN** an order closes with parts still in the line and the next queued order is for the same product
- **THEN** those parts are assigned to the next order and counted as `wip_carried_over`

#### Scenario: Carry-over limited to what the next order needs
- **WHEN** 5 parts of the same product are left over and the next order needs 2
- **THEN** 2 parts are carried over and 3 are counted as `wip_scrapped`

#### Scenario: Leftover parts of another product are scrapped
- **WHEN** an order closes with parts still in the line and the next order is for a different product
- **THEN** those parts are removed at dispatch of the next order and counted as `wip_scrapped`

### Requirement: Stage-1 release limited by the order
The first stage of a line SHALL start a new part only while the order's good parts delivered at the last stage plus its parts in process are fewer than the planned quantity.

#### Scenario: No over-production
- **WHEN** an order for 100 parts has 95 good delivered and 5 in process
- **THEN** stage 1 does not start another part

#### Scenario: Re-start after downstream scrap
- **WHEN** a part of that order is scrapped at stage 3
- **THEN** stage 1 may start one more part
