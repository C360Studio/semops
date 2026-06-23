# MAVLink PX4 Headless Stack Pass Review

Date: 2026-06-23

Scope: first successful `px4-headless-stack` run through the hosted SemOps COP stack.

## Evidence

Command:

```bash
SEMOPS_MAVLINK_SITL_DOCKER_PULL=true \
SEMOPS_MAVLINK_SITL_GATE_MODE=px4-headless-stack \
bash scripts/mavlink-sitl-gate.sh
```

Result:

- Evidence file: `tmp/mavlink-sitl-evidence/2026-06-23T18-30-24Z-px4-headless-stack.env`
- SemOps commit under test: `767fef0`
- Simulator image: `jonasvautherin/px4-gazebo-headless:1.17.0`
- Vehicle/world: `gz_x500` / `default`
- External track: `c360.edge-compose.cop.mavlink.track.system-1`
- Snapshot URL: `http://127.0.0.1:8080/api/cop/snapshot`
- `TestExternalSITLTelemetryCOPSnapshot`: passed in 23.52s
- Full command also passed hosted COP, MAVLink born-first, CoT born-first, CAP born-first, and SAPIENT preflight smokes.
- SemOps compose resources and the PX4 simulator container were removed by the helper after the run.

## Caveats

- This is simulator telemetry evidence only. It does not prove PX4 command authority, mission upload, signed links,
  TCP/serial transport, or hardware behavior.
- `SEMOPS_MAVLINK_SITL_SMOKE_REQUIRE_MOTION=false` for this first pass. A later gate should require motion once the
  simulator route is stable enough for stricter evidence.
- The simulator image is unofficial. It is acceptable for MVP/demo smoke evidence, but official PX4 conformance remains
  outside this result.
- PX4 logs are very noisy under `docker logs`; future diagnostics should avoid dumping raw PX4 shell prompt output into
  normal run summaries.

## Outcome

Close the narrow `4.7` PX4/MAVSDK telemetry smoke task. Keep broader `5.4`, command/control, ArduPilot parity, and
motion-required gates open.
