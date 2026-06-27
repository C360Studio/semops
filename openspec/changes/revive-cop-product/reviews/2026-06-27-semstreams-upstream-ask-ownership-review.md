# SemStreams Upstream Ask Ownership Review

Date: 2026-06-27
Scope: remaining `revive-cop-product` task group 9 upstream SemStreams ask candidates.

## Decision

File one non-blocking SemStreams issue for provenance, confidence, observed-at, and source-reference conventions in
projection contracts and graph-visible readback: SemStreams issue
[#367](https://github.com/C360Studio/semstreams/issues/367).

Defer the other remaining upstream candidates until they have sharper evidence:

- Manifest/tier placement remains deferred until placement answers a concrete operator decision beyond health/source
  state.
- Escalation event/status vocabulary remains deferred until semantic or statistical tier transitions generalize across
  more than one workflow.
- Spatial-temporal query helpers remain deferred until SemOps has a query-bound failure rather than local parsing,
  scoring, and readback code.
- Indexing profile/cardinality helpers remain deferred until a mixed-feed fixture proves clean entity boundaries are
  insufficient.

This closes the adversarial ownership review gate for task `9.7`, but it does not close the deferred ask tasks.

## Candidate Outcomes

- `9.1` manifest/tier placement: defer. The service-promotion matrix and runtime health already answer current
  operator questions. Local Compose/service metadata is not yet a framework scheduling or placement primitive.
- `9.2` escalation event/status vocabulary: defer. Semantic explanations are deterministic read-only artifacts, and
  association scoring is evidence projection. There is no reusable escalation lifecycle across tiers yet.
- `9.3` provenance/confidence convention: file. Multiple feed and fusion contracts now duplicate the same
  graph-visible source, source-ref, observed-at, and confidence posture.
- `9.4` spatial-temporal query helpers: defer. SemOps has geospatial pressure, but the current implementation is
  local WKT parsing, Haversine scoring, freshness checks, and UI readback rather than a SemStreams graph-query blocker.
- `9.6` indexing profile/cardinality helpers: defer. Current `signal`, `control`, `content`, and `trace` profiles
  still hold when SemOps splits current state, durable control, prose content, and replay trace into separate entity
  families.

## Evidence Checked

- `pkg/cop/contracts.go` repeats provenance predicates across source assets, MAVLink, TAK/CoT, CAP, weather, ADS-B,
  SAPIENT, KLV, command intent, fusion association, and association review contracts.
- `internal/api/cop/graph_provider.go` hydrates provenance and confidence into the COP snapshot for tracks, assets,
  tasks, advisories, hazards, footprints, weather, associations, and reviews.
- `internal/semantic/explanations.go` preserves source owner, source ref, observed time, confidence, and trajectory
  references in operator-facing explanation artifacts.
- `internal/egress/csapi` and `internal/egress/semconnect` carry source/provenance posture through the standards
  bridge without letting CS API decide indexing or ownership.
- SemStreams issue `#340` accepted raw-lane/current-state guidance and intentionally deferred source-reference
  vocabulary/helper contracts until more feeds proved the reusable shape. SemOps now has that repeated feed pressure.
- Duplicate check: existing SemStreams issues `#213`, `#214`, and `#216` cover LLM claim/evidence retrieval,
  extraction, and promotion. They do not cover deterministic projection-contract provenance conventions.

## Why The Provenance Ask Belongs Upstream

SemOps can continue carrying `cop.provenance.*` predicates locally, but the pattern is no longer COP-specific. The
same graph-visible fields answer product-neutral questions:

- Which source produced this state?
- Which raw/replay/storage reference supports it?
- When was it observed?
- What confidence did the projection assign?
- How do those graph-visible fields relate to `message.Triple` metadata, ownership claims, and raw-lane references?

SemStreams already owns projection contracts, graph mutation/query semantics, ownership governance, message triple
metadata, and reusable docs/pattern guidance. A SemStreams convention can reduce drift without forcing SemOps
vocabulary upstream.

## Rejected Product-Local Option

Keeping every product on local `*.provenance.*` predicates works short term, but it makes SemOps, SemSource,
SemConnect, SemTeams, and future feed products rediscover the same convention and explain the same boundary between
graph-visible provenance, triple metadata, ObjectStore/raw references, and ownership arbitration.

## Non-Goals For The Upstream Issue

- Do not add a mandatory PROV-O runtime model.
- Do not replace `message.Triple` source/timestamp/confidence metadata.
- Do not treat provenance as ownership arbitration.
- Do not solve LLM claim/evidence promotion; SemStreams already tracks that separately.
- Do not upstream SemOps COP predicate names.

## Follow-Up

- Track SemStreams issue `#367` as a docs/pattern or optional helper ask.
- Task `9.3` is complete with issue `#367`.
- Keep tasks `9.1`, `9.2`, `9.4`, and `9.6` open until their defer conditions produce concrete evidence.
