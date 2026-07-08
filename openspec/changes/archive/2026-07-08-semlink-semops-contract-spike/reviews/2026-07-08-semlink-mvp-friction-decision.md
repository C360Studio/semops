# SemLink MVP Friction Decision

Scope: final spike decision after SemLink initial feedback was dispositioned into the v0 native companion contract,
fixtures, route tests, and CS API projection fixture.

## Decision

Accept the MVP path as:

- Native SemLink-to-SemOps companion hot path for `MAV_CMD_REQUEST_MESSAGE` / `AUTOPILOT_VERSION` intent admission.
- CS API/SemConnect Amber as the SemOps standards projection edge, not a boat-local runtime dependency.
- Focused SemLink adapter work may resume against the v0 fixture set, provided it stays within the no-transmit
  companion readback-intent boundary.

## Feedback Disposition

SemLink's blocking feedback is dispositioned:

- Directionality is explicit: SemLink posts intent to SemOps; SemOps or operator tooling may separately pull
  SemLink `/api/evidence`.
- SemLink does not mint SemOps trusted operator posture; a SemOps deployment boundary owns trusted-header translation.
- `target_asset_id` and born-target proof stay SemOps responsibilities.
- Structured `AUTOPILOT_VERSION` result payload support is deferred to a focused readback-observation slice.

SemLink's friction feedback is dispositioned:

- `companion_node_id` is canonical; `mesh_node_id` remains a staged compatibility alias only.
- MAVLink-native fields are canonical: `target_system_id`, `target_component_id`, `command_id`, and
  `requested_message_id`.
- Request `id` is not required; SemOps derives command-intent identity.
- `requested_at` drives admission and expiry; `ack_observed_at` and `result_observed_at` are reserved for later
  status/result evidence.
- `ttl_seconds` replay behavior is explicit: SemOps derives `expires_at` and rejects stale replay unless later policy
  renews it.

## CS API Score

Score: Amber.

CS API/SemConnect is good enough for interoperable SemOps readback projection because the fixture preserves companion
and target Systems, readback ControlStream, Command-shaped admission, status/event evidence, native MAVLink IDs,
correlation, idempotency, provenance, authority scope, duplicate state, mutation count, and no-transmit posture.

CS API is not promoted into the SemLink hot path for MVP because it would add companion-runtime friction without
reducing governance ambiguity. The native contract uses MAVLink vocabulary for vehicle-native facts and C360
governance fields only where SemOps owns authority, idempotency, target proof, and audit posture.

## Follow-Ups

- SemLink can implement a focused optional adapter against the v0 fixtures.
- SemOps should retire compatibility aliases after SemLink adopts the v0 shape.
- Structured `AUTOPILOT_VERSION` observation evidence remains deferred until SemLink emits or exposes a decoded result
  payload.
- Hardware transmit, generic robotics command protocol work, CS API-first runtime dependency, cryptographic node auth,
  and raw MAVLink over CS API remain non-goals for this MVP.
