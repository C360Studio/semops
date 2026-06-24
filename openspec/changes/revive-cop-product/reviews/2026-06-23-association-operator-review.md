# Association Operator Review

Scope: COP API/UI acknowledge and challenge affordances for fusion association evidence.

## Verdict

Accept this slice as the first operator-visible safety control for association evidence. It lets the demo show
human-in-the-loop review without letting association scoring imply track merge, identity resolution, or feed authority.

## Findings

1. The review path is intentionally narrow.
   `POST /api/cop/associations/{id}/review` accepts only `acknowledged` and `challenged`, rejects unknown association
   IDs, and overlays the review in the COP snapshot. That is enough for operator affordance evidence.

2. The review path is not product-grade arbitration yet.
   The current store is API-local memory. It is acceptable for demo UX and e2e proof, but it is not a durable audit
   trail, governance claim, identity assertion, command acknowledgement, or compliance artifact.

3. Source-owned state remains protected.
   Review state is attached beside association evidence and does not mutate MAVLink, TAK, ADS-B, SAPIENT, or other
   feed-owned tracks. That preserves the SemStreams ownership posture.

## Boundaries

- This is not identity fusion.
- This is not a source-track merge.
- This is not persisted operator audit.
- This is not command authority, tasking, or upstream CS API status.
- This is not default enablement of automatic association in the demo stack.

## Follow-Ups

- Add durable graph-backed review/audit contracts before review decisions become command, identity, or compliance
  authority.
- Revisit conflict semantics when local and upstream operators challenge or acknowledge the same association.
- Add role/authority metadata when SemOps gets authentication and operator identity.

