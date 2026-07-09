# Weather EDR Spatial Fixtures Review

Date: 2026-06-23

Scope: synthetic OGC EDR-shaped area, trajectory, and corridor parser fixtures for tactical weather preflight.

## Decision

Accept the area, trajectory, and corridor fixtures as parser-only spatial query-shape evidence.

The slice is useful because it proves that SemOps can distinguish EDR point, area, trajectory, and corridor inputs,
validate simple WKT `POLYGON` and `LINESTRING` geometry, require corridor width/height metadata, and preserve
time-series tactical weather variables without graph writes. It intentionally does not promote these shapes into the
SemStreams weather point payload, graph projection, routing logic, or UI.

## Adversarial Notes

### High: Spatial parsing is not route-weather support

The fixtures prove that SemOps can parse simple spatial query shapes. They do not prove route scoring, drone safety,
weather avoidance, or decision authority.

Mitigation:

- Keep route-weather decisions as separate `control` state.
- Require model time, freshness, confidence, source, query shape, and operator-visible caveats before weather affects
  routing or safety logic.

### High: Synthetic CoverageJSON is not provider dimensionality evidence

The spatial fixtures use one-dimensional time-series ranges. Real EDR providers may return grids, vertical levels,
multi-dimensional arrays, provider-specific parameter metadata, alternate CRS handling, and missing values.

Mitigation:

- Capture legal live EDR examples before live-provider support language.
- Add fixtures for multi-dimensional area and corridor responses before graph projection or UI claims.

### Medium: WKT support is deliberately narrow

The first parser validates simple `POLYGON`, `LINESTRING`, and `MULTILINESTRING` coordinates. It does not fully
support Z/M trajectory timing, LINESTRINGM epoch values, complex polygons with holes, or all corridor edge cases.

Mitigation:

- Keep Z/M and complex geometry handling as explicit follow-up parser tests.
- Do not use this slice as OGC conformance evidence.

### Medium: Runtime payload remains point-only

`internal/components/weather` still emits the point forecast payload. That is correct until SemOps has accepted a
spatial tactical-weather payload, graph model, and UI semantics.

Mitigation:

- Keep spatial parser tests in `pkg/adapters/weather` for now.
- Promote spatial payloads only after ownership, indexing, freshness, confidence, and operator presentation are
  reviewed.

## Verification

- `go test ./pkg/adapters/weather`
- `go test ./internal/components/weather`
- `go test ./internal/contracts`
- `go test ./...`

## Follow-Ups

- Add legal live EDR server captures for point and spatial queries.
- Add Z/M trajectory and multi-dimensional CoverageJSON fixtures.
- Design the governed tactical-weather graph projection before UI or route-safety work.
