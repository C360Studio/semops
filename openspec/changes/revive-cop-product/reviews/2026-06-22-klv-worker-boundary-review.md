# KLV Worker Boundary Review

Date: 2026-06-22

## Decision

Design the first KLV/MISB implementation as a SemStreams component flow owned by SemOps:

1. Media-reference input component.
2. KLV demux processor.
3. MISB ST 0601 decode processor.
4. Governed graph projector processor.
5. Optional CoT, CS API JSON, or other interop processors.

This keeps KLV/MISB out of the COP server core and preserves the SemStreams flow contract: declared ports,
registered payloads, config schema, health, `DataFlow()`, Prometheus telemetry, and graph writes only through
declared request ports.

## Claims Policy

SemOps can use public examples commonly used by open-source FMV/KLV tooling plus deterministic truth fixtures as
demo-grade engineering-support evidence. That is good enough for the MVP and comparable-product positioning.

SemOps must not claim official STANAG 4609 conformance, certification, or lab validation until someone funds a
validator/lab effort with proper access and the resulting evidence is recorded.

## Boundary Details

- Input accepts SemSource storage references, local fixture paths, or future native media ingress references.
- Demux emits `semops.klv_packet.v1` trace payloads with packet offsets, timing, and bounded bytes or storage refs.
- Decode emits `semops.klv_misb0601_frame.v1` payloads for the first supported field subset.
- Projector writes derived COP facts through SemStreams graph request ports with a registry-minted
  `semops.feed.klv` owner token.
- Sensor geometry and platform/sensor current state are `signal`; clip/evidence lifecycle is `control`; packet,
  demux, and decode diagnostics are `trace`.
- Optional interop processors emit CoT or CS API JSON from the decoded model and do not replace internal governed
  graph projection.

## Objections Raised

- A sidecar-first design can feel heavier than linking a parser library directly. The benefit is isolating binary
  tooling, process memory, language choice, and restart behavior from the COP server.
- Carrying bounded raw KLV packet bytes on stream payloads may be acceptable for small fixtures, but large or live
  feeds should prefer packet storage references, JetStream durability, or SemStreams buffer/cache support after
  telemetry proves the need.
- Public samples are useful for smoke testing, but deterministic fixtures must carry acceptance because public samples
  rarely include authoritative truth sidecars.
- CoT and CS API outputs are tempting as the "real" model. They should remain bridge outputs so SemOps keeps one
  internal KLV/MISB-derived projection contract.

## Follow-Up Tasks

- Decide parser/demux strategy for the first spike: FFmpeg/GStreamer plus `klvdata`, jMISB sidecar, Rust worker, or
  Go-native parser.
- Add an opt-in local deterministic fixture spike without graph writes before projection work.
- Run adversarial review before adding KLV/MISB support language to demo or positioning copy.
