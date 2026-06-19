# KLV/STANAG 4609 Feed Evidence

Status: stretch proof spike.

## Decision

KLV/STANAG 4609 is the feed most likely to expose whether SemOps and SemStreams can honestly handle
binary-derived data. It should not be a product feed until a small fixture proves metadata extraction,
binary-by-reference storage, and memory-bounded handling.

## Local Evidence

- No SemOps KLV adapter exists in the current checkout.
- SemSource has video-source and video-handler code that extracts metadata and keyframes with ffprobe/ffmpeg.
- SemSource video handling streams hashing, but when a storage backend is configured it currently reads the full
  video file into memory before storage.
- SemSource can publish typed media entity state through SemStreams, but it is not a tested KLV parser.

## External Evidence

- jMISB implements multiple MISB standards. Its support table lists ST 0601 UAS Datalink as mostly implemented,
  ST 0805 KLV-to-CoT conversion as implemented, and ST 1402 MPEG-2 transport stream support as mostly implemented.
- jMISB documents a network-stream API that can receive video and metadata asynchronously.
- `klvdata` parses and constructs KLV formatted binary streams and targets MISB ST 0601 UAS metadata from
  STANAG 4609-compliant MPEG-TS streams.
- `klvdata` alone does not demux KLV from MPEG-TS; FFmpeg or GStreamer is needed in that workflow.

## Gates

### Fixture Gate

Target artifact:

- A tiny video-plus-KLV fixture that can be checked in or generated without restricted data.

Acceptance:

- Fixture licensing is clear.
- The fixture contains enough ST 0601 metadata to extract platform/sensor position and sensor footprint.
- The fixture is small enough for local tests.

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
go test ./internal/klv
```

Acceptance:

- ST 0601 metadata extracts deterministically from the fixture.
- Platform position, sensor position, frame time, and footprint evidence are represented in a typed model.
- Parser errors produce trace evidence without corrupting current COP state.

### Projection Gate

Target command after SemOps graph contracts exist:

```bash
go test ./internal/projectors/klv
```

Acceptance:

- Sensor footprints and extracted platform/sensor coordinates use `indexing_profile=signal`.
- Clip/evidence package lifecycle uses `indexing_profile=control`.
- Frame/keyframe descriptions or operator annotations use `indexing_profile=content` when present.
- Packet/frame decode events use `indexing_profile=trace`.
- Raw binary never goes into graph triples.

## Known Gaps

- No small legal fixture identified yet.
- No SemOps parser strategy chosen: Go-native, Java sidecar, Python sidecar, or SemSource extension.
- SemSource media path is promising but not proven for KLV or streaming binary.
- Current SemSource storage path needs a memory-bound review before large video claims.

## Adversarial Feed-Entry Questions

- Is the fixture legal, small, and representative enough?
- Are we claiming "streaming binary" before memory-bounded evidence exists?
- Is SemSource being used as a media sidecar rather than assumed to solve KLV?
- Does the graph contain metadata and references, never raw bytes?
- Is the parser sidecar choice justified by testability and deployment constraints?

## Source Links

- jMISB: <https://github.com/WestRidgeSystems/jmisb>
- klvdata: <https://github.com/paretech/klvdata>
