# SemLink AUTOPILOT_VERSION Result Contract Review

Date: 2026-07-09
Scope: SemLink companion readback v0 structured `COMMAND_ACK` and `AUTOPILOT_VERSION` evidence fixtures.

## Decision

Accept a fixture-level SemOps contract for structured SemLink result evidence before SemLink implementation resumes.

`COMMAND_ACK` is status/event evidence. Decoded `AUTOPILOT_VERSION` is observation/readback evidence. The two shapes use
separate timestamps (`ack_observed_at` and `result_observed_at`) and neither shape grants native execution or companion
transmit authority.

## Evidence

- `testdata/contracts/semlink-companion-readback-v0/status.command-ack.json` defines the post-admission MAVLink
  `COMMAND_ACK` status evidence shape.
- `testdata/contracts/semlink-companion-readback-v0/result.autopilot-version.json` defines the decoded
  `AUTOPILOT_VERSION` result payload SemLink should emit.
- `testdata/contracts/semlink-companion-readback-v0/csapi-projection.accepted.json` maps ACK to CS API SystemEvent
  evidence and maps decoded `AUTOPILOT_VERSION` to CS API Observation evidence.
- `internal/egress/csapi/semlink_projection_fixture_test.go` ties the request, admission response, ACK fixture, result
  fixture, and CS API projection together.

## Boundaries

- This is not a new SemLink-to-SemOps hot-path protocol.
- This does not require CS API, SemConnect, BlueOS REST, MAVLink2REST, or endpoint-manager state in SemLink.
- This does not add hardware transmit, mission upload, mode change, arm/disarm, or offboard control authority.
- SemLink still owns implementation of first-class decoded `AUTOPILOT_VERSION` evidence on its side.
