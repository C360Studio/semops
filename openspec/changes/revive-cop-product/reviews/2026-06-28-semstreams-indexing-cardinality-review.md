# SemStreams Indexing Profile And Cardinality Review

Date: 2026-06-28
Scope: `revive-cop-product` task `9.6`, mixed COP feed indexing profiles, and possible upstream SemStreams asks.

## Decision

Do not file a new SemStreams indexing-profile/cardinality issue yet.

The current four SemStreams profiles (`signal`, `control`, `content`, `trace`) are still adequate when SemOps keeps
entity boundaries clean:

- high-rate current-state telemetry stays `signal`;
- durable task, command, association, review, and lifecycle state stays `control`;
- advisory/chat/explanation prose stays `content`;
- raw packets, replay detail, decode traces, and debug records stay off graph or use `trace`.

The pressure is real, but it is already covered by SemStreams ADR-054 and the accepted raw-lane/current-state guidance.
SemOps should keep task `9.6` open until a mixed-feed workflow proves that clean entity boundaries are insufficient or
that products need a reusable multi-profile or field-level indexing policy.

## Evidence Checked

- `docs/feed-validation-and-indexing-ladder.md` names ADR-054 as the current SemStreams indexing-profile contract and
  requires every feed to pass parser, projection, replay, graph, compliance, and demo gates with explicit
  profile/cardinality review.
- `docs/feed-validation-and-indexing-ladder.md` already records the risky mixed-shape cases: TAK events, CAP alerts,
  weather layers, DJI telemetry/media/control, KLV video metadata and footprints, and fusion outputs.
- `docs/feed-validation-and-indexing-ladder.md` explicitly says not to invent more profiles first; split high-rate
  state, durable control state, textual content, and replay trace where storage shape differs.
- `pkg/cop/contracts.go` defines source-partitioned contracts for MAVLink, TAK, ADS-B, SAPIENT, KLV, weather, command
  intent, fusion association, and association-review entities with explicit `IndexingProfile` values.
- `pkg/cop/contracts_test.go` asserts the expected profile for each current feed and fusion contract.
- `internal/smoke/mavlink/live_graph_test.go` asserts SemStreams graph-ingest indexing-profile default counters do
  not move for the MAVLink/source-asset live graph smoke when metrics are available.
- `internal/api/cop/graph_provider.go` and `internal/api/cop/graph_provider_test.go` expose prefix discovery counts,
  limits, truncation, and query errors as COP diagnostics/alerts for mixed-feed cardinality pressure.
- SemStreams `graph.CreateEntityWithTriplesRequest` and `graph.UpdateEntityWithTriplesRequest` carry
  `indexing_profile` fields at the governed graph mutation boundary.
- SemStreams `pkg/projection.Contract` validates `content|control|signal|trace` profiles.
- SemStreams ADR-054 already frames graph visibility as storage/query contract and indexing as per-substrate policy,
  including `(indexing_profile, entity_type)` matrices and cardinality guards.
- SemStreams raw-lane/current-state projection guidance already maps feed shapes to profile cost posture.

## Why This Stays Local For Now

The repeated SemOps decisions are currently profile-assignment discipline, not a missing framework primitive. The
framework already has:

- the four profile values;
- birth-time profile stamping and explicit override hooks;
- projection-contract validation;
- defaulting metrics for unclassified writers;
- ADR-054 strict-mode preconditions, including shadow reports and skipped-entity metrics;
- a docs recipe for raw lanes plus current-state projection.

Opening another SemStreams issue now would likely duplicate issue `#340` or ask for new profiles before SemOps proves
the existing profile/entity split fails.

## Watch Conditions

Reopen and file task `9.6` if any of these happen:

- one COP entity must be partly `signal` and partly `content` despite clean entity splitting;
- a high-rate `control` family cannot be represented as separate trace/detail entities plus low-cardinality control
  state;
- SemOps needs reusable per-substrate examples for geospatial, weather, media, or fusion feeds beyond ADR-054 and the
  raw-lane guide;
- SemStreams lacks test helpers for asserting profile/cardinality behavior once multiple product repos duplicate the
  same tests;
- strict indexing-profile mode reaches implementation and SemOps needs a concrete profile/cardinality migration ask.

## Follow-Up

- Keep task `9.6` open and point it at this review as the current defer decision.
- Keep using SemOps contract tests, graph-smoke metrics, and COP discovery diagnostics as the local evidence gate.
- Revisit after SemStreams finishes issue `#340` or if the review of issue `#368` exposes profile/cardinality coupling.
