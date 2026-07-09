# MAVLink ArduPilot Managed Docker Review

Date: 2026-06-28
Scope: opt-in managed ArduPilot Docker startup support in `scripts/mavlink-sitl-gate.sh`.

## Verdict

Accept the managed Docker path as setup support for task 5.95, but do not close ArduPilot telemetry parity.

The helper now has enough structure to start a reviewed ArduPilot-family container on the SemOps Compose network before
the external MAVLink SITL smoke. It still requires explicit operator choice of image and keeps the actual parity claim
blocked until `ardupilot-stack` observes a real ArduPilot source through the hosted UDP component and COP snapshot.

## Findings

1. The helper does not choose an ArduPilot image for the operator.

   `SEMOPS_MAVLINK_SITL_ARDUPILOT_DOCKER_IMAGE` is required for managed Docker startup, and there is no default image.
   A missing configured image blocks unless `SEMOPS_MAVLINK_SITL_ARDUPILOT_DOCKER_PULL=true` is set after review.

2. The managed route uses the product stack, not generated frames.

   `ardupilot-stack` starts the container through the COP stack hook, attaches it to the SemOps Compose network, and
   routes the default `sim_vehicle.py` output to `semops:14550`. The external SITL smoke still has to observe
   graph-visible MAVLink telemetry before the gate can pass.

3. The parity gate remains open.

   This change only removes setup friction for a reviewed image. It does not provide a local ArduPilot image, does not
   run a passing stack smoke, and does not claim ArduPilot simulator interoperability.

## Verification

- `bash -n scripts/mavlink-sitl-gate.sh`
- `SEMOPS_MAVLINK_SITL_GATE_MODE=ardupilot-stack bash scripts/mavlink-sitl-gate.sh`
- `SEMOPS_MAVLINK_SITL_GATE_MODE=ardupilot-stack SEMOPS_MAVLINK_SITL_ARDUPILOT_DOCKER_IMAGE=example/ardupilot:missing bash scripts/mavlink-sitl-gate.sh`
- `SEMOPS_MAVLINK_SITL_GATE_MODE=ardupilot-start SEMOPS_MAVLINK_SITL_ARDUPILOT_DOCKER_IMAGE=example/ardupilot:missing bash scripts/mavlink-sitl-gate.sh`
- `go test ./internal/smoke/mavlink -run TestExternalSITLTelemetryCOPSnapshot -count=1 -v`
- `openspec validate revive-cop-product --strict`
