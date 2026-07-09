# CAP Hazard Geometry Normalization Review

Scope: follow-on SemOps adoption of the SemStreams spatial-temporal normalization path for CAP hazard evidence.

## Verdict

Accept a representative-point normalization slice for CAP hazard evidence.

CAP hazard-evidence contracts now claim canonical numeric spatial predicates
`geo.location.latitude` / `geo.location.longitude` and canonical event-time predicate
`time.observation.recorded`. The CAP projector emits those aliases as append-only evidence beside the existing
`cop.hazard.evidence` document.

## Boundary Choices

- CAP remains `append-evidence` and `content` profile. It still does not own authoritative `cop.hazard.geometry`,
  `cop.hazard.severity`, or `cop.hazard.status`.
- The index location is a representative point for discovery, not a replacement for the polygon or circle. The
  projector prefers the first CAP polygon centroid, then falls back to the first CAP circle center.
- The full CAP polygon/circle evidence remains in `cop.hazard.evidence` for COP rendering and inspection.
- Event time uses CAP `info.effective` when present, then alert `sent`, through `time.observation.recorded`.
- This slice does not add rich polygon containment indexing, CAP conformance claims, hazard lifecycle ownership, or
  derived route/asset hazard decisions.

## Evidence

- `pkg/cop/contracts.go` adds the canonical predicates to the CAP hazard-evidence contract.
- `internal/projectors/cap` emits event time for all projected CAP hazard evidence.
- The same projector emits numeric lat/lon only when the CAP area has polygon or circle geometry.
- Focused tests passed for `./pkg/cop` and `./internal/projectors/cap`.

## Residual Risk

- Representative-point indexing can find nearby CAP hazards, but it cannot answer full polygon/circle intersection.
- Authoritative hazard state remains a later deterministic hazard/fusion lane.
