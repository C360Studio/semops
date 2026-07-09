# MAVLink PX4 Motion-Required Review

Date: 2026-06-23

Scope: stricter PX4/Gazebo headless telemetry smoke with `SEMOPS_MAVLINK_SITL_SMOKE_REQUIRE_MOTION=true`.

## Evidence

Command:

```bash
SEMOPS_MAVLINK_SITL_GATE_MODE=px4-headless-stack \
SEMOPS_MAVLINK_SITL_SMOKE_REQUIRE_MOTION=true \
SEMOPS_MAVLINK_SITL_SMOKE_TIMEOUT=60s \
bash scripts/mavlink-sitl-gate.sh
```

Result:

- Evidence file: `tmp/mavlink-sitl-evidence/2026-06-23T18-54-08Z-px4-headless-stack.env`
- SemOps commit under test: `0884cdd`
- Simulator image: `jonasvautherin/px4-gazebo-headless:1.17.0`
- Vehicle/world: `gz_x500` / `default`
- External track: `c360.edge-compose.cop.mavlink.track.system-1`
- Snapshot URL: `http://127.0.0.1:8080/api/cop/snapshot`
- `TestExternalSITLTelemetryCOPSnapshot`: passed in 25.52s with `require_motion=true`
- SemOps compose resources and the PX4 simulator container were removed by the helper after the run.

## Caveats

- The motion delta is passive telemetry movement/noise from the simulator route, not commanded vehicle tasking.
- This still does not prove takeoff, mission upload, repositioning, command ACK semantics, failsafe behavior, or
  hardware behavior.
- Keep `5.4` open until ArduPilot parity and command/control gates have their own reviewed evidence.

## Outcome

Accept the PX4/Gazebo headless route as sufficient for MVP simulator telemetry with motion-required readback. Do not
use this result as command/control evidence.
