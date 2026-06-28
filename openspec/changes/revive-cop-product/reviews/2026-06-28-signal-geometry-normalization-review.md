# Signal Geometry Normalization Review

Scope: follow-on SemOps adoption of the SemStreams spatial-temporal normalization path for non-track signal entities:
KLV sensor footprints and tactical weather observations.

## Verdict

Accept the narrow signal-geometry normalization slice.

KLV sensor-footprint and weather observation contracts now claim canonical numeric spatial predicates
`geo.location.latitude` / `geo.location.longitude` and canonical event-time predicate
`time.observation.recorded`. The product WKT predicates remain unchanged for map rendering and inspection.

## Boundary Choices

- KLV uses `cop.sensor_footprint.frame_center` as the canonical point because the entity represents observed footprint
  state. Platform sensor position remains product evidence and is not used as the footprint location fallback.
- Weather uses `cop.weather.valid_time` as `time.observation.recorded` so SemStreams temporal range queries preserve
  the existing "weather applies at this time" behavior.
- Weather emits canonical lat/lon only for WKT `POINT` query geometry. Area, trajectory, and corridor shapes remain
  product WKT until shared shape-indexing pressure is concrete.

## Evidence

- `pkg/cop/contracts.go` adds the canonical predicates to KLV sensor-footprint and weather observation contracts.
- `internal/projectors/klv` emits event time and frame-center lat/lon, and tests guard against indexing platform
  sensor position as footprint location.
- `internal/projectors/weather` emits valid time and point query lat/lon, and tests guard against indexing corridor
  geometry as a point.
- Focused tests passed for `./pkg/cop`, `./internal/projectors/klv`, and `./internal/projectors/weather`.

## Residual Risk

- TAK task markers and GeoChat advisories still use product WKT predicates only. Normalize them separately if COP map
  discovery needs SemStreams spatial/temporal query support for control/content entities.
- Rich shape indexing for weather corridors, weather areas, and KLV polygons remains out of scope until SemStreams has
  a product-neutral shape-indexing recipe beyond point queries.
