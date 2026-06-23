# KLV Public Sample Provenance Review

Date: 2026-06-23

## Decision

Keep the FFmpeg-hosted `Day Flight.mpg` KLV sample as ignored local smoke evidence only.

Do not vendor it, add it to `fixtures/manifest.json`, require it in CI, or use it as deterministic acceptance data
unless a later legal/provenance review identifies an explicit redistribution license for the MPEG-TS media sample.

## Sources Checked

- FFmpeg sample archive directory: <https://samples.ffmpeg.org/MPEG2/mpegts-klv/>
- FFmpeg-hosted candidate sample URL: <https://samples.ffmpeg.org/MPEG2/mpegts-klv/Day%20Flight.mpg>
- `klvdata` README: <https://github.com/paretech/klvdata>
- FFmpeg legal page: <https://ffmpeg.org/legal.html>

## Findings

- The FFmpeg sample archive lists `Day Flight.mpg` and `Night Flight IR.mpg` with sizes and modification times, but
  the directory listing does not include license, attribution, redistribution, or chain-of-custody terms for the
  media.
- `klvdata` documents downloading `Day Flight.mpg` from the FFmpeg sample archive and using FFmpeg to extract a KLV
  data stream for parser exploration.
- `klvdata` is MIT-licensed, but that repository license applies to that project and does not grant redistribution
  rights for the FFmpeg-hosted MPEG-TS sample.
- FFmpeg's legal page governs FFmpeg licensing and compliance posture; it does not clear third-party sample media for
  redistribution by SemOps.
- The local ignored file `fixtures/klv/public-samples/day-flight.mpg` has SHA-256
  `a491ceff524b0008e3076d9eb30782badac2d53053731accc0a4e1226177260e` and size `102004664` bytes.

## Accepted Use

- A developer may run the opt-in smoke with an explicitly supplied local file path, source URL, and provenance string.
- The smoke can prove that SemOps can demux and decode a real-world-shaped MPEG-TS/KLV file under bounded local
  conditions.
- Demo and proposal language may describe this only as a local public-sample smoke, not as redistributable fixture
  evidence, formal conformance, official certification, or streaming-binary product support.

## Rejected Use

- Do not commit `Day Flight.mpg` or derived KLV bytes from it.
- Do not require network download from `samples.ffmpeg.org` in CI.
- Do not list this media in the fixture manifest as a portable artifact.
- Do not use the FFmpeg sample archive as a license grant.

## Follow-Up Tasks

- Find a legally redistributable KLV/MISB/STANAG sample before public-demo media travels with the repo.
- Keep deterministic SemOps-authored MISB ST 0601 truth data as the acceptance fixture.
- If a partner or lab supplies media, record license, retention, attribution, classification, and claim scope before
  promotion.
