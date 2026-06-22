# KLV FFmpeg Demux Review

Date: 2026-06-22

## Decision

Use FFmpeg/ffprobe as the first fixture-grade MPEG-TS KLV demux implementation behind the SemStreams demux component
boundary.

This is not a production live-media strategy and not a STANAG conformance claim. It is the fastest honest way to prove
that a media reference can become a bounded `semops.klv_packet.v1` payload without putting raw video bytes into graph
state.

## Guardrails

- Discover data streams with ffprobe instead of assuming FFmpeg auto-selects data streams.
- Extract with explicit `-map 0:<stream-index>`.
- Bound stdout by `max_packet_bytes`.
- Accept local file URI fixtures only until storage-reference materialization is designed.
- Publish packet BaseMessages only on the declared packet subject.
- Keep graph writes in the later projector component.

## Objections Raised

- Shelling out to FFmpeg is less elegant than linking a parser. Accepted for the spike because FFmpeg is mature,
  already familiar in SemSource media work, and keeps sidecar/language choices out of the decoder contract.
- FFmpeg may extract a data stream that contains concatenated KLV local sets rather than one semantic frame. Accepted
  for the first fixture-grade packet path; deterministic truth-to-MPEG-TS fixtures must tighten packet/frame
  boundaries before stronger support language.
- Local file URI only is too narrow for production. Accepted because storage-reference materialization and live media
  relay should be separate service-promotion decisions.

## Follow-Up Tasks

- Add deterministic truth-to-KLV-to-MPEG-TS fixture generation or legally reviewed public sample smoke.
- Decide packet splitting policy when FFmpeg emits multiple local sets.
- Add checksum validation or explicit checksum-policy evidence before support language.
- Review whether FFmpeg belongs in the SemOps container image, a media sidecar, or SemSource-owned generic media
  infrastructure after fixture smoke.
