# Raw-Lane Plus Current-State Projection Guidance

## Summary

SemOps now has repeated downstream pressure for a SemStreams-level recipe around bounded raw lanes plus governed
current-state graph projection. This is not blocking SemOps, but it looks framework-shaped enough that sibling products
should not have to rediscover the same boundary.

Filed upstream as [SemStreams issue #340](https://github.com/C360Studio/semstreams/issues/340).

## Downstream Evidence

SemOps has the same pattern in multiple feeds:

- MAVLink: frame bytes stay on a bounded raw lane/replay store; source asset, track current state, and command ACK
  readback are projected into governed graph entities with source references.
- TAK/CoT: XML payloads stay on raw/replay paths; operator tracks, marker tasks, advisories, and GeoChat content are
  projected as current state or content entities.
- ADS-B: OpenSky-shaped snapshots stay on raw/replay paths; aircraft tracks are projected as current state.
- SAPIENT: JSON/protobuf payloads stay on bounded raw/replay paths; only reviewed absolute-location detections project
  into graph state.
- KLV/MISB: media and packet bytes stay by reference; bounded sensor/frame-center evidence projects into graph state.

The repeated questions are:

- Which bytes must stay off graph?
- Which raw/replay/storage reference should become graph-visible provenance?
- Which entity carries current state?
- Which `indexing_profile` should apply to current state versus trace/debug evidence?
- How should component ports, payload registry entries, buffer/backpressure posture, and Prometheus flow metrics be
  documented together?
- Where should replay fixtures and portable demo data sit relative to raw capture?

## Ask

Please consider adding a SemStreams docs/pattern guide, and optional helper contracts only if useful, for:

- bounded raw-lane handling for high-rate or binary-ish feed payloads;
- current-state projection from raw/replay evidence into governed graph entities;
- source-reference/provenance conventions without forcing product-specific predicate names;
- recommended `signal`, `control`, `content`, and `trace` indexing-profile posture;
- component-flow placement: input component, decoder/processor, projector, raw capture, replay writer;
- guidance on when to use existing utilities such as buffer/cache/natsclient and when product code should stay local;
- a small example that is not tied to SemOps vocabulary.

## Non-Goals

- Do not add a mandatory object store or raw-payload service.
- Do not upstream SemOps-specific COP predicates.
- Do not require raw packets to become graph entities.
- Do not solve product retention policy, privacy policy, or fixture licensing.

## Why SemStreams

The pattern touches SemStreams-owned surfaces: component lifecycle, ports, payload registry, graph ownership,
indexing profiles, utility packages, and component telemetry. SemOps can continue with local code, but a canonical
recipe would reduce drift across SemOps, SemSource, SemTeams, SemConnect, and future C360 feed consumers.

## SemOps References

- SemOps OpenSpec task: `revive-cop-product` task `9.5`
- SemOps review:
  `openspec/changes/revive-cop-product/reviews/2026-06-23-semstreams-raw-lane-current-state-ask-review.md`
