# Mixed Feed Spatial Temporal Discovery Review

Scope: local SemOps proof that normalized `geo.location.*` plus `time.observation.recorded` predicates compose across
the current COP entity families.

## Verdict

Accept a local mixed-feed discovery smoke as the next geo-normalization proof.

The smoke builds representative graph entity states for MAVLink tracks, TAK tracks, TAK marker tasks, TAK GeoChat
advisories, CAP hazard evidence, ADS-B tracks, SAPIENT detections, KLV sensor footprints, and tactical weather
observations. It then filters them by canonical latitude, longitude, and event time.

## Boundary Choices

- This is a contract/local-readback smoke, not a live SemStreams server-side spatial query claim.
- The smoke requires `time.observation.recorded`; it does not let framework `UpdatedAt` fallback mask missing SemOps
  normalization.
- CAP hazard evidence contributes only a representative point. Rich polygon/circle intersection remains out of scope.
- The deferred SemStreams indexing/cardinality helper ask remains deferred until a real mixed-feed workflow proves
  clean entity boundaries and existing query composition are insufficient.

## Evidence

- `internal/contracts/semstreams_contract_test.go` adds
  `TestMixedFeedSpatialTemporalDiscoveryUsesCanonicalPredicates`.
- The smoke verifies in-window discovery across `signal`, `control`, and `content` profile entity families.
- The smoke excludes entities missing event time, missing location aliases, outside the spatial window, or outside the
  temporal window.
- Focused test passed for `./internal/contracts`.

## Residual Risk

- This does not prove deployed SemStreams `spatialSearch` or `temporalSearch` behavior.
- This does not prove rich geometry semantics for CAP, weather corridors/areas, or KLV polygons.
