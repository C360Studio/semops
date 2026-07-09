# SemLink Target Derivation Review

Scope: hosted SemLink ArduPilot readback requests that omit an explicit command target asset.

Decision: accept deterministic MAVLink target derivation before guarded command admission.

SemLink readback callers may omit `target_asset_id` for the MVP `AUTOPILOT_VERSION` path. The ingress derives the
canonical MAVLink source asset from `vehicle_system_id` plus the configured MAVLink org/platform, then passes that
target through the existing guarded projector. The graph target resolver still has to prove the derived asset is born
before command-intent persistence.

Boundary choices:

- Derivation removes a brittle caller requirement; it does not bypass born-first target admission.
- The route still accepts only SemLink companion readback intent and still exposes no native or companion transmit.
- This is source/target normalization for GCS-glass readback, not live SemLink route selection or mesh causality.

Evidence:

- `internal/projectors/mavlink/projector.go` exposes the canonical MAVLink source asset ID helper.
- `internal/ingress/semlink/readback.go` derives a missing target before command intent creation.
- `internal/ingress/semlink/readback_test.go` proves derived targets are written into command-intent triples.
- `cmd/semops/main_test.go` proves hosted composition queries the graph for the derived target and returns it.

Residual risk:

- The route still does not discover or contact a live SemLink node. BlueOS/Navigator route inventory remains a future
  slice, and graph birth remains the safety gate for target availability.
