# KLV Deterministic Fixture Review

Date: 2026-06-22

## Decision

Accept a SemOps-owned deterministic MISB ST 0601 truth fixture as the first executable KLV engineering-support gate.
The fixture lives as JSON truth data, is encoded into supported ST 0601 local-set packet bytes at test time, and is
decoded through the same SemOps KLV decoder used by the component worker path.

This is not a substitute for public MPEG-TS smoke evidence or formal STANAG 4609 conformance.

## Guardrails

- Keep fixture truth small, reviewable, and committed as text.
- Treat the generated KLV packet as test output, not a large vendored binary.
- Assert exact field presence and timestamps, but compare scaled numeric values within MISB integer quantization
  tolerances.
- Keep graph writes out of this fixture gate.
- Do not describe the fixture as live media, streaming binary, public-sample, or conformance evidence.

## Evidence

- `fixtures/klv/misb0601-truth.json` records the deterministic source truth.
- `internal/components/klv/fixture.go` encodes the supported ST 0601 subset into KLV local-set bytes.
- `internal/components/klv/fixture_test.go` decodes the generated packet back to a frame payload and compares against
  the truth fixture.
- `go test ./internal/components/klv` passes without network downloads or FFmpeg.

## Follow-Up

- Add optional MPEG-TS wrapping around the deterministic packet if the demux path needs end-to-end media-container
  proof.
- Add a public sample smoke only after license, provenance, size, and cache policy review.
- Add footprint polygon tags or derived footprint logic before claiming footprint extraction.
