# TAK Control Content Normalization Review

Scope: follow-on SemOps adoption of the SemStreams spatial-temporal normalization path for TAK/CoT task marker and
GeoChat advisory point evidence.

## Verdict

Accept the narrow TAK control/content normalization slice.

TAK task marker and GeoChat advisory contracts now claim canonical numeric spatial predicates
`geo.location.latitude` / `geo.location.longitude` and canonical event-time predicate
`time.observation.recorded`. The existing COP-facing WKT predicates remain unchanged for map rendering and
inspection.

## Boundary Choices

- TAK marker tasks remain `control` profile read-model evidence for operator map state, not native task execution.
- GeoChat advisories remain `content` profile text evidence, with point aliases only when the source CoT event has a
  point.
- Event time uses the CoT event timestamp through `time.observation.recorded`; SemStreams can fall back to framework
  `UpdatedAt` when older records do not carry the predicate.
- This slice does not add command ingress, native transmitter behavior, ACK reconciliation, task target edges, CAP
  hazard geometry, or rich shape indexing.

## Evidence

- `pkg/cop/contracts.go` adds the canonical predicates to TAK task and advisory contracts.
- `internal/projectors/cot` emits event time for TAK marker tasks and GeoChat advisories.
- The same projector emits numeric lat/lon only when the source CoT event carries a point.
- Focused tests passed for `./pkg/cop` and `./internal/projectors/cot`.

## Residual Risk

- CAP hazard representative points are covered by
  `2026-06-28-cap-hazard-geometry-normalization-review.md`; rich CAP/weather/KLV shapes still use product geometry
  only.
- Native command/control remains gated by the command-intent, authority, and feed ACK/readback lanes.
