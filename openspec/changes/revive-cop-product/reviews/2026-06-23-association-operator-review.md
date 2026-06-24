# Association Operator Review

Scope: COP API/UI acknowledge and challenge affordances for fusion association evidence.

## Verdict

Accept this slice as the first operator-visible safety control for association evidence. It lets the demo show
human-in-the-loop review without letting association scoring imply track merge, identity resolution, or feed authority.

## Findings

1. The review path is intentionally narrow.
   `POST /api/cop/associations/{id}/review` accepts only `acknowledged` and `challenged`, rejects unknown association
   IDs, writes hosted review state through a fusion-owned `association_review` graph audit entity, and overlays the
   review in the COP snapshot. That is enough for operator affordance evidence.

2. The review path is not product-grade arbitration yet.
   Hosted mode now has durable graph audit and fixed non-authoritative semantics:
   `reviewer_role=operator.unverified`, `authority_scope=local.display_only`, and
   `conflict_policy=latest_review_wins_display_only`. It still lacks authenticated operator identity, multi-authority
   conflict arbitration, upstream status semantics, and compliance policy. Fixture-only API mode may still use a local
   memory overlay.

3. Source-owned state remains protected.
   Review state is attached beside association evidence and does not mutate MAVLink, TAK, ADS-B, SAPIENT, or other
   feed-owned tracks. That preserves the SemStreams ownership posture.

## Boundaries

- This is not identity fusion.
- This is not a source-track merge.
- This is not authenticated operator authority.
- This is not local/upstream conflict resolution beyond latest local display readback.
- This is not command authority, tasking, or upstream CS API status.
- This is not default enablement of automatic association in the demo stack.

## Follow-Ups

- Revisit conflict arbitration when local and upstream operators challenge or acknowledge the same association.
- Add authenticated role/authority metadata when SemOps gets operator identity.
- Define command, identity, upstream CS API status, and compliance semantics before review decisions become authority.
