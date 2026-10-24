## ADDED Requirements

### Requirement: Line job endpoints
The REST API SHALL expose the line job controller: `GET /api/lines/{id}/job`, `PUT /api/lines/{id}/job` (load), and `POST /api/lines/{id}/job/start`, `/interrupt`, `/finish`. Responses SHALL return the job and its state; illegal transitions SHALL return 409 and unknown products 400.

#### Scenario: Read the job
- **WHEN** `GET /api/lines/automotive-welding-1/job`
- **THEN** the response holds order_id, operation_id, product_id, target_qty, state, good, scrap, started and parts_in_process

#### Scenario: Illegal transition
- **WHEN** `POST /api/lines/automotive-welding-1/job/start` is sent while the state is NO_JOB
- **THEN** the response is 409 with the current state
