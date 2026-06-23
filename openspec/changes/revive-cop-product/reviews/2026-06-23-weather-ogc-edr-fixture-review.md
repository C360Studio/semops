# Weather OGC EDR Fixture Review

Date: 2026-06-23

Scope: synthetic OGC API Environmental Data Retrieval-shaped point CoverageJSON fixture and decoder path for tactical
weather preflight evidence.

## Decision

Accept the OGC EDR-shaped point fixture as standards-facing parser/storage/governance evidence only.

The fixture is useful because it forces SemOps weather decoding to understand the EDR point-query shape, WKT `coords`,
CoverageJSON-style domain/range separation, units, and the same tactical variables carried by the Open-Meteo fixture.
It also proves that the weather decoder component can route a second provider shape through SemStreams registered
BaseMessage payloads and stream ports. It does not prove OGC conformance, live EDR provider behavior, broader query
shape support, route-safety authority, or weather-gateway readiness.

## Adversarial Notes

### High: EDR-shaped is not EDR conformant

`fixtures/weather/ogc-edr-position.json` is synthetic and local. It is not an OGC ETS result, a certified sample, or a
captured response from an operational EDR server.

Mitigation:

- Label the fixture as synthetic in docs and code.
- Keep official OGC validator/ETS, live server captures, and bridge tests as separate gates.
- Do not use this fixture as proposal language for OGC EDR conformance.

### High: Point is not the full weather problem

The first parser handles point-shaped CoverageJSON only. EDR also covers area, trajectory, corridor, radius, cube,
items, locations, and instances query resources, and weather product decisions will likely need at least area and
trajectory/corridor behavior.

Mitigation:

- Keep non-point query shapes out of accepted support language.
- Add fixtures for area and trajectory/corridor before route planning, incident-area overlays, or standards-facing
interop claims exceed point retrieval.

### Medium: CoverageJSON subset can hide parser assumptions

The first decoder accepts the shape SemOps generated: one point, one time axis, and one-dimensional ranges. Real
providers may use different parameter metadata, CRS defaults, missing values, multi-dimensional ranges, or alternate
encodings.

Mitigation:

- Add captured provider examples before live integration.
- Treat missing/null values, CRS selection, parameter naming, and multi-dimensional ranges as explicit parser tests
  instead of assuming the synthetic fixture covers them.

### Medium: Weather telemetry is not a decision rule

Wind, gusts, visibility, pressure, precipitation, and weather codes are operational inputs. Publishing decoded values
must not imply that SemOps can recommend drone routes or safety actions yet.

Mitigation:

- Keep route/weather decisions as separate `control` state.
- Require freshness, model time, confidence, source, query shape, and operator-visible caveats before weather affects
  routing or safety logic.

## Verification

- `go test ./pkg/adapters/weather`
- `go test ./internal/components/weather`
- `go test ./internal/contracts`
- `go test ./...`

## Follow-Ups

- Add OGC EDR area and trajectory/corridor fixtures.
- Capture at least one legal live EDR server response before live-provider support language.
- Decide whether weather provider polling remains a component chain or promotes into a SemOps weather gateway.
