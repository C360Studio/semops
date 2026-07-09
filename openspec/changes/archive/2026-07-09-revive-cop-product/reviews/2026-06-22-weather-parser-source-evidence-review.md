# Weather Parser Source Evidence Review

Date: 2026-06-22
Scope: first tactical weather parser fixture and source posture.

## Decision

Accept an Open-Meteo-shaped point forecast fixture as the first executable tactical weather evidence gate.

`pkg/adapters/weather` parses `fixtures/weather/open-meteo-point.json` and preserves provider, query shape, point
geometry, elevation, units, sample time, temperature, precipitation, visibility, surface pressure, wind speed, gusts,
wind direction, and weather code without graph writes.

This is parser and source-shape evidence only. Weather is still split into browser visual context, CAP/public alerts,
and localized tactical telemetry.

## Adversarial Notes

### High: Open-Meteo is not the internal weather architecture

Open-Meteo is a convenient fixture and early JSON source, not the core SemOps weather abstraction. The standards-facing
path remains OGC API EDR for position, area, trajectory, and corridor query shapes.

Resolution: keep the parser package provider-shaped and defer component/projection contracts until the first tactical
weather use case is chosen.

### High: Weather tiles are not graph state

Radar, precipitation, cloud, or WMS/tile layers may be useful visual context, but they should not enter SemStreams
unless they become evidence for a decision, alert, route, or safety rule.

Resolution: keep visual-tile source license/cache/reliability review separate from tactical telemetry parsing.

### Medium: NWS alerts stay on the CAP lane

NWS public warnings already fit the CAP feed path. Adding weather parser support must not duplicate CAP warning
projection or claim authoritative hazard truth.

Resolution: preserve CAP append-evidence behavior and treat tactical weather observations/forecasts as a separate
future `signal` projection.

## Verification

- `go test ./pkg/adapters/weather`
- `go test ./...`

## Follow-Ups

- Add an OGC EDR-shaped fixture for position, area, trajectory, or corridor query shapes.
- Design a SemStreams weather component package only after the first tactical query/use case is accepted.
- Keep live provider credentials, cache/rate-limit behavior, stale policy, and visual tile licensing under separate
  reviews.
