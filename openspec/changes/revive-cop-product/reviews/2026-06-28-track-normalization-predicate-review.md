# Track Normalization Predicate Review

Scope: first SemOps adoption of the SemStreams spatial-temporal normalization path from task `9.4`, limited to
source-owned track current-state projections.

## Verdict

Accept the narrow track normalization slice.

MAVLink, TAK/CoT, ADS-B, and SAPIENT track contracts now claim canonical numeric spatial predicates
`geo.location.latitude` / `geo.location.longitude` and canonical event-time predicate
`time.observation.recorded`. Their projectors emit those triples beside existing COP-facing WKT and observed-time
predicates, so product rendering and inspector fields remain stable while SemStreams spatial/temporal indexes get the
shared vocabulary they expect.

## Evidence

- `pkg/cop/contracts.go` adds the canonical predicates to source-owned track contracts only.
- `internal/projectors/mavlink`, `cot`, `adsb`, and `sapient` emit event time for track mutations.
- The same projectors emit numeric lat/lon only when the source supplied a valid position.
- SemStreams `v1.0.0-beta.119` keys the temporal index on `time.observation.recorded` first, with `UpdatedAt`
  fallback.
- Projector tests cover valid-position output and missing-position guards.
- `go test ./...` passed after the slice.

## Residual Risk

- KLV sensor footprints and weather query geometry are covered by the follow-on
  `2026-06-28-signal-geometry-normalization-review.md`; tasks and advisories still use product WKT predicates only.
  Normalize them separately after their query pressure is concrete.
- Server-side `graph.query.spatialTemporal` remains deferred until the existing prefix, spatial, temporal, and batch
  hydration composition proves insufficient.
