# SAPIENT Preflight Component Review

Date: 2026-06-21

## Decision

Accept a preflight-only SAPIENT SemStreams component package, but keep SAPIENT graph projection, owner claims, product
service hosting, and conformance claims blocked.

`internal/components/sapient` is acceptable because it does not invent SAPIENT graph semantics. It provides a raw HTTP
input component and decoder processor for configured SAPIENT/Apex-style HTTP sources, emits registered raw and decoded
`message.BaseMessage` payloads on stream ports, captures replay records, reports stale-source health, and exposes
flow metrics. The decoder uses the existing `pkg/adapters/sapient` JSON and descriptor-based protobuf preflight
boundary.

This is not a SemOps-hosted SAPIENT server, not a Dstl harness result, and not permission to add `OwnerSAPIENT`.

## Objections Raised

- A SAPIENT component package could be read as product support. The package must stay preflight-only until the Dstl
  harness scope, service mode, and projection ownership are reviewed.
- HTTP polling is not the whole SAPIENT story. Apex/middleware HTTP access is a useful service-shape reference, but
  native SAPIENT sessions, tasking, and middleware interop may need different input components.
- Protobuf descriptor decode is developer preflight evidence, not generated-binding parity or full-message coverage.
- Detections are tempting to graph immediately, but range/bearing detections need sensor pose, reference frame, and
  uncertainty before becoming global COP state.
- Associated detections and cross-source correlation are fusion/evidence responsibilities, not source-owner facts.

## Evidence Checked

- `pkg/adapters/sapient` already validates representative JSON fixtures and descriptor-based binary protobuf
  messages before graph writes.
- `pkg/adapters/sapient` raw lane and replay records retain JSON/protobuf bytes with source, encoding, content, node
  ID, and message time when parse succeeds.
- `internal/components/sapient` exposes `HTTPClientPort`, `TimerPort`, raw and decoded stream ports, config schema,
  health, flow metrics, payload registry entries, replay capture, and malformed-message capture before parse failure.
- `internal/contracts` asserts SAPIENT component shape and explicitly keeps the decoder output as a NATS stream rather
  than a graph request port.

## Follow-Up Tasks

- Run or qualify the Dstl BSI Flex 335 v2 Test Harness before any compliance language.
- Decide whether generated Go bindings are needed beyond dynamic descriptors.
- Review SAPIENT source identity, first graph entity model, indexing profile, and ownership before adding
  `OwnerSAPIENT`.
- Decide whether product service mode starts with Apex/middleware HTTP, native session behavior, or a SemOps-owned
  SAPIENT-facing service.
