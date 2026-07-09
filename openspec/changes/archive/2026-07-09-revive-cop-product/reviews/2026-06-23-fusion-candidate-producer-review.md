# Fusion Candidate Producer Review

Date: 2026-06-23
Scope: OpenSpec task 8.22, bounded graph-discovered candidate production for fusion association.

## Decision

Accept the first hosted fusion candidate producer as plumbing for controlled association experiments. Do not enable
automatic demo association by default yet.

## Findings

- The producer is a SemStreams lifecycle component, not a hidden NATS hook: it exposes a timer input, graph-prefix
  request output, candidate-batch output, `Health()`, and `DataFlow()`.
- Candidate generation is bounded at source discovery, pair-comparison, and batch-count levels, which gives operators
  and Prometheus useful places to detect backpressure before scoring gets expensive.
- Source pairs are generated one-way from the configured source order, avoiding same-scan primary/candidate reversal
  for the same association entity.
- Candidate payloads preserve source track IDs, native IDs, positions, observed times, confidence, and source refs for
  downstream scoring and provenance.

## Boundaries

- The producer does not write graph state, mutate source tracks, merge identities, or own association evidence.
- The producer is disabled by default via `SEMOPS_FUSION_CANDIDATES_ENABLED=false`.
- This is not yet a full-stack demo proof. The next gate must seed source tracks, run producer plus projector, and
  verify COP association readback through the stack.
- Identity policy, operator merge/split controls, and automatic demo association remain blocked behind e2e proof and
  adversarial operator review.

## Verification

- `go test ./internal/components/fusion`
- `go test ./internal/app`
