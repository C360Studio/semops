# SemLink Hold-Out Review Request

Status: ready for SemLink review.

Scope: final SemLink hold-out review before SemOps accepts the MVP native companion readback contract.

## Review Inputs

SemLink should review the pushed `codex/cop-component-telemetry` branch, starting with these artifacts:

- `docs/contracts/semlink-companion-readback-v0.md`
- `testdata/contracts/semlink-companion-readback-v0/request.accepted.json`
- `testdata/contracts/semlink-companion-readback-v0/response.accepted.json`
- `testdata/contracts/semlink-companion-readback-v0/response.duplicate.json`
- `testdata/contracts/semlink-companion-readback-v0/response.expired.json`
- `testdata/contracts/semlink-companion-readback-v0/request.rejected-unsupported-message.json`
- `testdata/contracts/semlink-companion-readback-v0/response.rejected-unsupported-message.json`
- `testdata/contracts/semlink-companion-readback-v0/csapi-projection.accepted.json`
- `internal/api/cop/handler_test.go`
- `internal/egress/csapi/semlink_projection_fixture_test.go`

## Questions For SemLink

1. Can SemLink implement the MVP native companion adapter by POSTing the v0 request fixture shape to SemOps without
   CS API in the boat-local hot path?
2. Can SemLink populate `companion_node_id`, `target_system_id`, `target_component_id`, `command_id=512`,
   `requested_message_id=148`, `correlation_id`, `idempotency_key`, `requested_at`, `ttl_seconds`, and `source_ref`
   from companion runtime state without adding awkward duplicate state?
3. Are the response fields enough for SemLink to correlate accepted, duplicate, expired, and unsupported-message
   outcomes without SemLink minting SemOps trusted operator posture?
4. Does the contract keep `target_asset_id` and born-target proof on the SemOps side clearly enough?
5. Does the CS API projection fixture preserve the facts SemLink expects SemOps/SemConnect to expose at the standards
   edge without implying a SemLink CS API runtime dependency?
6. Which fields, if any, add companion-runtime friction or duplicate MAVLink/BlueOS concepts?
7. Which missing fields, if any, are blocking for a focused optional SemLink adapter slice?

## Classification Requested

SemLink should classify each item as one of:

- Blocking: SemLink cannot implement the MVP adapter safely or correctly until SemOps changes it.
- Friction: SemLink can implement it, but the shape adds avoidable runtime complexity.
- Interop: the native hot path is workable, but CS API/SemConnect projection loses or distorts a fact.
- Deferred: valid concern, but not required for the MVP adapter.

## Boundaries During Review

SemLink can resume a focused optional adapter slice against the fixtures, but should not expand into:

- hardware transmit;
- CS API-first SemLink runtime dependency;
- SemOps trusted-operator header minting inside SemLink;
- SemLink-owned `target_asset_id` or born-target proof;
- generic robotics command protocol work;
- durable multi-replica idempotency; or
- raw MAVLink over CS API.

## Expected SemOps Disposition

SemOps will record SemLink feedback under this change's `reviews/` folder, then either update the contract, mark a
non-goal, or create follow-up tasks for each blocking or friction item before contract acceptance.
