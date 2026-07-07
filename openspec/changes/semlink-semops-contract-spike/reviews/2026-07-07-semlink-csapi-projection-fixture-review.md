# SemLink CS API Projection Fixture Review

Scope: SemOps/SemConnect interop proof for the accepted SemLink companion readback intent.

Decision: accept a CS API projection fixture as standards-edge evidence only. The SemLink-to-SemOps hot path remains
native for MVP.

## Fixture Shape

The accepted SemLink readback intent now has a CS API projection fixture at
`testdata/contracts/semlink-companion-readback-v0/csapi-projection.accepted.json`.

The fixture maps:

- SemLink companion node and MAVLink target asset to CS API `System` resources.
- The readback lane to a CS API `ControlStream`-shaped resource.
- SemOps command intent admission to a CS API `Command`-shaped record.
- Admission status to a CS API `SystemEvent`-shaped record.
- Future decoded `AUTOPILOT_VERSION` payload to a deferred CS API `Observation`-shaped record.

## Verification

`internal/egress/csapi/semlink_projection_fixture_test.go` loads the native request fixture, accepted response fixture,
and CS API projection fixture together. It proves the projection preserves:

- MAVLink command/message IDs and target system/component IDs.
- Companion and target identity.
- Correlation and idempotency keys.
- Source reference, issue time, expiry, and accepted status.
- SemOps authority scope, claim scope, duplicate state, mutation count, and no-native/no-companion-transmit posture.

## Capability Gaps

- CS API score remains Amber. It is the correct SemOps/SemConnect projection edge, not the boat-local hot path.
- CS API Command Status egress remains deferred. Admission status is represented as SystemEvent-shaped fixture evidence
  until command lifecycle/status publication is reviewed.
- Decoded `AUTOPILOT_VERSION` Observation remains deferred until SemLink emits structured result payload evidence.
- Raw MAVLink over CS API remains a non-goal; MAVLink telemetry/control stays native.
- This fixture does not add hosted SemOps CS API behavior or SemLink CS API runtime dependency.
