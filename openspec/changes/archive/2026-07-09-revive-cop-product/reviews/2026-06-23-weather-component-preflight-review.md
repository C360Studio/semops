# Weather Component Preflight Review

Date: 2026-06-23

Scope: fixture-backed weather input and decoder components for Open-Meteo-shaped tactical weather evidence.

## Decision

Accept the first weather component-flow slice as provider-shaped preflight evidence only.

The components are useful because they bring tactical weather into SemStreams lifecycle, payload registry, ports,
config schema, health, and flow metrics before runtime wiring or graph projection. They do not prove live Open-Meteo,
OGC EDR, MSC GeoMet, NWS forecast/observation, radar/tile hosting, cache/stale behavior, or route-safety authority.

## Adversarial Notes

### High: Fixture flow is not a weather gateway

`internal/components/weather` reads a deterministic provider-shaped fixture and emits raw/decoded BaseMessages. It is
framework-compliance evidence, not a live provider integration.

Mitigation:

- Keep Open-Meteo live polling, EDR query support, cache/stale policy, and provider reliability as separate gates.
- Do not wire default runtime weather ingestion until provider cadence, freshness, rate limits, and contact policy are
  reviewed.

### High: Weather values can silently become decisions

Wind, visibility, pressure, precipitation, and weather codes are operationally tempting. The first decoder only
publishes evidence and must not imply route scoring, drone safety recommendations, or operator decision authority.

Mitigation:

- Keep route/weather decision state separate from decoded telemetry.
- Require freshness, confidence, query-shape, source/model time, and operator-visible caveats before routing or safety
  rules consume weather.

### Medium: Visual weather is a different layer

Browser-side radar, tiles, WMS, or raster overlays do not need backend graph ingestion unless an operator workflow
turns them into evidence.

Mitigation:

- Keep visual tile source review separate from tactical weather telemetry.
- Track license, cache, and reliability before any visual provider becomes a default COP layer.

## Verification

- `go test ./internal/components/weather`
- `go test ./internal/contracts`
- `go test ./...`

## Follow-Ups

- OGC EDR-shaped point, area, trajectory, and corridor fixtures were added as parser/preflight evidence.
- Decide whether live Open-Meteo/EDR polling belongs in a SemStreams HTTP poller component or a SemOps weather gateway.
- A point-forecast graph projector component was added after the ownership, indexing profile, freshness, and
  route-safety posture review in `2026-06-23-weather-projector-component-review.md`.
