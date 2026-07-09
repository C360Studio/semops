# Command Deadline Reconciliation Review

Scope: pure deadline reconciliation for expired and timed-out command intent

## Decision

Accept a pure deadline reconciler for command intent. It maps requested commands whose deadline has passed to
`expired`, and accepted, executing, or cancel-requested commands whose deadline has passed to `timeout`. The projection
uses the same status-only update path as native readback and does not rewrite desired state, authority, priority, or
target edges.

## Adversarial Findings

- Deadline policy must be separate from handler latency. A delayed CS API request, replay, or driver restart must not
  turn an old requested command into a fresh native action.
- Expired and timed out are different states. `expired` means the command was not accepted before the deadline;
  `timeout` means an accepted/executing/cancel-requested command failed to reach terminal native evidence in time.
- This is not a scheduler. A hosted component still needs polling/query support, state checkpoints, metrics, and
  backpressure behavior before SemOps can claim runtime deadline enforcement.
- Terminal commands remain sticky. Succeeded, failed, cancelled, rejected, duplicate, superseded, expired, or timed-out
  commands cannot be reclassified by a later deadline check.

## Acceptance

- Requested commands past deadline produce `expired`.
- Accepted, executing, and cancel-requested commands past deadline produce `timeout`.
- Deadline updates do not emit target, desired-state, authority, or priority triples.
- Early deadline checks and terminal current commands are rejected.

## Result

Accepted as a policy/projection seam. Keep hosted scheduler, CS API status egress, native cancellation ACK, link-loss,
partial execution, and live transmit behind later reviews.
