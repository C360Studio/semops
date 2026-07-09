# MAVLink ArduPilot Parity Gate Review

Date: 2026-06-25
Scope: `scripts/mavlink-sitl-gate.sh` ArduPilot-specific telemetry parity gate before any ArduPilot simulator
interoperability claim.

## Verdict

Accept an explicit fail-closed `ardupilot-stack` mode as the right next guard, but do not close ArduPilot telemetry
parity.

The gate now makes ArduPilot evidence distinct from the already-passing PX4/Gazebo headless lane. The local run blocked
because this laptop has no `sim_vehicle.py` and no ArduPilot/ArduCopter Docker image, even though the PX4 image exists.
That is useful readiness evidence, not simulator interoperability evidence.

## Findings

1. ArduPilot parity must not inherit PX4 evidence.

   `ardupilot-stack` stamps `simulator_family=ardupilot` and rejects contradictory family input. This prevents a PX4
   telemetry pass from being reused as ArduPilot evidence.

2. Motion should be the default for ArduPilot parity.

   The mode defaults `SEMOPS_MAVLINK_SITL_SMOKE_REQUIRE_MOTION=true` unless the operator explicitly overrides it. A
   heartbeat-only or static-position pass should not become the first ArduPilot interoperability claim.

3. The local result is blocked, not skipped into success.

   The 2026-06-26T00:44:17Z run exited with `result=blocked_no_local_simulator`, which is the correct result while no
   ArduPilot source is present. Do not mark task 5.95 complete until a real ArduPilot SITL source feeds the hosted UDP
   component and passes COP snapshot readback.

## Required Before Claim

- Install or run a real ArduPilot SITL source, preferably with `sim_vehicle.py -v ArduCopter --out=udp:127.0.0.1:14550`
  or an explicitly reviewed ArduPilot-family container.
- Run `SEMOPS_MAVLINK_SITL_GATE_MODE=ardupilot-stack bash scripts/mavlink-sitl-gate.sh`.
- Preserve the ignored evidence file path, simulator name/version, vehicle, launch command, UDP route, SemOps commit,
  motion requirement, result, and unresolved limitations in the review/docs.
- Keep live command/control, mission upload, MAVSDK/offboard behavior, hardware, serial/TCP transport, and signed-link
  claims under separate gates.
