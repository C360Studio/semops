# SAPIENT Raw Replay Review

Date: 2026-06-20

## Decision

Accept bounded raw replay for the SAPIENT planning lane.

`pkg/adapters/sapient` now captures raw JSON and protobuf payload bytes with source, encoding, receive time, content
kind, and native identity metadata when parsing succeeds. It persists replay records as JSON Lines and decodes replay
through the same JSON/protobuf preflight boundary.

This approves exact-byte replay evidence for representative SAPIENT preflight payloads. It does not approve a hosted
SAPIENT adapter, Dstl harness compliance, graph projection, generated Go bindings, product support, or command/tasking
authority.

## Objections Raised

- Replay must retain exact native bytes rather than normalized graph state, or it cannot help compare Dstl harness
  behavior, dynamic descriptor behavior, and future projection behavior.
- Parser-failing bytes are still valuable evidence. The raw lane therefore accepts known encodings even when parsed
  metadata is unavailable.
- JSON Lines persistence encodes `[]byte` payloads as base64 through Go JSON. That is acceptable for fixture replay,
  but future large SAPIENT payloads should move toward object storage or bounded artifact references.
- Raw replay is not graph projection. Raw payload bytes must stay off graph entities until source-reference,
  ownership, and indexing policy are reviewed.
- SAPIENT tasking and acknowledgements touch control-plane semantics. Replay does not grant command authority.

## Evidence Checked

- `pkg/adapters/sapient/raw_lane.go` bounded in-memory capture, source-token normalization, supported encoding checks,
  defensive copies, and record/byte eviction.
- `pkg/adapters/sapient/replay.go` JSON Lines append/load validation and replay decode through `RawMessageRecord.Message`.
- `pkg/adapters/sapient/raw_lane_test.go` metadata capture, malformed-byte retention, eviction, invalid input, and
  defensive-copy coverage.
- `pkg/adapters/sapient/replay_test.go` append/load coverage for JSON and protobuf payloads plus malformed replay
  rejection.
- `go test ./pkg/adapters/sapient -count=1` passes.

## Accepted Risks

- The raw replay format is adapter-local and not yet a cross-feed SemStreams framework contract.
- Payload storage is file-backed fixture evidence, not a production archive or chain-of-custody mechanism.
- The current protobuf payloads are generated from representative JSON fixtures rather than captured SAPIENT network
  traffic.

## Follow-Up Tasks

- Run or qualify the Dstl BSI Flex 335 v2 Test Harness before compliance language.
- Design hosted SAPIENT raw ingress, health counters, and replay storage before adding service wiring.
- Run a SAPIENT projection ownership/indexing review before graph writes.
- Revisit whether raw-lane/current-state projection guidance belongs upstream in SemStreams after MAVLink plus enough
  non-MAVLink feeds prove common pressure.
