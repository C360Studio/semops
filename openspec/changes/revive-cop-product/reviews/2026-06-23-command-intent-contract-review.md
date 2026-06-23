# Command Intent Contract Review

Date: 2026-06-23

Scope:

- Add a product-owned command-intent graph contract before CS API ingress, local operator tasking, automation, or
  native feed transmitters write desired command state.
- Keep live native command/control, CS API request handling, local override, and transmitter safety out of scope.

## Findings

1. Desired intent must not be feed-owned.
   MAVLink, TAK/CoT, DJI, SAPIENT, and future feed drivers can publish native ACK/status evidence, but the desired
   command itself is SemOps control-plane state. The `semops.command.intent` owner avoids letting a feed adapter become
   the policy engine.

2. The contract closes a real impedance gap, not the whole feature.
   The contract includes authority, priority, expiry, correlation, idempotency, requested-by, desired-state, status,
   provenance, and a strict target edge. That is enough to prevent future writers from inventing ad hoc task cells, but
   it does not implement acceptance policy, cancellation, supersession, stale-command rejection, or native execution.

3. The strict target edge keeps command intent born-first.
   Desired commands target a source asset that must already exist. A future CS API bridge or local operator component
   must either resolve the target first or reject/defer the request; it must not depend on auto-vivify.

4. The current contract intentionally avoids command result semantics.
   ACKs, partial execution, timeout, link loss, and native rejection remain feed/status evidence. The next writer slice
   needs a reconciliation model that maps desired intent plus native readback into operator-visible status without
   blurring ownership.

## Decision

Accept the command-intent contract as the governed landing zone for future desired tasking state. Do not connect it to
CS API ingress, local UI controls, or MAVLink transmit until authority, priority, TTL/deadline, local override,
cancellation, supersession, and async status reconciliation are reviewed against executable fixtures.

## Verification

- `go test ./pkg/cop ./internal/copownership`

