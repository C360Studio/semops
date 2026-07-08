# SemLink Initial Feedback Disposition

Scope: SemLink review feedback on the `semlink-semops-contract-spike` direction before handler migration.

Decision: accept the feedback as conditional approval, with blocking/friction items dispositioned into the v0 contract
draft and handler follow-up work.

## Summary

SemLink can implement the MVP native companion contract without CS API in the hot path, but implementation should not
resume until SemOps publishes the field-level contract and fixtures and SemLink has a chance to review them.

CS API/SemConnect score: Amber. It remains the correct SemOps interoperability and projection edge. The boat-local
SemLink hot path stays native for MVP.

## Blocking Feedback

| Feedback | Disposition | Contract impact |
| --- | --- | --- |
| Directionality must be explicit | Accepted | v0 separates SemLink push/admission from SemOps pull of `/api/evidence` |
| SemLink must not mint trusted SemOps operator headers | Accepted | v0 assigns trusted-header translation to SemOps gateway/sidecar/test harness |
| `target_asset_id` and born-target proof cannot be SemLink responsibility | Accepted | v0 removes `target_asset_id` from request and makes SemOps derive/prove target |
| Structured `AUTOPILOT_VERSION` payload is not first-class in SemLink yet | Deferred | v0 admits command intent and ACK/status; structured result observation is a follow-up |

## Friction Feedback

| Feedback | Disposition | Contract impact |
| --- | --- | --- |
| Rename `mesh_node_id` | Accepted | v0 uses `companion_node_id`; current mesh naming is compatibility-only |
| Prefer MAVLink-native target and command fields | Accepted | v0 uses `target_system_id`, `target_component_id`, `command_id`, and `requested_message_id` |
| Treat request `id` as optional/generated | Accepted | v0 omits required request ID; SemOps derives native command-intent ID |
| Split timestamps | Partially accepted | v0 uses `requested_at`; `ack_observed_at` and `result_observed_at` are reserved for status/result evidence |
| Define TTL/replay behavior | Accepted | v0 derives `expires_at` and rejects stale replay unless a later policy renews it |

## Interop Feedback

| Feedback | Disposition | Contract impact |
| --- | --- | --- |
| MAVLink owns command/message/target/ACK vocabulary | Accepted | v0 field names and values are MAVLink-native |
| CS API/SensorThings/SWE belong at the SemOps/SemConnect edge | Accepted | v0 keeps CS API out of the hot path and requires projection proof |
| Governance fields are C360 audit/safety concepts | Accepted | v0 marks idempotency, authority, duplicate, born-target proof, and no-transmit posture as governance |
| `AUTOPILOT_VERSION` result maps to observation/readback payload | Accepted/deferred | v0 records this mapping; implementation waits for structured result payload support |

## Deferred Feedback

- CS API-first SemLink path.
- Hardware transmit.
- Generic robotics command protocol.
- Durable multi-replica idempotency.
- Cryptographic node auth/mTLS/JWT.
- Raw MAVLink over CS API.

## Required Follow-Ups

- SemOps handler DTOs now accept the v0 `companion_node_id` / `target_system_id` / `command_id` /
  `requested_message_id` shape.
- Preserve compatibility aliases only where needed for staged rollout, then retire them after SemLink adoption.
- Route tests now load the v0 fixtures and prove SemOps command-intent admission behavior.
- Send the v0 contract and fixtures back to SemLink for final hold-out review before SemLink resumes implementation.
