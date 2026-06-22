# KLV Fixture Strategy Review

Date: 2026-06-22

## Decision

Accept the fixture strategy as a four-step ladder:

1. Opaque SemSource synthetic binary fixture for storage/governance proof only.
2. Public KLV MPEG-TS sample smoke in SemOps after license and provenance review.
3. Deterministic MISB ST 0601 fixture in SemOps with truth JSON, encoded KLV, optional MPEG-TS wrapping, and exact
   parsed-output assertions.
4. Formal STANAG 4609 conformance as a separate validator or lab track.

The immediate next slice is not "claim STANAG." It is to design the SemOps-owned KLV/MISB worker boundary and fixture
strategy: storage-reference input, demux/parser choice, derived-fact output, SemStreams ownership/indexing contracts,
component telemetry, and memory/backpressure posture.

## Objections Raised

- Public samples can prove the plumbing sees real KLV, but they do not prove deterministic correctness unless a trusted
  truth sidecar exists.
- Generated fixtures can be too perfect. They must be supplemented with at least one real public smoke sample before
  demo credibility language.
- "MISB ST 0601 support" is an engineering claim. "STANAG 4609 conformance" or "certification" requires a separate
  validator or lab track.
- CoT and CS API JSON should be interop outputs from the worker, not the internal KLV parse model.
- Large binary samples should not enter CI or the repo until license, provenance, size, and cache/download behavior are
  reviewed.

## Evidence Checked

- `klvdata` documents parsing and constructing KLV streams, support for MISB ST 0601 and ST 0102, and the need for
  FFmpeg or GStreamer to demux KLV from MPEG-TS.
- `klvdata` points to a small binary sample and the public FFmpeg `Day Flight.mpg` KLV MPEG-TS sample.
- jMISB documents support status for ST 0601, ST 0805 KLV-to-CoT, and ST 1402 MPEG-2 transport stream support, while
  stating it is not affiliated with or endorsed by MISB.
- FFmpeg documentation confirms data streams require explicit mapping; this matters for KLV extraction commands.
- `klv-uas` exists as a Rust UAS KLV parser candidate, but needs a maturity spike before adoption.

## Accepted Risks

- The first real-world sample may be too large or awkward for CI and may need local-cache or optional-smoke handling.
- A deterministic encoder path may take longer than wiring a parser against an existing `.bin` payload, but it gives
  much stronger acceptance evidence.
- Parser choice is still open: Go-native, jMISB sidecar, Python `klvdata` sidecar, Rust worker, or another library.

## Follow-Up Tasks

- Verify license/provenance and cache policy for any public KLV sample before adding automation.
- Define the supported MISB ST 0601 field subset for the first worker spike.
- Decide whether the first implementation is sidecar-first or native-first.
- Add OpenSpec/design for the KLV/MISB worker component ports, payload registry entries, ownership contracts, and
  telemetry/backpressure expectations.
