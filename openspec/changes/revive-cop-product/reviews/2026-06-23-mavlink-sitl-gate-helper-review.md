# MAVLink SITL Gate Helper Review

Date: 2026-06-23

## Decision

Add a guarded helper for the external MAVLink SITL gate.

`scripts/mavlink-sitl-gate.sh` exists to make future PX4/MAVSDK/ArduPilot runs repeatable without weakening the
evidence boundary. It defaults to preflight mode, which checks local readiness and proves the external smoke skips
without a snapshot URL. Focused and stack modes require a named simulator source before they run the evidence gate.

This does not close the PX4/MAVSDK/SITL evidence tasks.

## Evidence Accepted

- Preflight mode runs `go test ./internal/smoke/mavlink -run TestExternalSITLTelemetryCOPSnapshot -count=1 -v` without
  a snapshot URL and expects the guarded skip path.
- Focused mode wraps the direct external SITL smoke against an existing COP snapshot URL.
- Stack mode wraps `scripts/cop-stack-smoke.sh` with `SEMOPS_COP_SMOKE_MAVLINK_SITL_ENABLED=true` and keeps the
  generated-frame system `42` separate from external simulator system `1`.
- Focused and stack modes require `SEMOPS_MAVLINK_SITL_SIMULATOR_NAME`.
- Focused and stack modes require local simulator tooling or `SEMOPS_MAVLINK_SITL_ALLOW_REMOTE_SOURCE=true` for an
  already-running remote or hardware-adjacent source.
- Evidence files are written under ignored `tmp/mavlink-sitl-evidence/` by default.

## Red-Team Findings

1. Helper ergonomics can weaken evidence.

   Requiring `SEMOPS_MAVLINK_SITL_SIMULATOR_NAME` prevents a successful stack run from being mistaken for a simulator
   run when only generated frames were present.

2. Remote sources are valid but must be declared.

   A simulator can run outside the local PATH or Docker image set. The explicit remote-source override records that
   posture and keeps the absence of local tooling visible.

3. Evidence files are not portable fixtures.

   Gate evidence is local run metadata and should stay ignored. A future captured MAVLink fixture must go through the
   fixture manifest and promotion review before it travels with the demo.

## Verification

- `bash -n scripts/mavlink-sitl-gate.sh`
- `SEMOPS_MAVLINK_SITL_GATE_MODE=preflight bash scripts/mavlink-sitl-gate.sh`
- `SEMOPS_MAVLINK_SITL_GATE_MODE=focused bash scripts/mavlink-sitl-gate.sh` exits `2` without
  `SEMOPS_MAVLINK_SITL_SIMULATOR_NAME`
- `go test ./internal/smoke/mavlink -run TestExternalSITLTelemetryCOPSnapshot -count=1 -v`
- `openspec validate revive-cop-product --strict`
- `git diff --check`
