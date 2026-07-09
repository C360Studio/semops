# KLV Synthetic Packet Fixture Review

Date: 2026-06-23

## Decision

Accept `fixtures/klv/misb0601-packet.hex` as the first portable KLV binary fixture for SemOps and SemSource storage
governance tests.

The fixture is hex-encoded in git for reviewability, but it materializes to an 88-byte MISB ST 0601 KLV packet derived
from `fixtures/klv/misb0601-truth.json`.

## Why This Is Safe

- The packet is SemOps-authored synthetic data, not captured public media, vendor telemetry, or partner-provided data.
- The committed text fixture is small, deterministic, and listed in `fixtures/manifest.json`.
- `internal/components/klv` decodes the materialized bytes and compares them byte-for-byte with the packet generated
  from the truth fixture.
- The artifact gives SemSource a real byte payload for storage/governance proof work without asserting STANAG 4609
  conformance or live streaming support.

## Claim Boundary

This fixture supports only these claims:

- SemOps can carry a portable KLV packet byte artifact through source control and manifest governance.
- The tested MISB ST 0601 subset encodes and decodes deterministically.
- SemSource may use the materialized bytes as a synthetic binary storage/governance proof.

It does not support these claims:

- Public KLV sample redistribution clearance.
- STANAG 4609 conformance.
- MISB ST 0601 full-field coverage.
- MPEG-TS demux fidelity.
- Live video, multi-feed media, or streaming-binary product support.

## Follow-Up Tasks

- Keep searching for a legal representative MPEG-TS KLV sample before public-demo media travels with SemOps.
- Record any partner- or lab-supplied media with license, retention, attribution, classification, and claim scope.
- Add richer packet fixtures only when additional MISB fields are parser-supported and reviewed.
