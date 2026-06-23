# Weather Feed Evidence

Status: critical COP layer with first parser fixture and SemStreams component-flow evidence; not yet implemented as a
live weather provider, graph-writing feed, or routing/safety authority.

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
- `fixtures/weather/open-meteo-point.json` is the first selected provider-shaped fixture. It is deterministic point
  forecast telemetry for parser acceptance only, not a live provider or service-reliability claim.
- `pkg/adapters/weather` now parses Open-Meteo-shaped point forecasts and preserves provider, query shape, position,
  elevation, units, sample time, temperature, precipitation, visibility, surface pressure, wind speed, gusts, wind
  direction, and weather code without graph writes.
- `internal/components/weather` wraps the provider-shaped Open-Meteo fixture as a SemStreams file input component and
  decoder processor with registered payloads, file/NATS ports, config schema, health, and flow metrics.
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

- Parse deterministic Open-Meteo-shaped or OGC EDR-shaped JSON without graph writes. [partial: Open-Meteo-shaped point
  forecast fixture done; OGC EDR-shaped fixture still open]
- Preserve wind, gusts, visibility, pressure, precipitation, temperature, weather code, source time, provider, query
  geometry, and freshness. [partial: Open-Meteo point forecast variables, provider, point geometry, units, and sample
  time done; stale/freshness policy remains a component/projection gate]
- Publish raw and decoded provider-shaped weather forecasts through SemStreams registered payloads and NATS stream
  ports without graph writes. [done for Open-Meteo-shaped point fixture]
- Project localized tactical weather as `signal`, route/weather decision state as `control`, advisory text as
  `content`, and provider/replay diagnostics as `trace`.
- Keep browser-only visual layers out of graph state unless an operator workflow turns them into evidence.

## Known Gaps

- No OGC EDR-shaped fixture or standards-facing bridge test exists yet.
- No live Open-Meteo, NWS forecast/observation, MSC GeoMet, or radar/tile provider integration exists yet.
- No weather graph projector, ownership claim, runtime wiring, cache/stale policy, or UI tactical layer exists yet.
- No weather routing/safety rule is accepted yet.
- No tile/radar source has passed license, cache, or reliability review.

## Verification

- `go test ./pkg/adapters/weather`
- `go test ./internal/components/weather`
- `go test ./internal/contracts`
- `go test ./...`

## Source Links

- OGC API - Environmental Data Retrieval: <https://ogcapi.ogc.org/edr/>
- NWS API documentation: <https://www.weather.gov/documentation/services-web-api>
- MSC GeoMet OGC API: <https://api.weather.gc.ca/>
- Open-Meteo API docs: <https://open-meteo.com/en/docs>
