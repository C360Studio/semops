# ADS-B And SAPIENT Portable Fixtures Review

Date: 2026-06-23

## Decision

Accept small committed portable fixtures for ADS-B OpenSky-shaped replay and SAPIENT runtime preflight/graph-smoke
paths.

These fixtures improve demo portability because the local fixture service and scenario runner no longer depend only on
code-hosted payload constructors for their story. They remain engineering and demo data, not live provider evidence.

## Objections Raised

- OpenSky-shaped JSON can be mistaken for captured ADS-B or OpenSky provider traffic.
- SAPIENT-shaped JSON can be mistaken for Dstl harness output, Apex middleware traffic, or SAPIENT compliance.
- The SAPIENT absolute-location fixture can create false confidence because it deliberately avoids UTM and
  range/bearing edge cases.
- Runtime fixture-service payloads can drift away from committed portable files if not checked.
- Adding portable files can make the manifest feel complete while MAVLink/TAK generated fixtures still live in code.

## Evidence Checked

- `fixtures/adsb/opensky-hadr.jsonl` is generated from `pkg/adapters/adsb.OpenSkyFixtureRecords` and checked back
  against that generator.
- The ADS-B fixture includes a repeated ICAO24 update, missing-position evidence, and an MLAT position-source row.
- `fixtures/sapient/task-ack.json` matches `sapient.TaskAckFixtureJSON()` and preserves preflight-only decoded stream
  behavior.
- `fixtures/sapient/absolute-detection.json` matches `sapient.DetectionFixtureJSON()` and stays within the reviewed
  LAT_LNG_DEG_M/WGS84 absolute-location projection subset.
- `fixtures/manifest.json` records provenance, tier, SHA-256, size, review path, synthetic fields, and claim scope for
  all three new fixtures.
- `go test ./pkg/adapters/adsb ./pkg/adapters/sapient ./internal/fixturemanifest` verifies file/generator/runtime
  alignment and manifest coverage.

## Accepted Risks

- ADS-B live OpenSky credentials, rate limits, provider reliability, readsb/dump1090, raw receiver protocols, ASTERIX,
  and cross-source association remain unproven.
- SAPIENT product service support, Dstl harness compliance, tasking, association, alert acknowledgements, UTM
  conversion, and range/bearing projection remain unproven.
- The fixture manifest still does not cover every code-generated MAVLink/TAK scenario datum; those should be promoted
  only when they become portable file-backed data.

## Follow-Up Tasks

- Keep live provider captures ignored until source, license, retention, and claim-language review clears promotion.
- Add file-backed MAVLink/TAK entries only if the current generated fixtures become demo-travel artifacts.
- Run or qualify the Dstl harness before SAPIENT compliance language.
- Decide whether ADS-B next needs authenticated OpenSky mode, receiver/readsb/dump1090 input components, or ASTERIX.
