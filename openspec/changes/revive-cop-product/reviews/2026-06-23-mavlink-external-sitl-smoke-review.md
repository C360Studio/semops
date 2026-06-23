# MAVLink External SITL Smoke Review

Date: 2026-06-23
Scope: skipped-by-default external PX4/MAVSDK/SITL telemetry smoke for the hosted COP stack

## Decision

Accept the external SITL telemetry smoke harness as the next MAVLink simulator-fidelity step, but do not close the
PX4/MAVSDK evidence gate until it is run against a real simulator.

The harness is intentionally observer-only: it polls the Caddy-routed COP snapshot for a simulator-produced MAVLink
track and does not inject generated frames itself. That keeps the existing generated-frame graph smoke as deterministic
CI evidence while creating a real external-source gate for PX4, MAVSDK, ArduPilot SITL, or a hardware-adjacent MAVLink
source.

## Red-Team Findings

1. Harness existence is not simulator evidence.

   The new test skips unless a snapshot URL is provided. It is useful because it defines the acceptance surface, but a
   local PX4/MAVSDK/ArduPilot run still needs to be recorded before claiming simulator fidelity.

2. Telemetry is not command/control.

   The smoke requires live feed health, position, velocity, provenance, source refs, and repeated updates. It does not
   arm, take off, command modes, upload missions, validate ACK handling, or prove PX4 mode semantics.

3. The simulator must feed the hosted component path.

   The smoke observes `GET /api/cop/snapshot`; the simulator must emit MAVLink to SemOps's UDP input component. That
   avoids a test-only projector shortcut and keeps SemStreams lifecycle, payload registry, graph writes, and Caddy
   readback in the path.

4. System IDs must be explicit.

   PX4 commonly uses system ID `1`, while the deterministic stack smoke uses system `42`. The Compose stack now honors
   `SEMOPS_COP_MAVLINK_SYSTEM_IDS`, and the opt-in stack mode defaults to `1,42` so both paths can coexist.

## Evidence Accepted

- `internal/smoke/mavlink/external_sitl_test.go` skips unless `SEMOPS_MAVLINK_SITL_SMOKE_SNAPSHOT_URL` is set.
- The smoke requires `feed.mavlink` live health, source `mavlink`, non-zero position, velocity evidence,
  `semops.feed.mavlink` provenance, non-empty source reference, and repeated updates.
- `SEMOPS_MAVLINK_SITL_SMOKE_REQUIRE_MOTION=true` can require position movement for stronger demo evidence.
- `scripts/cop-stack-smoke.sh` can opt into the external SITL smoke with
  `SEMOPS_COP_SMOKE_MAVLINK_SITL_ENABLED=true`.
- `compose.cop.yml` passes `SEMOPS_COP_MAVLINK_SYSTEM_IDS` through to the app runtime.

## Follow-Ups

- Run the opt-in smoke against PX4 SITL or MAVSDK and record simulator version, command, system ID, and pass/fail
  evidence before closing the MAVLink simulator-fidelity gate.
- Add a separate safe command/control gate only after readiness checks, command authority, ACK handling, and
  post-command state polling are reviewed.
