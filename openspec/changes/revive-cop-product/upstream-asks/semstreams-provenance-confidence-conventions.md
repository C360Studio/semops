# SemStreams Provenance And Confidence Convention Ask

Status: filed as SemStreams issue [#367](https://github.com/C360Studio/semstreams/issues/367).

## Summary

SemOps now has repeated downstream pressure for a SemStreams-level convention around graph-visible provenance,
confidence, observed-at, and source-reference fields in projection contracts. This is not a blocker for SemOps, but it
is framework-shaped enough that sibling products should not each invent local predicate names and documentation.

## Downstream Evidence

SemOps repeats the same provenance posture across many feed and fusion surfaces:

- MAVLink source assets, tracks, and command ACK readback.
- TAK/CoT source assets, operator tracks, tasks, and advisories.
- CAP advisory/hazard evidence.
- Weather observations with provider, query shape, model/valid/freshness time, and source confidence.
- ADS-B current-state tracks, including lower-confidence no-position evidence.
- SAPIENT absolute-location detection tracks.
- KLV sensor/frame-center and footprint evidence with packet/media refs.
- Command intent, cancellation, and status reconciliation.
- Fusion association and association-review evidence.
- CS API/SemConnect read-side bridge payloads that must preserve SemOps provenance without owning the decision.

The repeated question is not product vocabulary. It is how SemStreams products should consistently expose:

- source system or producer;
- raw/replay/storage source reference;
- observed time;
- projection confidence;
- relationship to `message.Triple` source/timestamp/confidence metadata;
- relationship to ownership claims and owner tokens.

## Ask

Please consider adding a SemStreams docs/pattern guide, and optional helper constants or predicate-group helpers only
if useful, for graph-visible provenance and confidence conventions in projection contracts.

The first version could clarify:

- when graph-visible provenance predicates are useful in addition to `message.Triple` metadata;
- recommended names or vocabulary namespaces for source, source reference, observed time, and confidence;
- how source references relate to raw lanes, replay fixtures, ObjectStore references, and typed artifacts;
- how provenance differs from ownership arbitration and owner-token enforcement;
- how read models should hydrate provenance without implying authority, compliance, or native execution;
- a small example that is not tied to SemOps COP vocabulary.

## Non-Goals

- Do not require every triple to duplicate metadata as graph predicates.
- Do not add a mandatory PROV-O runtime, ObjectStore dependency, or raw-payload service.
- Do not solve LLM-derived claim/evidence promotion; those have separate SemStreams issues.
- Do not move SemOps-specific `cop.*` predicates into SemStreams.
- Do not treat provenance confidence as fusion confidence, reviewer authority, or ownership status.

## Why SemStreams

The pattern touches SemStreams-owned surfaces: projection contracts, graph mutation/query contracts, message triple
metadata, ownership governance, raw-lane guidance, typed artifact references, and cross-product documentation. SemOps
can continue locally, but a canonical convention would reduce drift across SemOps, SemSource, SemTeams, SemConnect,
and future C360 feed consumers.

## SemOps References

- SemOps OpenSpec task: `revive-cop-product` task `9.3`
- SemStreams issue: <https://github.com/C360Studio/semstreams/issues/367>
- SemOps review:
  `openspec/changes/revive-cop-product/reviews/2026-06-27-semstreams-upstream-ask-ownership-review.md`
- Related accepted SemStreams context: issue `#340`, raw-lane plus current-state projection guidance
