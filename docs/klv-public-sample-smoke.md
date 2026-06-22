# KLV Public Sample Smoke

Status: opt-in developer smoke, not CI and not conformance.

SemOps can exercise the KLV demux and MISB ST 0601 decoder against a local public MPEG-TS KLV sample without
vendoring or downloading media in the test. The smoke is skipped unless a developer supplies a local file path and
explicit provenance notes.

## Candidate Sources

- FFmpeg sample archive: <https://samples.ffmpeg.org/MPEG2/mpegts-klv/> lists `Day Flight.mpg` and
  `Night Flight IR.mpg`. The directory listing is useful provenance, but it does not include redistribution or
  license terms, so SemOps MUST NOT vendor either file or treat it as a cleared fixture.
- `klvdata`: <https://github.com/paretech/klvdata> documents a workflow using FFmpeg to extract KLV from
  `Day Flight.mpg`, and its repository includes a small binary KLV packet fixture under the project license. That
  repository license does not clear the FFmpeg-hosted MPEG-TS sample for redistribution.
- FFmpeg legal notes: <https://ffmpeg.org/legal.html> apply to FFmpeg itself. SemOps invokes an installed FFmpeg
  binary for optional smoke tests and does not vendor FFmpeg or media samples.

## Run

Place the sample outside version control, for example:

```bash
mkdir -p fixtures/klv/public-samples
# Put a locally obtained MPEG-TS KLV sample here, for example Day Flight.mpg.
```

Then run:

```bash
SEMOPS_KLV_PUBLIC_SAMPLE_PATH="fixtures/klv/public-samples/Day Flight.mpg" \
SEMOPS_KLV_PUBLIC_SAMPLE_SOURCE_URL="https://samples.ffmpeg.org/MPEG2/mpegts-klv/Day%20Flight.mpg" \
SEMOPS_KLV_PUBLIC_SAMPLE_PROVENANCE="candidate smoke sample; not vendored; license not cleared for redistribution" \
go test ./internal/components/klv -run TestPublicKLVSampleSmokeWithLocalPath -count=1
```

Optional bounds:

```bash
SEMOPS_KLV_PUBLIC_SAMPLE_MAX_EXTRACT_BYTES=8388608
SEMOPS_KLV_PUBLIC_SAMPLE_MAX_PACKET_BYTES=65536
SEMOPS_KLV_PUBLIC_SAMPLE_MAX_PACKETS=4096
```

## Acceptance

The smoke passes only if:

- The local sample has a data stream discoverable by `ffprobe`.
- FFmpeg extracts bounded KLV bytes from the selected data stream.
- The SemOps demux component publishes registered `semops.klv_packet.v1` messages.
- The SemOps decoder component publishes at least one registered `semops.klv_misb0601_frame.v1` message with
  supported sensor or frame-center geometry.

Passing this smoke means "real-world demux/parser path exercised once." It does not mean deterministic correctness,
official STANAG 4609 conformance, public-sample redistribution clearance, live media support, or streaming-binary
product support.
