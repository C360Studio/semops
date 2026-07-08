# Upstream Ask Deferral Refresh

Date: 2026-07-08
Scope: Remaining open `revive-cop-product` task group 9 upstream SemStreams ask candidates.

## Decision

Keep tasks `9.1`, `9.2`, and `9.6` open. Do not file new SemStreams issues from this checkpoint.

The SemLink companion-contract closeout and SemStreams `TimerPort` adoption did not create new reusable framework
pressure for:

- manifest/tier placement;
- escalation event/status vocabulary; or
- indexing profile/cardinality helpers.

## Evidence Checked

- `reviews/2026-06-27-semstreams-upstream-ask-ownership-review.md` already defers `9.1` and `9.2` until placement or
  escalation semantics answer a reusable cross-product question.
- `reviews/2026-06-28-semstreams-indexing-cardinality-review.md` already defers `9.6` until clean entity boundaries or
  ADR-054 profiles prove insufficient.
- SemStreams issue `#312` closed the concrete `TimerPort` cadence-boundary need and SemOps adopted it in
  `v1.0.0-beta.144`.
- The accepted SemLink companion contract remains a SemOps/SemLink native governance contract with CS API/SemConnect
  at the interop edge; it does not require a new SemStreams placement, escalation, or cardinality primitive.

## Follow-Up

- Revisit `9.1` only when manifest or tier placement changes an operator decision beyond local health/source state.
- Revisit `9.2` only when semantic or statistical tier transitions generalize across more than one workflow.
- Revisit `9.6` only when a mixed-feed workflow proves the current entity/profile split is insufficient, or multiple
  products duplicate profile/cardinality helper tests.
