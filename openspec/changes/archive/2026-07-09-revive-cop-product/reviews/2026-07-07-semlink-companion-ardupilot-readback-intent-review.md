# SemLink Companion ArduPilot Readback Intent Review

Scope: next SemOps slice after SemLink was rechartered as the likely BlueOS/Navigator/Pi companion and mesh-node path.

## Verdict

Accept a narrow helper that maps SemLink companion pressure into governed command intent for ArduPilot readback only.

The helper builds a SemOps command-intent entity for `MAV_CMD_REQUEST_MESSAGE` / `AUTOPILOT_VERSION`, stamps SemLink
companion provenance, requires a born target asset through the guarded projector, and lets MAVLink ACK/status evidence
reconcile lifecycle state without rewriting desired state, authority, or target edges.

## Boundary Choices

- This is GCS-glass/readback support, not live vehicle authority.
- SemLink is a governed source edge; it does not become the owner of broad COP command authority.
- Mission upload, mode change, arm/disarm, offboard control, hardware command authority, and local safety override
  remain outside the MVP helper.
- ACK/status reconciliation stays separate from desired command intent, so native drivers report evidence rather than
  owning desired state.

## Evidence

- `internal/projectors/command/semlink.go` adds `NewSemLinkArduPilotReadbackIntent`.
- `internal/projectors/command/semlink_test.go` proves SemLink readback intent creation, guarded admission, MAVLink ACK
  reconciliation, and rejection of unsafe companion actions.
- Focused package test: `go test ./internal/projectors/command`.

## Residual Risk

- No live SemLink node, BlueOS extension, Navigator hardware, or ArduPilot command route is exercised in this slice.
- The helper does not yet wire an HTTP/gRPC/API boundary; it is the governed projection guard for the future edge.
