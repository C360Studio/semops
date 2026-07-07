## Context

SemOps currently has an opt-in SemLink ArduPilot readback route seam that admits only governed
`AUTOPILOT_VERSION` command intent. The route requires trusted SemOps caller headers, mesh-node binding, born-target
lookup or derivation, idempotency, and explicit no-native/no-companion-transmit posture.

SemLink is building companion-local evidence APIs and BlueOS-style deployment handoff material. That evidence work
names SemOps as a downstream consumer, but it does not yet encode the SemOps readback-admission contract.

SemConnect already carries CS API bridge and conformance responsibility for SemOps. The companion path should use
standards vocabulary where practical, but SemConnect should not become a private command bus or a required dependency
for boat-local companion readiness unless the spike proves it reduces friction.

## Goals / Non-Goals

**Goals:**

- Produce a shared SemLink-to-SemOps contract that SemLink can implement without guessing.
- Give SemLink a formal feedback path before SemOps accepts the contract.
- Use MAVLink vocabulary for native autopilot semantics.
- Map SemLink-origin readback intent and status to CS API/SensorThings/SWE concepts at the SemOps standards edge.
- Keep C360-only fields limited to governance, authority, provenance, idempotency, and audit semantics.
- Include SemConnect as a standards hold-out set for projection feedback.
- Decide with evidence whether CS API belongs in the hot SemLink-to-SemOps path.

**Non-Goals:**

- Requiring SemLink to expose or consume CS API for the MVP companion path.
- Claiming hardware command transmit authority, mission upload, mode change, arm/disarm, offboard control, or local
  safety override.
- Promoting SemConnect into the authority that decides whether a command is safe to issue.
- Sending high-rate raw MAVLink through CS API.
- Defining a general robotics command protocol beyond the MVP readback request.

## Decisions

### 1. Native companion path is the default MVP wire

SemLink-to-SemOps MVP integration uses a compact native companion contract owned by SemOps. It can be implemented by
a boat-local SemLink companion without SemConnect in the request path.

The native contract exists to reduce operational friction and to preserve SemOps-specific governance semantics that
are not standards-domain facts.

### 2. Standards mapping is mandatory

Every native contract field must appear in a standards mapping table:

- MAVLink mappings for vehicle-native commands, message IDs, target system/component IDs, ACK/status, and readback
  observations.
- CS API/SensorThings/SWE mappings for Systems, Deployments, ControlStreams, Commands, SystemEvents, Datastreams, and
  Observations when those concepts fit.
- C360 governance-exception mappings for fields such as trusted authority scope, idempotency, mesh-node provenance,
  born-target proof, owner token, no-transmit posture, and graph admission result.

Fields without a standards mapping or a written governance-exception reason are cut from the MVP contract.

### 3. SemConnect proves interop at the SemOps edge

SemOps must prove that accepted SemLink readback intent and resulting status can be projected through SemConnect as
CS API-shaped resources without losing:

- target system and control stream identity;
- native MAVLink command/message IDs;
- request correlation and sender/provenance;
- accepted/rejected/duplicate/stale/superseded status;
- no-native/no-companion-transmit posture; and
- result/readback evidence such as ACK and `AUTOPILOT_VERSION` observation payload.

If SemConnect cannot carry one of those facts without private contortions, the spike records either a C360 governance
exception or a SemConnect follow-up.

### 4. SemLink feedback is an acceptance gate

SemOps owns the contract, but SemLink owns the companion runtime that must implement it. The spike must include a
SemLink feedback loop before the contract is accepted.

SemLink feedback is categorized as:

- **Blocking:** the contract cannot be implemented safely or practically from SemLink companion runtime state.
- **Friction:** the contract can be implemented, but a field, header, timing rule, or status shape duplicates existing
  MAVLink, BlueOS, or companion evidence concepts.
- **Interop:** the contract misses a standards mapping, uses private vocabulary where a standard term fits, or makes
  later SemConnect projection harder.
- **Deferred:** the feedback is valid but belongs after the MVP `AUTOPILOT_VERSION` readback slice.

Blocking and friction feedback must be dispositioned before the MVP decision. Disposition may be an accepted contract
change, a documented non-goal, or a follow-up task with an explicit reason it does not block the MVP.

### 5. CS API path is scored before promotion

The spike scores CS API/SemConnect as a possible SemLink-to-SemOps path instead of assuming it:

- **Green:** CS API carries the request and status with less or equal friction than the native companion contract.
- **Amber:** CS API is excellent for SemOps interop projection, but the native contract remains the hot path.
- **Red:** CS API projection loses meaning or requires private shape overloads; keep native path and file SemConnect
  follow-ups before making interoperability claims.

Green is not required for MVP. Amber is acceptable if the SemOps interop projection is lossless enough for partners.

### 6. Raw telemetry and command authority stay separate

SemLink may emit MAVLink on a telemetry port for autopilot-facing runtime behavior. CS API/SemConnect is for
interoperable resource state, tasking records, status, and observations. It is not a replacement for high-rate native
telemetry transport.

SemOps remains the authority plane: standards tasking input, native companion input, and future operator actions all
normalize into governed command intent before any native actuation is attempted.

## Spike Deliverables

- `docs/contracts/semlink-companion-readback-v0.md` describing the native contract and standards mapping.
- JSON fixtures for accepted, duplicate, rejected, and CS API projection examples.
- Focused tests that prove the native fixture maps into SemOps command intent.
- A SemConnect interop proof or documented fixture review that maps the accepted intent/status into CS API resources.
- A SemLink feedback/disposition record that shows which contract changes were accepted, rejected, or deferred.
- A short review record with the friction score and follow-up routing.

## Risks / Trade-offs

- A native companion contract can drift into private protocol if the mapping table is treated as documentation only.
  The contract fixture tests must fail when fields lack mapping or governance-exception coverage.
- CS API may be good enough for interop but too heavy for boat-local companion MVP. That is acceptable if the
  boundary is explicit.
- SemConnect may need capability work. Those gaps should become SemConnect changes, not SemOps-local CS API behavior.
