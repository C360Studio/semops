## Why

SemLink is moving toward a boat-local BlueOS/Navigator/Pi companion and mesh-node product, while SemOps is adding the
governed COP/GCS-glass path for SemLink-origin ArduPilot readback intent. The two projects now need one shared
contract before SemLink resumes integration work.

The contract must avoid two failure modes:

- inventing private C360 wire vocabulary where MAVLink, OGC Connected Systems API, SensorThings, or SWE terms already
  fit cleanly; and
- forcing CS API/SemConnect into the hot SemLink-to-SemOps companion path when that adds friction to boat-local
  runtime, offline operation, or MVP safety gates.

SemOps should own this spike because it owns COP command authority, GCS glass, and the standards-facing bridge
boundary around SemLink-provided state. SemConnect should be included as a hold-out standards project so the contract
is tested against CS API projection pressure instead of drifting into an isolated private protocol.

## What Changes

- Define a SemOps-owned SemLink companion contract spike for the MVP ArduPilot `AUTOPILOT_VERSION` readback path.
- Require every contract field to carry a standards mapping or a C360 governance-exception justification.
- Keep the SemLink-to-SemOps companion path native and low-friction unless the spike proves CS API lowers friction.
- Require SemOps to project accepted SemLink readback intent/status through SemConnect/CS API without semantic loss.
- Add a friction scoring review that decides whether CS API should remain only at the SemOps interop edge, become an
  optional SemLink path, or drive follow-up SemConnect capability work.

## Capabilities

### New Capabilities

- `semlink-semops-companion-contract`: Defines the native companion contract, standards mapping, governance exceptions,
  SemConnect interop proof, and friction scoring required before SemLink resumes integration work.

## Impact

- SemOps OpenSpec, contract fixtures, and follow-up implementation tickets.
- SemLink companion evidence and readback integration planning.
- SemConnect CS API fixture/projection coverage if the interop proof exposes missing ControlStream, Command,
  SystemEvent, Datastream, or Observation behavior.
- Future SemOps API, command-intent, and CS API egress/ingress adapters.
