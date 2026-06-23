# Command Intent Projector Review

Date: 2026-06-23

Scope:

- Add a pure command-intent projector for desired tasking state.
- Validate authority, priority, expiry, correlation, idempotency, requested-by, desired-state, and target fields.
- Keep CS API ingress, local UI controls, target resolution, and native command transmission out of scope.

## Findings

1. The projector is the right next pressure test after the contract.
   It makes the desired-state contract executable and catches missing TTL, authority, priority, idempotency, and
   correlation fields before any bridge or transmitter depends on them.

2. It must not solve target birth.
   The strict `cop.task.target` edge points at an existing source asset. The projector deliberately writes only the
   command-intent task; a future CS API or operator component must resolve the target asset first or reject/defer the
   request.

3. It must not become a hidden transmitter.
   The projector produces SemStreams graph mutation plans only. No MAVLink, TAK/CoT, DJI, or SAPIENT command bytes are
   emitted, and no native status semantics are inferred from desired state.

4. The current status model remains intentionally narrow.
   `requested`, `cancelled`, or other desired-state statuses are control-plane intent. Native ACKs, partial execution,
   timeout, stale rejection, duplicate rejection, and link loss remain feed/status evidence to reconcile later.

## Decision

Accept the pure projector as the command-intent planning gate. Do not wire CS API ingress, UI controls, local override,
or native transmit until executable fixtures cover target resolution, stale-command rejection, cancellation,
supersession, duplicate/idempotent replay, priority arbitration, and async status reconciliation.

## Verification

- `go test ./internal/projectors/command`
