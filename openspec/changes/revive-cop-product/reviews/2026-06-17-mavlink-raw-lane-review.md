# MAVLink Raw Lane Adversarial Review

Date: 2026-06-17
Scope: `COP-003` bounded raw frame lane in `pkg/adapters/mavlink`

## Finding Summary

- Severity: Medium. The raw lane is bounded and test-backed, but it is still in-memory. Durable replay fixtures and
  stack-level retention policy are not implemented.
- Severity: Medium. Current-state projections now carry `cop.provenance.source_ref`, but the source reference points
  to the latest retained raw record only. Operators should not read it as a long-term audit archive.
- Severity: Medium. This proves MAVLink raw-frame boundedness for small frames, not KLV/video binary streaming.
  KLV remains a separate memory-bound proof spike.
- Severity: Low. The source-reference predicate is product-local. Do not upstream it to SemStreams until a second
  feed proves the same convention is reusable.

## Role Review

architect:

- Accepts the lane as a component, not a service. The service boundary belongs to the MAVLink adapter when UDP/TCP,
  SITL, placement, or scaling forces appear.
- Requires durable replay storage and active health polling before using the lane in the structural demo.

go-reviewer:

- Accepts tests for metadata capture, byte and record eviction, oversize rejection, replay lookup, and defensive
  copies.
- Requires live adapter tests to prove parse, raw capture, projection, and graph write ordering together.

technical-writer:

- Requires docs to call this bounded raw capture, not replay completion.
- Requires KLV/binary claims to stay blocked until SemSource or SemOps proves by-reference video metadata handling
  under bounded memory tests.

## Decision

Accept the bounded raw lane as a `COP-003` increment. Keep `COP-004` responsible for service wiring, durable replay
storage, graph health polling, and multi-feed smoke evidence.
