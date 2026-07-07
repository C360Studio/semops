# SemLink Companion Ingress Review

Scope: first SemOps ingress boundary for SemLink companion/mesh ArduPilot readback requests.

## Verdict

Accept a local ingress package that admits SemLink companion readback requests into governed command intent only.

The ingress converts a SemLink BlueOS/Navigator/Pi companion request into the already-reviewed `AUTOPILOT_VERSION`
intent helper, sends it through the guarded command projector, requires born target assets, and explicitly returns no
native or companion transmit authority from admission.

## Boundary Choices

- This is not an HTTP, gRPC, BlueOS, or Navigator runtime surface yet.
- This is not native MAVLink transmit authority and does not execute commands.
- Unsafe companion actions are rejected before projection by the readback intent helper.
- Missing target assets produce admission rejection with no graph mutations.

## Evidence

- `internal/ingress/semlink/readback.go` adds `Ingress.AdmitArduPilotReadback`.
- `internal/ingress/semlink/readback_test.go` proves accepted intent-only projection, unsafe-action rejection, and
  unborn-target rejection.
- Focused package test: `go test ./internal/ingress/semlink`.

## Residual Risk

- A hosted API route, BlueOS extension call path, authentication, and live SemLink node are still future slices.
- Idempotency collapse is inherited from the guarded projector but not yet exercised by a SemLink-specific duplicate
  fixture.
