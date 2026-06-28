# SemStreams Spatial-Temporal Query Helper Review

Date: 2026-06-28
Scope: `revive-cop-product` task `9.4`, COP spatial-temporal query pressure, and live SemStreams query/index surfaces.

## Decision

File one non-blocking SemStreams issue for spatial-temporal graph query helper guidance. The ask should be framed as a
composition/doc-pattern gap, not as "SemStreams has no spatial or temporal support." Filed as SemStreams issue
[#368](https://github.com/C360Studio/semstreams/issues/368).

Update after SemStreams triage: issue `#368` was closed as answered by recipe PR
[#369](https://github.com/C360Studio/semstreams/pull/369). SemStreams accepted the normalization direction rather than
per-product/per-source predicate extraction. Spatial indexing stays on canonical numeric `geo.location.latitude` /
`geo.location.longitude`; product WKT and `cop.*` geometry predicates remain for rendering and inspection.

The triage also produced SemStreams issue
[#370](https://github.com/C360Studio/semstreams/issues/370), which tracks changing the temporal index to key on
`time.observation.recorded` first, with framework-managed `UpdatedAt` as fallback. Until that lands, temporal queries
are write-time/freshness keyed because graph-ingest stamps `UpdatedAt` during ingest.

SemStreams already ships useful primitives:

- paged full-state prefix query via `graph.query.prefix`;
- batch hydration via `graph.query.batch`;
- spatial bounds and polygon-containment query subjects in `graph-index-spatial`;
- temporal range query subject in `graph-index-temporal`;
- RFC 7946 GeoJSON helpers with point-in-polygon support;
- GraphQL `spatialSearch` and `temporalSearch` gateway fields.

The COP gap is that hosted processors and the API snapshot need a predictable way to combine source/type prefix scope,
spatial scope, observed-time scope, batch hydration, and truncation diagnostics while preserving product-owned scoring
and authority decisions.

## Red-Team Findings

1. The pressure is now concrete enough to file, but still non-blocking.

   SemOps can continue with prefix discovery and local post-filtering. The issue is worth upstreaming because fusion,
   weather, KLV footprint readback, and tactical map filters are all converging on the same "state near place/time"
   pattern.

2. The sharpest mismatch is predicate extraction.

   SemStreams `graph-index-spatial` currently extracts numeric predicates such as `geo.location.latitude`,
   `geo.location.longitude`, `latitude`, and `longitude`. SemOps COP state writes WKT predicates such as
   `cop.track.position`, `cop.weather.query_geometry`, and sensor footprint geometry. A reusable convention needs to
   say whether products should publish generic numeric coordinates, configure indexed predicates, or get a WKT/GeoJSON
   extraction helper.

3. The helper must not absorb SemOps fusion logic.

   SemOps' association scorer owns distance thresholds, time-delta thresholds, observation-age windows, confidence
   scoring, ambiguity margins, source-priority tie-breakers, and operator review posture. SemStreams should only help
   find bounded candidates and explain query limits.

4. The API surface has doc/contract drift risk.

   SemStreams docs mention radius, geohash-prefix, time bucket, and recent query ideas, but the implemented request
   handlers read here are bounds, polygon, and temporal range. A docs-first ask should make implemented shapes and
   non-goals explicit before products build against imagined helpers.

5. GraphQL is not enough for hosted COP processors.

   The GraphQL gateway exposes separate spatial and temporal root fields, but SemOps hosted components use NATS
   request/reply and typed Go contracts. The reusable helper should work below GraphQL or clearly describe how
   processors should compose existing NATS subjects.

## Evidence Checked

- `internal/components/fusion/candidates.go` discovers source-owned tracks by `graph.query.prefix`, caps each source
  scan, parses WKT `POINT` values, and passes observations to the geotemporal scorer.
- `internal/fusion/association/association.go` applies Haversine distance, observed-time delta, stale-window,
  confidence, ambiguity, and source-priority logic locally.
- `internal/api/cop/graph_provider.go` hydrates prefix-discovered graph state into COP tracks, hazards, footprints,
  weather observations, associations, and freshness states with local WKT parsing.
- `docs/cop-ui-stack.md` records dynamic layer population pressure from feed capabilities, filters, time window, and
  source health.
- `docs/feed-validation-and-indexing-ladder.md` records OGC EDR-shaped point, area, trajectory, and corridor weather
  evidence with explicit route-safety and live-provider non-goals.
- SemStreams `graph/query_prefix_types.go` defines paged full-state prefix request/response types.
- SemStreams `processor/graph-index-spatial/query.go` implements bounds and single-polygon containment queries.
- SemStreams `processor/graph-index-temporal/query.go` implements temporal range queries.
- SemStreams `processor/graph-index-spatial/component.go` indexes only generic numeric coordinate predicates today.
- SemStreams `graph/geo/geojson/doc.go` states the package provides GeoJSON and point-in-polygon support, not full GIS.
- SemStreams `gateway/graph-gateway/component.go` routes GraphQL `spatialSearch` and `temporalSearch` separately.

## Upstream Ask Shape

Ask for a SemStreams docs/pattern guide, and optional helper APIs only if useful, covering:

- prefix/type plus spatial plus observed-time query composition;
- ID-result intersection and bounded batch hydration back to `EntityState`;
- configurable spatial and temporal predicate extraction;
- WKT/GeoJSON coordinate-order conventions;
- truncation diagnostics when each query leg has different caps;
- implemented-vs-deferred spatial shapes: bounds, polygon, radius, MultiPolygon, antimeridian, and richer GIS ops;
- NATS request/reply usage for processors as distinct from GraphQL gateway usage.

## Non-Goals

- Do not upstream SemOps `cop.*` vocabulary.
- Do not add route-safety, weather authority, identity merge, or operator command semantics.
- Do not require SemStreams to provide full GIS, buffering, union/intersection, or map rendering.
- Do not block SemOps COP progress on this ask.

## Follow-Up

- Track SemStreams PR `#369` for the recipe and SemStreams issue `#370` for event-time temporal indexing.
- Task `9.4` is complete with issue `#368`; SemOps adoption should follow the normalization path rather than chasing
  configurable edge cases.
