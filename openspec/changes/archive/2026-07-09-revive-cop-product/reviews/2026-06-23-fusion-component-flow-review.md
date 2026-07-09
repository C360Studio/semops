# Fusion Component Flow Review

Date: 2026-06-23

Scope: hosted fusion association processor component, payload registry contract, graph writer, runtime opt-in wiring,
and component telemetry

## Decision

Accept the first hosted fusion association flow. SemOps can now host statistical association as a SemStreams processor
component that consumes bounded candidate batches and writes fusion-owned association evidence through born-first graph
mutations.

## Findings

1. The component follows SemStreams lifecycle and port shape.
   `internal/components/fusion.ProjectorComponent` exposes a NATS input port for
   `semops.fusion.track_candidates`, graph mutation request output ports, payload registry decoding, `Health()`, and
   `DataFlow()`. It does not subscribe to hidden subjects or bypass component contracts.

2. Graph writes stay governed.
   The component scores candidate batches with `internal/fusion/association`, projects evidence through
   `internal/projectors/fusion`, and writes through `graph.mutation.entity.create_with_triples` /
   `graph.mutation.entity.update_with_triples`. Entity-exists responses reconcile born state and reproject updates.

3. Runtime hosting is opt-in.
   `SEMOPS_FUSION_ENABLED=true` starts only the fusion projector component and subscribes to the configured candidate
   subject. The fusion owner token comes from the existing COP ownership registration path; the runtime does not add a
   second ownership claim.

4. Runtime telemetry is visible.
   The hosted component appears in `ComponentMetricSources()` as `fusion/projector`, so the existing Prometheus and
   `/api/cop/runtime` rollup path can report health, throughput, errors, and last activity.

## Boundaries

- This does not create candidate batches from graph discovery yet.
- This does not run automatic association in the demo stack by default.
- This does not merge tracks, mutate source entities, arbitrate identity, or expose operator merge/split controls.
- This does not tune association thresholds per mission or prove large-scale cardinality/backpressure behavior.

## Verification

- `go test ./internal/app ./internal/components/fusion ./internal/projectors/fusion ./internal/fusion/association ./pkg/cop`

## Follow-Up

Add a bounded candidate producer from graph-discovered source tracks, then run an adversarial identity-policy review
before enabling automatic association in the demo stack.
