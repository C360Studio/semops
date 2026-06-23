# Weather Feed Evidence

Status: critical COP layer with first Open-Meteo-shaped point, OGC EDR-shaped point, OGC EDR-shaped spatial parser
fixtures, SemStreams component-flow evidence for point payloads, and a governed tactical-weather graph contract. It is
not yet implemented as a live weather provider, graph-writing runtime feed, conformance target, UI layer, or
routing/safety authority.

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
  supports discovery plus position, area, trajectory, and corridor retrieval patterns. The OGC API EDR 1.1 standard
  also defines radius, cube, items, locations, and instances query resources; SemOps only has point, area, trajectory,
  and corridor fixture evidence today.
- MSC GeoMet is a strong public interoperability target because it exposes ECCC/MSC data through OGC APIs and WMS/WCS.
- Open-Meteo is a useful JSON source for early fixtures and non-compliance demo telemetry.
- `fixtures/weather/open-meteo-point.json` is the first selected provider-shaped fixture. It is deterministic point
  forecast telemetry for parser acceptance only, not a live provider or service-reliability claim.
- `fixtures/weather/ogc-edr-position.json` is a synthetic OGC EDR-shaped CoverageJSON point fixture. It is a
  storage/governance/parser proof for point query handling, not an official OGC ETS run, live EDR server response, or
  conformance sample.
- `fixtures/weather/ogc-edr-area.json`, `fixtures/weather/ogc-edr-trajectory.json`, and
  `fixtures/weather/ogc-edr-corridor.json` are synthetic OGC EDR-shaped spatial fixtures. They prove simple WKT
  `POLYGON`, `LINESTRING`, and corridor width/height parsing for time-series tactical variables; they do not prove
  provider-specific CoverageJSON dimensionality, Z/M time-coordinate handling, route-safety decisions, or standards
  conformance.
- `pkg/adapters/weather` now parses Open-Meteo-shaped and OGC EDR-shaped point forecasts and preserves provider,
  query shape, position, elevation, units, sample time, temperature, precipitation, visibility, surface pressure, wind
  speed, gusts, wind direction, and weather code without graph writes.
- `pkg/adapters/weather` also parses OGC EDR-shaped area, trajectory, and corridor fixtures into spatial forecast
  preflight evidence without graph writes or point-payload promotion.
- `pkg/cop` now defines `weather_observation` as localized tactical weather signal evidence under
  `semops.feed.weather`, with source-partitioned ownership and no hazard, alert, task, or route-decision predicates.
- `internal/projectors/weather` now contains a pure mutation-plan projector for weather observations. It proves the
  graph contract and owner-token fence without wiring a graph writer, runtime component, UI, or route-safety rule.
- `internal/components/weather` wraps provider-shaped weather fixtures as SemStreams file input components and decoder
  processors with registered payloads, file/NATS ports, config schema, health, and flow metrics for point forecasts.
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
  forecast fixture done; OGC EDR-shaped point, area, trajectory, and corridor CoverageJSON fixtures done; broader EDR
  query shapes still open]
- Preserve wind, gusts, visibility, pressure, precipitation, temperature, weather code, source time, provider, query
  geometry, and freshness. [partial: Open-Meteo and OGC EDR point forecast variables, provider, point geometry, units,
  and sample time done; OGC EDR spatial fixtures preserve simple WKT geometry, units, and sample time; stale/freshness
  policy remains a component/projection gate]
- Publish raw and decoded provider-shaped weather forecasts through SemStreams registered payloads and NATS stream
  ports without graph writes. [done for Open-Meteo-shaped and OGC EDR-shaped point fixtures]
- Project localized tactical weather variable samples as `signal`, route/weather decision state as `control`, advisory
  text as `content`, and provider/replay diagnostics as `trace`. [partial: graph contract and pure planner done; graph
  writer/runtime component still open]
- Keep browser-only visual layers out of graph state unless an operator workflow turns them into evidence.

## Known Gaps

- No OGC EDR radius, cube, item, location, or instance fixture exists yet.
- No OGC EDR spatial runtime component payload, graph writer, route-weather model, or UI tactical-weather layer exists
  yet.
- No OGC EDR conformance/ETS run, live EDR server capture, or standards-facing bridge test exists yet.
- No live Open-Meteo, NWS forecast/observation, MSC GeoMet, or radar/tile provider integration exists yet.
- No weather graph writer, runtime wiring, cache/stale policy, or UI tactical layer exists yet.
- No weather routing/safety rule is accepted yet.
- No tile/radar source has passed license, cache, or reliability review.

## Verification

- `go test ./pkg/adapters/weather`
- `go test ./internal/components/weather`
- `go test ./internal/contracts`
- `go test ./...`

## Source Links

- OGC API - Environmental Data Retrieval: <https://ogcapi.ogc.org/edr/>
- OGC API - Environmental Data Retrieval Standard 1.1: <https://docs.ogc.org/is/19-086r6/19-086r6.html>
- NWS API documentation: <https://www.weather.gov/documentation/services-web-api>
- MSC GeoMet OGC API: <https://api.weather.gc.ca/>
- Open-Meteo API docs: <https://open-meteo.com/en/docs>
