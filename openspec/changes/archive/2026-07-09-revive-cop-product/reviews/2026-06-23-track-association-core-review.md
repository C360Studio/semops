# Track Association Core Review

Date: 2026-06-23
Scope: pure statistical track-association scorer and fusion-owned association contract

## Decision

Accept the first track-association slice as pure fusion evidence. It is enough to close the statistical association
core gate, but not enough to claim hosted association service behavior, graph projection, COP API readback, UI
semantics, or identity merge.

## Red-Team Findings

1. Association is evidence, not truth.

   The scorer emits confidence, algorithm identity, distance/time evidence, source-track IDs, and ambiguity status.
   It does not merge tracks, rewrite source positions, or decide source authority.

2. Feed owners must not backdoor association.

   MAVLink, ADS-B, SAPIENT, and TAK track contracts remain source-partitioned. The new association contract is owned by
   `semops.fusion.structural` and uses strict born-first edges back to already-born source tracks.

3. Ambiguity must stay visible.

   When two candidates are close in score, the result is marked `ambiguous` instead of pretending the best candidate is
   a clean identity match. That posture matters for HADR airspace where multiple aircraft may cluster near the same
   route or landing zone.

4. No scenario or UI claim yet.

   The package is graph-free and UI-free. A later projector/readback slice must prove source-track discovery, owner
   token use, graph writes, and operator-facing display before demo copy claims association in the COP.

## Accepted Evidence

- `pkg/cop` now declares `association` as a control-profiled COP entity and adds
  `FusionTrackAssociationContract`.
- The fusion association contract derives two strict `ForeignEdgeClaim` values targeting existing track entities.
- `internal/fusion/association` scores geotemporal cross-source candidates and preserves source references.
- Tests cover close MAVLink/ADS-B association, ambiguous candidates, far/stale rejection, same-source rejection, and
  fusion ownership contract derivation.

## Follow-Ups

- Add a projector that writes association evidence through SemStreams owner tokens.
- Add COP API/UI readback only after the graph path is proven against scenario data.
- Revisit spatial-temporal query helper asks if association matching becomes query-bound rather than in-memory.
