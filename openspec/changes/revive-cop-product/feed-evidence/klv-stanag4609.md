# KLV/STANAG 4609 Feed Evidence

Status: stretch proof spike.

## Decision

KLV/STANAG 4609 is the feed most likely to expose whether SemOps and SemStreams can honestly handle
binary-derived data. It should not be a product feed until a small fixture proves metadata extraction,
binary-by-reference storage, and memory-bounded handling.

## Local Evidence

- No production SemOps KLV adapter or live MPEG-TS demuxer exists in the current checkout.
- SemSource has video-source and video-handler code that extracts metadata and keyframes with ffprobe/ffmpeg.
- SemSource video handling streams hashing, but when a storage backend is configured it currently reads the full
  video file into memory before storage.
- SemSource can publish typed media entity state through SemStreams, but it is not a tested KLV parser.
- SemOps does not currently have a real legal KLV, STANAG 4609, or SKG binary fixture to provide to SemSource.
- `internal/components/klv` now declares the first registered SemStreams payload schemas for the future KLV worker:
  `semops.klv_media_ref.v1`, `semops.klv_packet.v1`, and `semops.klv_misb0601_frame.v1`.
- `internal/components/klv` now exposes the first SemStreams component skeleton for media-reference input, KLV demux,
  MISB ST 0601 decode, and governed projector stages with declared file, stream, and graph request ports.
- `internal/components/klv` now includes the first Go-native deterministic MISB ST 0601 local-set decoder for bounded
  packet bytes. The decoder component can consume registered packet BaseMessages and publish registered decoded-frame
  BaseMessages when configured with a SemStreams bus.
- `internal/components/klv` now includes a fixture-grade FFmpeg/ffprobe demux worker path. The demux component can
  consume registered media-ref BaseMessages, select an explicit data stream with ffprobe, extract bounded KLV bytes
  with FFmpeg `-map`, split concatenated MISB ST 0601 local sets, and publish one registered packet BaseMessage per
  split packet. Storage-reference-only media refs require an explicit bounded materializer. This is not a live media
  or production STANAG demux claim.

## SemSource Fixture Handoff

If SemSource needs an immediate fixture while migrating to governed SemStreams, SemOps should not block it on a real
KLV/SKG sample that does not exist yet. Use a deliberately synthetic binary fixture and label it as a storage and
governance proof, not protocol conformance.

The synthetic fixture may prove:

- Raw bytes are stored by reference rather than written into graph triples.
- Hashes, storage references, byte ranges, extraction notes, and provenance become governed metadata entities.
- SemSource uses SemStreams owner tokens, declared predicates, and indexing profiles for the metadata it publishes.
- The configured storage path is memory bounded for the fixture shape being tested.

The synthetic fixture must not be used to claim KLV/STANAG 4609, SAPIENT, SKG, streaming-binary, or parser
conformance. Promoting beyond storage/governance proof requires a legal representative fixture, parser strategy,
metadata extraction tests, and a separate adversarial review.

## SemSource Alignment

SemSource's governed SemStreams migration now treats opaque binary handling as a substrate proof. It can store,
hash, reference, and publish governed metadata for binary artifacts, but it explicitly leaves KLV, MISB ST 0601,
STANAG 4609, SAPIENT, SKG, parser, translation, and protocol-conformance claims to SemOps or a SemOps-owned worker.

SemOps accepts that boundary. The SemOps product path is:

1. Consume SemSource storage references or native media ingress.
2. Demux and parse KLV/MISB payloads in a SemOps-owned worker or sidecar.
3. Publish governed derived facts, CoT, CS API JSON, or COP-specific projections through SemStreams contracts.
4. Keep protocol and streaming-binary claims gated by fixture provenance, parser tests, memory evidence, and
   adversarial review.

Demux does not need to happen in SemSource for this plan to work well. For MVP, SemOps should keep the
`media_ref -> demux -> decode -> project` flow because MPEG-TS KLV handling, MISB ST 0601 decode, and
engineering-support claims are COP product concerns. A future SemSource or media-stack component may still provide
generic media-track extraction, byte-range materialization, or live media relay support, especially if FFmpeg,
GStreamer, or MediaMTX becomes shared infrastructure. That generic service should not own SemOps KLV/STANAG claims;
SemOps should consume its output through declared SemStreams ports and keep KLV semantics downstream.

Recommended SemOps fixture ladder:

1. Opaque synthetic binary fixture from SemSource for storage/governance proof only.
2. Public KLV sample smoke in SemOps, after license and provenance review. This proves real-world demux/parser
   plumbing, not deterministic correctness or conformance.
3. Deterministic MISB ST 0601 fixture in SemOps: truth JSON to generated KLV packet bytes to parsed output, with
   optional MPEG-TS wrapping as a later container-proof step. This is the first credible engineering-support gate
   because assertions can compare parsed output to known truth within MISB integer quantization tolerances.
4. Formal STANAG 4609 conformance as a separate validator or lab track.

Engineering support language may cite public examples commonly used by open-source FMV/KLV tooling plus deterministic
fixtures. It must not use "official conformance", "certification", or equivalent language until a funded validator or
lab effort with proper access exists.

## Current Parser Strategy

The first KLV/MISB spike should stay Go-native and deterministic:

- Decode a bounded MISB ST 0601 local-set packet generated in tests.
- Prove frame time, platform designation, sensor position, frame center, azimuth, and elevation extraction without
  graph writes.
- Publish decoded-frame BaseMessages only through the declared frame output port when a bus is provided.
- Keep MPEG-TS demux behind the demux component boundary.
- Use FFmpeg/ffprobe as the first fixture-grade demux implementation, with explicit data-stream selection and bounded
  output. Defer GStreamer, `klvdata`, jMISB, or a Rust worker until public-sample smoke or throughput requirements
  justify the sidecar/toolchain cost.
- Split concatenated MISB ST 0601 local sets after extraction so each packet has its own packet ref, byte offset, and
  trace payload.
- Accept storage-reference-only media refs only through an explicit bounded materializer with cleanup and
  `max_materialized_bytes` enforcement.
- Do not vendor or download public media samples until license, provenance, cache, and CI policy are recorded.

## Next Slice

Add deterministic truth-to-MPEG-TS fixture generation or a legally reviewed public sample smoke before projection code
or broader support language. The current deterministic packet fixture is parser-core evidence only.

The worker should:

- Accept either a SemSource storage reference or native media ingress reference as input.
- Demux KLV from MPEG-TS without requiring raw video bytes in graph triples.
- Parse the first MISB ST 0601 subset needed for sensor position, frame time, frame center, azimuth, elevation, and
  later footprint inputs.
- Emit governed derived facts through SemStreams contracts, with `signal` for projected sensor geometry, `trace` for
  packet/decode diagnostics, and `control` for clip/evidence package lifecycle.
- Publish CoT or CS API JSON only as derived interop outputs, not as the internal KLV model.
- Expose component telemetry, backpressure posture, and memory bounds before any streaming-binary language.

### Worker Boundary

The first product shape should be a SemStreams component flow, not a monolithic COP server feature:

1. Media-reference input component.
   - Consumes SemSource storage references, local fixture paths, or future native media ingress references.
   - Declares `FilePort`, `NATSPort`, `NATSRequestPort`, or `HTTPClientPort` resources that match the chosen source.
   - Emits registered `semops.klv_media_ref.v1` messages with source URI, content hash, fixture/provenance metadata,
     optional byte range, and storage reference.
2. KLV demux processor.
   - Reads the referenced media through bounded buffers or a sidecar process.
   - Emits registered `semops.klv_packet.v1` trace messages containing packet metadata, byte offsets, timing, and
     either bounded packet bytes or packet storage references.
   - Does not write graph mutations.
3. MISB decode processor.
   - Parses the first supported ST 0601 subset into registered `semops.klv_misb0601_frame.v1` messages.
   - Starts with platform position, sensor position, frame time, frame center, and fields needed to compute a
     footprint.
   - Emits decode diagnostics as `trace` rather than corrupting current COP state.
4. KLV projector processor.
   - Writes governed graph mutations only through declared SemStreams graph request ports.
   - Uses `semops.feed.klv` owner tokens minted by the SemStreams registry/bind path.
   - Projects sensor geometry and platform/sensor current state as `signal`, clip/evidence package lifecycle as
     `control`, and packet/decode diagnostics as `trace`.
5. Optional interop processors.
   - Emit CoT, CS API JSON, or other bridge outputs from the decoded internal model.
   - Do not replace the governed internal projection contract.

Initial enablement should be opt-in, for example `SEMOPS_KLV_ENABLED=false` by default. Public sample use should also
be opt-in or locally cached so CI does not depend on large network downloads.

## External Evidence

- jMISB implements multiple MISB standards. Its support table lists ST 0601 UAS Datalink as mostly implemented,
  ST 0805 KLV-to-CoT conversion as implemented, and ST 1402 MPEG-2 transport stream support as mostly implemented.
- jMISB documents a network-stream API that can receive video and metadata asynchronously.
- `klvdata` parses and constructs KLV formatted binary streams and targets MISB ST 0601 UAS metadata from
  STANAG 4609-compliant MPEG-TS streams.
- `klvdata` alone does not demux KLV from MPEG-TS; FFmpeg or GStreamer is needed in that workflow.
- `klvdata` includes a small binary packet sample and documents using FFmpeg to extract KLV data from the public
  FFmpeg `Day Flight.mpg` MPEG-TS sample.
- The FFmpeg sample archive currently lists `Day Flight.mpg` and `Night Flight IR.mpg` under `MPEG2/mpegts-klv/`.
- FFmpeg can manually map data streams; data streams are not selected automatically.
- `klv-uas` is a Rust crate for UAS KLV parsing, but its docs are sparse enough that it should be a candidate parser,
  not the default strategy until a spike proves maturity.

## Gates

### Payload Registry Gate

Target command:

```bash
go test ./internal/components/klv ./internal/contracts
```

Acceptance:

- `semops.klv_media_ref.v1` round-trips through SemStreams `message.BaseMessage`. [done]
- `semops.klv_packet.v1` round-trips through SemStreams `message.BaseMessage`. [done]
- `semops.klv_misb0601_frame.v1` round-trips through SemStreams `message.BaseMessage`. [done]
- Packet payloads require bounded packet bytes or a packet storage reference. [done]
- Decoded frame payloads require an explicit decoded-field inventory. [done]

### Component Skeleton Gate

Target command:

```bash
go test ./internal/components/klv ./internal/contracts
```

Acceptance:

- Media-reference input is a SemStreams input component with a `FilePort` and registered media-ref stream output.
  [done]
- Demux is a SemStreams processor component from media refs to KLV packet trace payloads. [done]
- Decode is a SemStreams processor component from KLV packets to MISB ST 0601 frame payloads. [done]
- Projector is a SemStreams processor component from decoded frames to graph create/update request ports. [done]
- Flowgraph connects media-ref -> demux -> decode -> projector through tappable stream ports. [done]

### Parser-Core Gate

Target command:

```bash
go test ./internal/components/klv
```

Acceptance:

- First spike uses a Go-native deterministic MISB ST 0601 local-set decoder rather than a sidecar. [done]
- Bounded packet bytes decode into `semops.klv_misb0601_frame.v1` fields without graph writes. [done]
- Frame time, platform designation, sensor position, sensor azimuth/elevation, and frame center decode from the
  fixture packet. [done]
- Storage-reference-only packet decode fails explicitly until a bounded packet materializer exists. [done]
- Unsupported tags are warning evidence, not current-state projection. [done]

### Decoder Worker Gate

Target command:

```bash
go test ./internal/components/klv
```

Acceptance:

- Decoder runtime is opt-in through an explicit SemStreams bus dependency. [done]
- A registered `semops.klv_packet.v1` BaseMessage is decoded through the payload registry. [done]
- The worker publishes a registered `semops.klv_misb0601_frame.v1` BaseMessage to the declared frame subject. [done]
- The decoder worker does not publish graph mutation subjects. [done]

### FFmpeg Demux Worker Gate

Target command:

```bash
go test ./internal/components/klv
```

Acceptance:

- Demux runtime is opt-in through an explicit SemStreams bus dependency. [done]
- A registered `semops.klv_media_ref.v1` BaseMessage is decoded through the payload registry. [done]
- ffprobe is invoked with explicit data-stream selection instead of assuming FFmpeg auto-selects data streams. [done]
- FFmpeg extracts the selected stream with explicit `-map 0:<stream-index>` and bounded stdout. [done]
- Extracted data streams are split into distinct bounded MISB ST 0601 packet payloads with packet refs and byte
  offsets. [done]
- Storage-reference-only media refs are accepted only through an explicit bounded materializer with
  `max_materialized_bytes` and cleanup. [done]
- Optional local FFmpeg fixture generation wraps the deterministic KLV packet in MPEG-TS, then demuxes and decodes it
  back to the truth fixture when `ffmpeg` and `ffprobe` are installed. [done]
- The worker publishes a registered `semops.klv_packet.v1` BaseMessage to the declared packet subject. [done]
- The demux worker does not publish graph mutation subjects. [done]
- Remote URI demux remains rejected until a separate network/media ingress boundary is chosen. [done]

### Fixture Gate

Target artifact:

- Public sample smoke: a small or locally cached video-plus-KLV sample with documented license and provenance.
- Deterministic fixture: truth JSON, generated KLV payloads, optional generated MPEG-TS wrapper, and expected decoded
  output.

Acceptance:

- Fixture licensing is clear.
- Public sample smoke extracts plausible ST 0601 metadata from a real MPEG-TS stream without claiming conformance.
- Deterministic fixture output matches the source truth field set and numeric values within MISB integer quantization
  tolerances for the supported fields. [done]
- Deterministic MPEG-TS wrapping proves generated KLV packet bytes survive a local media-container mux/demux cycle
  without network downloads when FFmpeg tooling is available. [done]
- The deterministic fixture contains enough ST 0601 metadata to extract sensor position and frame-center evidence;
  full footprint polygon extraction remains a projector/parser extension. [partial]
- The fixture is small enough for local tests. [done]
- CI does not depend on large network downloads or require FFmpeg to be installed.

### Media Gate

Target command in SemSource or a SemOps sidecar:

```bash
go test ./handler/video ./processor/video-source
```

Acceptance:

- Video metadata and keyframe extraction work on the small fixture.
- Binary storage is by reference.
- Memory use is bounded, or the full-file read path is explicitly bypassed for large/streaming cases.

### KLV Extraction Gate

Target command after choosing a parser strategy:

```bash
go test ./internal/components/klv
```

Acceptance:

- ST 0601 metadata extracts deterministically from the fixture.
- Sensor position, frame time, frame center, azimuth, elevation, and platform designation are represented in a typed
  model.
- Footprint polygon extraction remains a separate extension gate.
- Parser errors produce trace evidence without corrupting current COP state.

### Projection Gate

Target command after SemOps graph contracts exist:

```bash
go test ./internal/projectors/klv
```

Acceptance:

- KLV-owned source-partitioned `sensor_footprint` current state uses `indexing_profile=signal`. [done]
- Sensor position, frame center, azimuth, elevation, media reference, packet reference, and provenance source are
  projected through born-first graph plans with owner-token fencing. [done]
- Footprint polygon extraction remains a separate extension gate. [not done]
- Clip/evidence package lifecycle uses `indexing_profile=control`.
- Frame/keyframe descriptions or operator annotations use `indexing_profile=content` when present.
- Packet/frame decode events use `indexing_profile=trace`.
- Raw binary never goes into graph triples.

## Known Gaps

- No public small legal KLV MPEG-TS sample has passed SemOps license/provenance review yet.
- No production MPEG-TS/live-media demux strategy chosen beyond the fixture-grade FFmpeg/ffprobe worker path.
- No public KLV sample has passed SemOps license/provenance review yet.
- No committed MPEG-TS binary is vendored; deterministic MPEG-TS wrapping is generated in local tests from the truth
  fixture when FFmpeg tooling is present.
- Media-reference input remains a topology skeleton; the projector now has contract-tested plan writing for
  sensor/frame-center state but is not wired into the hosted runtime.
- Demux and decoder workers exist for local file URI fixtures, bounded storage-ref materialization, split packet
  payloads, and bounded packet bytes, but no live media, public sample, or graph projection runtime exists yet.
- SemSource media path is promising but not proven for KLV or streaming binary.
- Current SemSource storage path needs a memory-bound review before large video claims.

## Adversarial Feed-Entry Questions

- Is the fixture legal, small, and representative enough?
- Are public samples being used only as smoke tests while deterministic truth fixtures carry acceptance?
- Are we claiming "streaming binary" before memory-bounded evidence exists?
- Is SemSource being used as a media sidecar rather than assumed to solve KLV?
- Does the graph contain metadata and references, never raw bytes?
- Is the parser sidecar choice justified by testability and deployment constraints?
- Are "engineering support" and "formal certification" kept separate in docs and demos?

## Source Links

- jMISB: <https://github.com/WestRidgeSystems/jmisb>
- klvdata: <https://github.com/paretech/klvdata>
- FFmpeg MPEG-TS KLV samples: <https://samples.ffmpeg.org/MPEG2/mpegts-klv/>
- FFmpeg docs: <https://ffmpeg.org/ffmpeg.html>
- klv-uas docs: <https://docs.rs/klv-uas/latest/klv_uas/>
