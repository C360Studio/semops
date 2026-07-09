# CS API Write-Side Ingress Review

Status: accepted as command-intent-only ingress.

## Scope

- Add a pure CS API write-side mapper for Command and ControlStream command input.
- Route mapped input through the existing guarded command-intent projector.
- Keep hosted CS API service handling, native transmit, scheduling, and upstream command-status publication out of
  scope.

## Decision

`internal/ingress/csapi` is acceptable because it produces governed SemOps command intent only. It requires the command
impedance fields that matter before projection: authenticated authority, target, priority, TTL/deadline, idempotency
key, correlation ID, requested-by, desired state, provenance, and local override policy. Admission still uses the
command projector's born-target, expiry, and idempotency checks.

The command-intent contract now carries `cop.task.local_override_policy`, closing the gap between the OpenSpec wording
and the graph state available to native execution gates later.

## Boundaries

- This is not a hosted CS API Command or ControlStream endpoint.
- This is not SemConnect command-status publication.
- This is not native MAVLink, TAK, SAPIENT, DJI, or other feed actuation.
- The ingress result explicitly reports that native execution and upstream status publication are not authorized.

## Verification

- `go test ./internal/ingress/csapi ./internal/projectors/command ./internal/api/cop ./pkg/cop`
- `go test ./...`
- `openspec validate revive-cop-product --strict`
- `git diff --check`
