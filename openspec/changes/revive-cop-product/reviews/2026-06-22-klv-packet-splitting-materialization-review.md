# KLV Packet Splitting And Materialization Review

Date: 2026-06-22
Scope: KLV demux worker packet splitting and storage-reference materialization boundary.

## Decision

The KLV demux worker may split concatenated MISB ST 0601 local sets extracted from a selected FFmpeg data stream and
publish one registered `semops.klv_packet.v1` BaseMessage per packet. Storage-reference-only media refs may enter the
same path only through an explicit `MediaMaterializer` that enforces `max_materialized_bytes` and cleanup.

This remains fixture-grade worker evidence. It is not a live-media, streaming-binary, public-sample, or official
STANAG 4609 conformance claim.

## Adversarial Findings

- Packet splitting is valuable because real extracted KLV data streams may contain more than one local set. Treating
  the entire extracted stream as one packet would hide offsets and make downstream trace evidence ambiguous.
- Splitting must remain conservative: every segment starts with the MISB ST 0601 universal key, every packet enforces
  `max_packet_bytes`, and every media ref enforces `max_extract_bytes` plus `max_packets`.
- Storage-reference support is only safe when it is dependency-injected. A generic materializer contract lets
  SemSource or a media sidecar stage bytes later without putting SemOps into the storage business or bypassing
  SemStreams component ports.
- Remote URI and live media remain intentionally out of scope. Adding network fetch directly to demux would collapse
  transport, storage, and parsing concerns into the processor.
- No graph writes are introduced. The demux worker publishes packet trace messages only; governed COP state still
  belongs in the KLV projector.

## Evidence

- `go test ./internal/components/klv`

## Follow-Ups

- Decide whether packet storage-reference-only decode needs a bounded materializer similar to media demux.
- Add public KLV sample provenance before any real-world media smoke.
- Keep hosted KLV runtime opt-in until live media ingress, storage policy, and operator-facing UI claims are reviewed.
