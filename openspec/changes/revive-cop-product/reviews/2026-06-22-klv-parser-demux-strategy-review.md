# KLV Parser And Demux Strategy Review

Date: 2026-06-22

## Decision

Use a Go-native deterministic MISB ST 0601 local-set decoder for the first parser spike, and keep MPEG-TS demux
behind the SemStreams demux component boundary.

This is not the final production media strategy. It is the smallest honest path that proves SemOps can turn bounded
KLV packet bytes into typed decoded-frame payloads without graph writes, public sample licensing, network downloads,
or a new Python/JVM/Rust sidecar in the critical path.

## Rationale

- The immediate acceptance question is semantic and deterministic: do known packet bytes decode into the expected
  frame time, platform designation, sensor position, frame center, azimuth, and elevation?
- Public video-plus-KLV samples are useful for smoke testing demux behavior, but they are weak deterministic evidence
  unless we also have authoritative truth sidecars.
- `klvdata`, jMISB, FFmpeg/GStreamer, and Rust workers remain credible candidates, but introducing them before the
  SemStreams payload and component boundary is proven would make the first failure mode harder to interpret.
- Keeping demux as a separate component preserves the option to swap in FFmpeg/GStreamer, jMISB, or a Rust worker
  later without changing the downstream decoded-frame payload or projector contract.

## Current Evidence

- `internal/components/klv/decoder.go` decodes the first supported MISB ST 0601 local-set subset from bounded packet
  bytes.
- `internal/components/klv/decoder_test.go` builds deterministic packet bytes locally and asserts decoded values
  without graph writes.
- `internal/components/klv/components_test.go` proves the opt-in decoder worker consumes a registered packet
  BaseMessage and publishes a registered decoded-frame BaseMessage to the declared frame subject without graph
  mutation publication.
- `internal/components/klv/demux_test.go` proves the fixture-grade FFmpeg/ffprobe demux worker consumes a registered
  media-ref BaseMessage, selects an explicit data stream, bounds extracted bytes, and publishes a registered packet
  BaseMessage without graph mutation publication.
- `go test ./internal/components/klv` passes for the parser core, payload registry tests, and component skeleton.

## Risks

- The decoder covers only a deliberately small subset and must not be described as full MISB ST 0601 or STANAG 4609
  conformance.
- The current path does not demux MPEG-TS and does not validate the checksum tag.
- Storage-reference-only packet decoding is intentionally blocked until a bounded materialization path exists.
- Production performance and compatibility may still justify a sidecar parser after public-sample smoke or throughput
  tests.

## Follow-Up Tasks

- Wire the deterministic decoder core into an opt-in SemStreams worker path.
- Choose the MPEG-TS demux implementation only after public-sample license/provenance review.
- Add checksum validation or explicit checksum-policy evidence before any stronger parser-support language.
- Run adversarial review before MISB/STANAG engineering-support language appears in demo copy.
