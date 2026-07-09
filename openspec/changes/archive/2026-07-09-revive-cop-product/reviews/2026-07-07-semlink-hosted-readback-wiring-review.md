# SemLink Hosted Readback Wiring Review

Scope: hosted SemOps API composition after the SemLink readback route seam landed.

Decision: accept explicit-config hosted wiring for SemLink ArduPilot readback intent.

The slice makes the existing `/api/cop/semlink/ardupilot/readback` route usable in the hosted API process only when
`SEMOPS_COP_SEMLINK_READBACK_ENABLED=true`. The route is still command intent only: the app composes the guarded
SemLink ingress, graph-backed command target resolver, command-intent graph writer, and the
`semops.command.intent` owner token. The default remains disabled, and an enabled runtime fails startup if graph
requesting or the command owner token is unavailable.

Boundary choices:

- Target birth checks use `graph.query.entity` instead of fixture/static target IDs.
- Command-intent writes use the normal SemOps command owner token and `SEMOPS_COP_SEMLINK_READBACK_WRITE_TIMEOUT`.
- Enabling the route does not enable native MAVLink command transmit or SemLink companion transmit.
- The API remains a SemOps GCS-glass ingress boundary; SemLink stays the boat-local companion/mesh product path.

Evidence:

- `internal/projectors/command/target_resolver.go` adds a graph-backed target resolver for guarded command admission.
- `cmd/semops/main.go` wires the SemLink ingress and command graph writer only under
  `SEMOPS_COP_SEMLINK_READBACK_ENABLED=true`.
- `cmd/semops/main_test.go` proves configured hosted wiring performs graph target lookup, writes a command-intent
  create mutation, and returns no-transmit response posture.
- `internal/app/runtime_test.go` covers the new config flag and write timeout parsing/validation.

Residual risk:

- This still does not contact a live SemLink node, BlueOS extension, Navigator hardware, or ArduPilot vehicle.
- The route assumes the caller supplies a born target asset ID; route/source discovery for SemLink mesh nodes remains
  a future slice.
- Operator authentication and request signing are still outside this slice; deployments should keep the flag disabled
  until an upstream auth boundary is in place.
