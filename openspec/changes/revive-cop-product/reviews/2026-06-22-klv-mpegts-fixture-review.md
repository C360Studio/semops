# KLV MPEG-TS Fixture Review

Date: 2026-06-22

## Decision

Accept an optional local FFmpeg MPEG-TS fixture smoke as the next KLV binary/container gate. The test generates a tiny
MPEG-TS file at runtime from the deterministic MISB ST 0601 truth packet, demuxes the data stream through the SemOps
KLV demux component using real `ffmpeg`/`ffprobe`, and decodes the result back to the truth fixture.

This proves the local container seam without vendoring binary media or making FFmpeg a hard CI dependency.

## Guardrails

- Skip the smoke when `ffmpeg` or `ffprobe` is not installed.
- Generate temporary KLV and MPEG-TS files during the test; do not commit large binary media.
- Keep the smoke to a single KLV local set. Multi-frame packet splitting is still open.
- Keep public-sample smoke and formal conformance separate.
- Keep graph writes out of this gate.

## Evidence

- `internal/components/klv/mpegts_test.go` writes the deterministic KLV packet to a temp file, muxes it with a tiny
  generated test video into MPEG-TS, demuxes the data stream through `DemuxComponent`, and decodes it with
  `DecodeMISB0601Packet`.
- `go test ./internal/components/klv` passes locally with FFmpeg installed.

## Follow-Up

- Decide packet splitting when FFmpeg extracts multiple concatenated KLV local sets.
- Add a legally reviewed public MPEG-TS KLV sample smoke before real-world media credibility language.
- Decide whether FFmpeg lives in the SemOps container image, a media sidecar, or SemSource-owned generic media
  substrate.
