# KLV Public Sample Smoke Review

Date: 2026-06-22
Scope: Public MPEG-TS KLV sample provenance and opt-in smoke gate.

## Decision

SemOps may use public KLV media samples only as local opt-in smoke evidence. The first smoke gate is
`TestPublicKLVSampleSmokeWithLocalPath`, skipped unless a developer supplies a local file path plus source URL and
provenance notes. The test does not download, cache, or vendor public media.

## Evidence Sources

- FFmpeg sample archive lists `Day Flight.mpg` and `Night Flight IR.mpg` under `MPEG2/mpegts-klv`.
- `klvdata` documents a workflow using FFmpeg to extract KLV from `Day Flight.mpg`.
- Neither fact clears the MPEG-TS media for redistribution inside SemOps.

## Adversarial Findings

- Public samples are useful for real-world demux/parser smoke, but they are weak correctness evidence because the
  exact truth values are not SemOps-owned.
- The FFmpeg sample archive listing is provenance, not a license grant. SemOps must not vendor those files unless a
  separate redistribution review clears them.
- Requiring `SEMOPS_KLV_PUBLIC_SAMPLE_SOURCE_URL` and `SEMOPS_KLV_PUBLIC_SAMPLE_PROVENANCE` prevents a local path from
  silently becoming an undocumented fixture.
- The smoke must stay outside default CI because the likely candidate files are large and the test depends on local
  FFmpeg/ffprobe tooling.

## Evidence

- `go test ./internal/components/klv` skips the opt-in public smoke when `SEMOPS_KLV_PUBLIC_SAMPLE_PATH` is unset.
- Downloaded `Day Flight.mpg` locally to ignored `fixtures/klv/public-samples/day-flight.mpg`.
- Size: `102004664` bytes.
- SHA-256: `a491ceff524b0008e3076d9eb30782badac2d53053731accc0a4e1226177260e`.
- Tools: FFmpeg `8.1.1`, ffprobe `8.1.1`.
- Passing smoke command:

```bash
SEMOPS_KLV_PUBLIC_SAMPLE_PATH=fixtures/klv/public-samples/day-flight.mpg \
SEMOPS_KLV_PUBLIC_SAMPLE_SOURCE_URL=https://samples.ffmpeg.org/MPEG2/mpegts-klv/Day%20Flight.mpg \
SEMOPS_KLV_PUBLIC_SAMPLE_PROVENANCE=local-download-from-ffmpeg-sample-archive-2026-06-22-sha256-a491ceff524b0008e3076d9eb30782badac2d53053731accc0a4e1226177260e \
go test ./internal/components/klv -run TestPublicKLVSampleSmokeWithLocalPath -count=1 -v
```

Result: passed. Two initial failed runs exposed path-resolution bugs in the smoke harness; the test now accepts
repo-root relative paths by resolving them to absolute local file URIs before demux.

## Follow-Ups

- Keep deterministic SemOps truth fixtures as the engineering-support acceptance gate.
- Redistribution/license review remains open before any public media can be vendored or treated as a cleared fixture.
