# KLV SemSource Binary Proof Qualification Review

Date: 2026-06-23

## Decision

Close the KLV/SemSource binary proof spike as storage/governance evidence only.

SemSource's opaque synthetic proof is useful because it exercises the cross-product boundary SemOps needs: raw binary
bytes stay in storage, graph state carries governed metadata and references, and SemOps remains the owner of KLV,
MISB, STANAG, parser, translation, and product-support claims.

This does not authorize any streaming-binary, live media, video-service, KLV parser, STANAG 4609 conformance, or
certification language.

## Evidence Checked

- SemSource documents a SemOps binary source proof boundary that treats the fixture as opaque storage/governance
  evidence and assigns protocol interpretation to SemOps or a SemOps-owned worker.
- SemSource's synthetic proof stores opaque binary bytes by reference, publishes hash, size, byte-range, storage-ref,
  and proof-finding metadata, and keeps raw bytes out of graph triples.
- SemSource's local file store path can stream fixture bytes by reader, while the generic byte-slice fallback remains
  explicit for stores that do not expose a streaming write API.
- SemOps now has a SemOps-authored deterministic MISB ST 0601 packet fixture plus an optional local FFmpeg MPEG-TS
  wrapper generated from synthetic `lavfi` video. Generated media stays ignored and is not vendored.
- SemOps KLV UI/API evidence reads governed `sensor_footprint` graph state only; it does not require raw video or KLV
  bytes in the browser.

## Objections Raised

- A storage/governance proof can look like a streaming-binary proof if the task is closed without caveats.
- SemSource video-handler behavior still matters for future product video paths; the current proof does not by itself
  prove production-scale media ingestion.
- The SemOps deterministic packet/MPEG-TS path proves a narrow engineering subset, not real-world media variability or
  formal conformance.
- If SemSource later offers generic media-track extraction, that service must emit declared SemStreams ports and
  generic media evidence rather than inheriting SemOps KLV/STANAG claims.

## Claim Boundary

Allowed language:

- SemOps and SemSource have a cross-product binary-by-reference storage/governance proof.
- SemSource can provide opaque binary storage references and governed metadata for SemOps-owned feed workers.
- SemOps has deterministic KLV/MISB fixture gates for a tested ST 0601 subset.

Blocked language:

- SemOps supports streaming binary data.
- SemSource is a KLV, MISB, STANAG, SAPIENT, or SKG parser.
- SemOps or SemSource has STANAG 4609 conformance or certification.
- The current proof demonstrates live media ingress, video serving, thumbnails, keyframes, or operator video playback.

## Result

Task `5.8` may close with the caveat that it is storage/governance evidence only. The next KLV product gates remain
public-sample smoke, deterministic MISB subset expansion, footprint polygon extraction, media-scale memory and
backpressure evidence, and any future formal validator/lab track.

## Verification

- SemSource: `go test ./internal/binaryproof`
- SemOps: `openspec validate revive-cop-product --strict`
- SemOps: `git diff --check`
