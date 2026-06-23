# MAVLink External SITL Smoke

This smoke is the first SemOps simulator-fidelity gate for MAVLink telemetry. It proves an external PX4, MAVSDK,
ArduPilot SITL, or equivalent MAVLink source can feed the hosted SemOps UDP component and appear in the COP snapshot.

It does not prove command/control, mission handling, serial/TCP transport, signed links, hardware behavior, or PX4
conformance.

## Prerequisites

- A simulator or MAVSDK route that emits MAVLink heartbeat and `GLOBAL_POSITION_INT` telemetry to UDP
  `127.0.0.1:14550`.
- The simulator's first vehicle should use MAVLink system ID `1`, or `SEMOPS_MAVLINK_SITL_SMOKE_EXPECTED_TRACK_ID`
  should be set to the expected graph track ID.
- Docker resources sufficient to run the COP stack.

## One-Command Stack Gate

Start the simulator first, or keep it ready to emit telemetry while the stack starts. Then run:

```bash
SEMOPS_COP_MAVLINK_SYSTEM_IDS=1,42 \
SEMOPS_COP_SMOKE_MAVLINK_SITL_ENABLED=true \
bash scripts/cop-stack-smoke.sh
```

The script still runs the deterministic generated-frame graph smoke for system `42`. The external SITL smoke observes
system `1` through `GET /api/cop/snapshot` and does not inject its own MAVLink frames.

For stricter motion evidence:

```bash
SEMOPS_COP_MAVLINK_SYSTEM_IDS=1,42 \
SEMOPS_COP_SMOKE_MAVLINK_SITL_ENABLED=true \
SEMOPS_MAVLINK_SITL_SMOKE_REQUIRE_MOTION=true \
bash scripts/cop-stack-smoke.sh
```

## Focused Smoke Against A Running Stack

If the COP stack is already running and a simulator is emitting telemetry:

```bash
SEMOPS_MAVLINK_SITL_SMOKE_SNAPSHOT_URL=http://127.0.0.1:8080/api/cop/snapshot \
SEMOPS_MAVLINK_SITL_SMOKE_EXPECTED_TRACK_ID=c360.edge-compose.cop.mavlink.track.system-1 \
go test ./internal/smoke/mavlink -run TestExternalSITLTelemetryCOPSnapshot -count=1 -v
```

Useful optional knobs:

- `SEMOPS_MAVLINK_SITL_SMOKE_TIMEOUT`: default `2m`.
- `SEMOPS_MAVLINK_SITL_SMOKE_MIN_UPDATES`: default `2`.
- `SEMOPS_MAVLINK_SITL_SMOKE_REQUIRE_MOTION`: default `false`.

## Acceptance

The smoke requires:

- `feed.mavlink` is live.
- The expected MAVLink track is present with source `mavlink`.
- Position is non-zero.
- Velocity evidence is present.
- Provenance owner is `semops.feed.mavlink`.
- Source reference is non-empty.
- At least two simulator updates are observed.
- If motion is required, latitude or longitude changes between the first and last accepted observations.

## Claim Boundary

Passing this smoke is simulator telemetry evidence only. Command authority and mission semantics need a separate
reviewed gate with safe commands, ACK handling, post-command state polling, and simulator-specific readiness checks.
