# Weather Feed Evidence

Status: critical COP layer, not yet implemented as a SemOps backend component.

## Decision

Weather is not one feed. SemOps should model it as three layers:

1. Visual context for the operator, usually browser-side tiles or WMS.
2. Alert/advisory evidence, already aligned with the CAP lane.
3. Tactical meteorological telemetry for points, incident areas, corridors, or routes when safety/routing logic needs
   raw values.

This keeps SemOps from dragging global model files into the graph while still making weather actionable when it
affects assets, operators, hazards, or routes.

## Source Posture

- OGC API - Environmental Data Retrieval is the standards-facing architecture for tactical weather queries because it
  supports position, area, trajectory, and corridor retrieval patterns.
- MSC GeoMet is a strong public interoperability target because it exposes ECCC/MSC data through OGC APIs and WMS/WCS.
- Open-Meteo is a useful JSON source for early fixtures and non-compliance demo telemetry.
- NWS API remains valuable for alerts, forecasts, and observations. Its alert endpoints fit the current CAP lane, and
  CAP can be requested through content negotiation.
- Radar and raster display data should not be assumed to come from `api.weather.gov`; visual products need their own
  source and cache/license review.

## Product Boundary

- Visual tiles can be a frontend layer only when they are human context.
- Tactical weather that influences drone safety, route scoring, incident status, or fusion must become governed graph
  evidence with provenance, provider, model time, query shape, freshness, and confidence.
- CAP/weather warnings append evidence and should not own stricter hazard truth until a dedicated hazard model earns
  that authority.

## First Acceptance Gates

- Parse deterministic Open-Meteo-shaped or OGC EDR-shaped JSON without graph writes.
- Preserve wind, gusts, visibility, pressure, precipitation, temperature, weather code, source time, provider, query
  geometry, and freshness.
- Project localized tactical weather as `signal`, route/weather decision state as `control`, advisory text as
  `content`, and provider/replay diagnostics as `trace`.
- Keep browser-only visual layers out of graph state unless an operator workflow turns them into evidence.

## Known Gaps

- No first weather provider fixture has been selected.
- No SemStreams component package exists for weather.
- No weather routing/safety rule is accepted yet.
- No tile/radar source has passed license, cache, or reliability review.

## Source Links

- OGC API - Environmental Data Retrieval: <https://ogcapi.ogc.org/edr/>
- NWS API documentation: <https://www.weather.gov/documentation/services-web-api>
- MSC GeoMet OGC API: <https://api.weather.gc.ca/>
- Open-Meteo API docs: <https://open-meteo.com/en/docs>
