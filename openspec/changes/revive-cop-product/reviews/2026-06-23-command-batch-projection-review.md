# Command Batch Projection Review

Scope: guarded command-intent admission, arbitration, and status projection before native execution

## Decision

Accept the guarded batch projection path as the first graph-ready command policy seam. The order is admission,
arbitration, command-intent status projection, and only then future native reconciliation. Rejected and duplicate
commands do not enter arbitration. Superseded commands are projected as command-intent graph state, not exposed to
native drivers.

## Adversarial Findings

- Projecting each admitted command independently before arbitration can leave multiple `requested` commands for the
  same target, inviting a future driver to execute the wrong one.
- Running arbitration before target/idempotency admission wastes policy decisions on commands that should fail closed
  before graph planning.
- Batch projection still is not a CS API endpoint or native command service. It is a reusable control-plane seam for
  future ingress and executor components.
- Accepted/superseded graph status is still incomplete lifecycle support. Cancellation, supersession reason details,
  native ACK reconciliation, timeout, link loss, and partial execution need separate status evidence before demo claims.

## Acceptance

- Batch projection records one admission result per input.
- Duplicate idempotency and unresolved-target commands are skipped before arbitration.
- Arbitration sees only admitted intents.
- Accepted and superseded decisions are projected through the command-intent graph contract.
- Native execution candidates contain accepted decisions only.

## Result

Accepted as a graph-ready command policy gate. Keep CS API handlers, UI controls, cancellation endpoints, native
safety interlocks, and live transmit behind later reviews.
