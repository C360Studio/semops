# Fixture Manifest Tier Review

Date: 2026-06-23

## Decision

Accept the first fixture manifest gate as SemOps product governance for portable COP demo data.

The manifest is useful now because the demo needs data that travels, while live captures, public examples, generated
media, and synthetic story fixtures have different claim power. The gate must remain product evidence, not a claim that
SemStreams itself has a universal fixture framework.

## Objections Raised

- A manifest can create false confidence if synthetic fixtures are mistaken for captured provider behavior.
- Optional ignored captures can rot locally while the manifest still passes on machines without those files.
- SHA-256 and byte-size checks prove artifact integrity, not protocol conformance or provider compatibility.
- The ignored-directory allowlist can become a back door for unreviewed data if it grows without review.
- Large media samples can accidentally enter the repo if KLV/DJI/video fixture promotion is not watched closely.

## Evidence Checked

- `fixtures/manifest.json` records CAP, DJI, weather, and KLV fixture tier, provenance, claim scope, review, SHA-256,
  size, observed fields, and synthetic fields.
- `go test ./internal/fixturemanifest` validates manifest structure, committed artifact hashes/sizes, review links,
  allowed tiers/statuses, and optional ignored live captures when present.
- The manifest validator walks portable files under `fixtures/` and fails if a new fixture bypasses the ledger.
- Ignored local paths remain limited to CAP schema/capture/replay folders and KLV generated/public-sample/cache media.
- `fixtures/cap/lifecycle/hadr-flood.jsonl` is a committed derived story fixture checked byte-for-byte against the CAP
  lifecycle generator.

## Accepted Risks

- The current manifest covers file-backed fixture artifacts, not code-generated scenario data for every feed.
- CAP's live NWS sample remains ignored local evidence; it does not travel with the demo.
- ADS-B and SAPIENT still rely heavily on code-hosted/mock-provider fixtures rather than committed portable files.
- Manifest discipline does not replace feed-specific adversarial review before product, standards, or compliance
  language appears in demos or proposals.

## Follow-Up Tasks

- Add manifest entries whenever ADS-B, SAPIENT, MAVLink, TAK/CoT, or CS API fixtures become file-backed portable data.
- Promote real NWS update/cancel/expire samples only after source, license, retention, and claim-language review.
- Keep KLV/DJI media samples ignored until legal provenance and repo-size posture are reviewed.
- File a SemStreams fixture/tier placement ask only if multiple C360 products need the same manifest contract.
