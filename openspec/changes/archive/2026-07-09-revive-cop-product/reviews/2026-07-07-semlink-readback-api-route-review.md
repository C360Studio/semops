# SemLink Readback API Route Review

Scope: next slice after the command-intent graph writer made SemLink and CS API admission plans persistable.

Decision: accept an opt-in COP API route seam for SemLink ArduPilot readback intent.

The route lets a configured SemOps runtime accept a SemLink companion request for the MVP
`MAV_CMD_REQUEST_MESSAGE` / `AUTOPILOT_VERSION` readback, pass it through the existing guarded SemLink ingress, and
write only the resulting command-intent graph plan. It returns the same explicit posture as COP readback:
native execution is not allowed, companion transmit is not allowed, and rejected admissions are reported without
writing a graph mutation.

Boundary choices:

- The route is `/api/cop/semlink/ardupilot/readback`, not a generic command execution endpoint.
- The handler requires both a SemLink readback ingress and a command plan writer; unconfigured runtimes fail closed.
- Accepted admissions call the command graph writer, so the API boundary has a real persistence seam.
- Rejected admissions and duplicate idempotency keys return admission state and do not call the writer.
- Writer failure returns an error response that still carries no-transmit posture.

Evidence:

- `internal/api/cop/semlink_readback.go` adds the route DTOs, option-injected ingress/writer contracts, and response
  posture.
- `internal/api/cop/handler_test.go` proves accepted admission writes a plan, rejected admission does not write,
  writer failure does not grant transmit authority, and unconfigured runtime returns `503`.

Residual risk:

- The default application composition still needs live SemLink/BlueOS configuration, graph-backed target discovery,
  and operator/auth policy before the route should be enabled outside tests.
- This does not exercise a live SemLink node, Navigator hardware, ArduPilot vehicle, or companion transmit path.
- Command status/ACK reconciliation remains the feed-readback lane; this route records desired readback intent only.
