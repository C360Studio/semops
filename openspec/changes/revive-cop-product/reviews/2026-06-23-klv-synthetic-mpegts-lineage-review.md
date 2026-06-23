# KLV Synthetic MPEG-TS Lineage Review

Date: 2026-06-23

## Decision

Keep SemOps-generated MPEG-TS fixtures fully synthetic for the MVP.

The preferred lineage is:

1. SemOps-authored MISB ST 0601 truth JSON.
2. SemOps-generated KLV packet bytes.
3. FFmpeg `lavfi` `testsrc` video generated at fixture-build time.
4. FFmpeg MPEG-TS mux output written only to ignored local/generated paths.

Do not use prebuilt third-party `.ts` binaries, scraped broadcast samples, or externally downloaded media as committed
fixtures. Do not use CC BY media such as Big Buck Bunny or Sintel as the default QA fixture path, because attribution
and source-retention duties are unnecessary for the current container/demux proof.

## Evidence Accepted

- `cmd/semops-klv-fixture` uses FFmpeg `-f lavfi -i testsrc=size=16x16:rate=1` plus the deterministic KLV packet
  input to generate `fixtures/klv/generated/deterministic.ts`.
- `cmd/semops-klv-fixture/main_test.go` now asserts the generator uses the synthetic `lavfi` source and MPEG-TS mux
  options instead of an external video input.
- `scripts/cop-stack-smoke.sh` generates the deterministic MPEG-TS file under `fixtures/klv/generated/`, an ignored
  local path, before enabling the opt-in KLV stack smoke.
- `internal/components/klv/mpegts_test.go` independently exercises temp-file MPEG-TS generation and demux/decode when
  local FFmpeg tooling is present.

## Future Video Work

Operator-facing "eye candy" video remains a later media-production task.

That future work may use either:

- Pure synthetic FFmpeg filtergraph video with richer overlays, motion, and generated audio.
- SemOps-authored rendered assets.
- External CC0 or otherwise cleared media only after a fixture review records license, attribution, source URL,
  retention policy, SHA-256, and claim scope.

It must stay separate from the current KLV/MISB acceptance path unless it also carries reviewed KLV metadata and passes
the demux/decode/projector gates.
