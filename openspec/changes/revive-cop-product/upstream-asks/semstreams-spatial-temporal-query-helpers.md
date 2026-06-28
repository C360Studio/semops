# Spatial-Temporal Graph Query Helper Guidance

## Summary

SemOps COP now has repeated downstream pressure for a SemStreams-level recipe around spatial-temporal graph queries.
This is not a blocker for SemOps, and it is not a request for SemStreams to become a full GIS engine. SemStreams
already has useful primitives: typed prefix queries, spatial bounds/polygon query handlers, temporal range queries,
GeoJSON helpers, and GraphQL gateway fields. The gap is the reusable composition pattern COP-style products need when
they have source-scoped current-state entities with location and observed-time evidence.

Filed upstream as [SemStreams issue #368](https://github.com/C360Studio/semstreams/issues/368). SemStreams closed the
issue as answered by the spatial-temporal recipe PR
[#369](https://github.com/C360Studio/semstreams/pull/369) and opened temporal-index follow-up
[#370](https://github.com/C360Studio/semstreams/issues/370), which landed in SemStreams `v1.0.0-beta.119`.

## Upstream Outcome

SemStreams chose the normalization path rather than configurable per-product predicate extraction:

- Spatial indexing stays on canonical numeric `geo.location.latitude` / `geo.location.longitude` triples.
- Product predicates such as `cop.track.position`, WKT, and footprint geometry stay product-side for rendering and
  inspection.
- Products compose existing query primitives: prefix/type scope, spatial query, temporal query, client-side
  intersection, then `graph.query.batch` hydration.
- A future `graph.query.spatialTemporal` helper is gated behind measured cross-product need, not built up front.

The review also identified a real temporal-index gap. SemStreams issue
[#370](https://github.com/C360Studio/semstreams/issues/370) landed in `v1.0.0-beta.119`: the temporal index now uses
`time.observation.recorded` as the primary event-time key, with framework-managed `UpdatedAt` as fallback.

## Downstream Evidence

SemOps currently composes the workflow locally:

- Fusion candidate production queries configured source-owned track prefixes through `graph.query.prefix`, caps tracks
  per source, parses WKT `POINT` track positions, and then scores candidates with local distance, observed-time delta,
  observation-age, confidence, and ambiguity thresholds.
- The COP snapshot provider uses prefix discovery to hydrate tracks, tasks, advisories, hazards, sensor footprints,
  weather observations, fusion associations, and source health, then applies local WKT parsing and freshness status.
- Weather evidence carries query shape, WKT query geometry, valid time, model time, and fresh-until time so later
  tactical workflows can ask "what evidence applies near this place and time?" without turning weather into route
  authority.
- KLV/MISB evidence carries sensor position, frame center, and optional footprint polygons as graph predicates that
  are visible to the COP map and inspector, but not yet part of a shared spatial-query convention.

## Existing SemStreams Pieces

The SemStreams checkout already has the ingredients, but they are not yet packaged as a COP-ready helper:

- `graph.PrefixQueryRequest` / `graph.PrefixQueryResponse` page full `EntityState` values by prefix with opaque
  cursors.
- `graph.query.batch` can hydrate candidate ID sets after client-side intersection.
- `processor/graph-index-spatial` registers `graph.spatial.query.bounds` and `graph.spatial.query.polygon`, returning
  entity IDs plus indexed coordinates.
- `processor/graph-index-temporal` registers `graph.temporal.query.range`, returning entity IDs for a time range.
- `graph/geo/geojson` provides RFC 7946 geometry types and point-in-polygon support, while deliberately avoiding full
  GIS scope.
- `graph-gateway` exposes separate GraphQL `spatialSearch` and `temporalSearch` fields.
- `graph/query.SearchOptions` can describe geo bounds and time ranges, but the lower-level helper contract for
  source-scoped state hydration is still unclear.

## Gaps To Clarify

Please consider adding docs and optional helper contracts for:

- how product code should compose prefix/type scope, spatial scope, and observed-time scope without broad prefix scans;
- whether a reusable request/response helper should return only IDs, full `EntityState` values, or IDs plus a bounded
  batch-hydration helper;
- what truncation diagnostics should look like when spatial, temporal, prefix, and hydration caps differ;
- which timestamp should be indexed for current state: entity `updated_at`, triple timestamp, graph-visible
  observed-at predicate, or a caller-selected predicate;
- whether spatial extraction should support configured WKT/GeoJSON predicates such as `cop.track.position` and
  `cop.weather.query_geometry`, or whether products should also publish generic numeric lat/lon predicates;
- how coordinate-order rules should be stated when products use WKT lon/lat and SemStreams GeoJSON uses
  RFC 7946 lon/lat;
- which spatial shapes are implemented today: bounds and single-polygon containment are real request subjects, while
  radius, geohash-prefix, MultiPolygon, antimeridian handling, and richer geometry operations should be explicitly
  documented as helpers, follow-ups, or non-goals;
- how GraphQL gateway fields relate to NATS request/reply subjects for hosted processors that do not go through
  GraphQL.

## Ask

Please consider adding a SemStreams docs/pattern guide, and optional helper APIs only if useful, for
spatial-temporal graph queries over governed current-state entities.

The first useful shape could be a recipe and testable examples for:

- bounded prefix query plus spatial/temporal index intersection;
- batch hydration of candidate entity IDs back to `EntityState`;
- configurable location and observed-time predicate extraction;
- explicit freshness/truncation diagnostics;
- GeoJSON/WKT coordinate and geometry-shape conventions;
- examples that are product-neutral rather than SemOps COP vocabulary.

## Non-Goals

- Do not upstream SemOps-specific `cop.*` predicates.
- Do not make SemStreams a general-purpose GIS, routing, weather, or mission-planning engine.
- Do not include SemOps fusion scoring, identity merge, operator authority, or route-safety decisions in the helper.
- Do not require every product to duplicate location as both WKT/GeoJSON and generic numeric lat/lon unless that is the
  documented migration convention.
- Do not treat the ask as a blocker for SemOps; local prefix discovery and post-filtering can continue while the shared
  convention matures.

## Why SemStreams

The pattern touches SemStreams-owned surfaces: graph query contracts, spatial and temporal index processors, gateway
query routing, GeoJSON helpers, entity hydration, payload/error conventions, and cross-product docs. SemOps can keep
local WKT parsing and scoring, but a canonical recipe would reduce drift across SemOps, SemConnect, SemSource,
SemTeams, and future C360 feed products that need "state near this place during this time" without inventing separate
query glue.

## SemOps References

- SemOps OpenSpec task: `revive-cop-product` task `9.4`
- SemStreams issue: <https://github.com/C360Studio/semstreams/issues/368>
- SemStreams recipe PR: <https://github.com/C360Studio/semstreams/pull/369>
- SemStreams temporal-index follow-up: <https://github.com/C360Studio/semstreams/issues/370>
- SemOps review:
  `openspec/changes/revive-cop-product/reviews/2026-06-28-semstreams-spatial-temporal-query-helper-review.md`
- Related SemStreams spatial result issue: <https://github.com/C360Studio/semstreams/issues/95>
- Related SemStreams batch hydration/prefix issue: <https://github.com/C360Studio/semstreams/issues/172>
