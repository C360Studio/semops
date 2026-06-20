# ADS-B And SAPIENT Boundary Review

Date: 2026-06-20

## Decision

Accept a narrow Phase 2 feed-boundary slice:

- ADS-B may enter as OpenSky-shaped fixture parsing in `pkg/adapters/adsb`.
- SAPIENT remains an artifact-discovery boundary with no implementation until authoritative ICD, schema/protobuf,
  samples, validator, or conformance tooling is found.

This does not approve ADS-B graph projection, live OpenSky polling, ASTERIX, raw receiver protocols, SAPIENT product
support, or SAPIENT conformance claims.

## Objections Raised

- OpenSky is not "ADS-B the protocol"; it is a useful JSON feed shape that can include ADS-B, ASTERIX, MLAT, and
  FLARM position sources.
- Nullable OpenSky fields make it easy to fake coordinates or stale position times if the parser normalizes too early.
- Live OpenSky access now requires OAuth2 client credentials for authenticated use and carries rate/credit limits, so
  it should not become a demo critical path.
- SAPIENT remains risky because no authoritative public compliance suite, schema, fixture corpus, or validator has
  been verified.
- Adding ADS-B before association/fusion policy could encourage accidental identity resolution inside the adapter.

## Evidence Checked

- Current OpenSky REST documentation still defines `/states/all` as a JSON object with `time` and row-array `states`
  where callsign, position timestamp, longitude, latitude, altitude, velocity, track, vertical rate, receiver IDs,
  squawk, and category can be nullable or optional.
- OpenSky position-source values distinguish ADS-B, ASTERIX, MLAT, and FLARM, so source quality must be preserved.
- `pkg/adapters/adsb` parses bounded OpenSky snapshot fixtures into a typed intermediate model before graph projection.
- Unit coverage proves normal aircraft rows, missing-position rows, MLAT/UAV rows, unknown-source rows, and malformed
  rows behave explicitly.
- Public SAPIENT searches did not identify an authoritative artifact set suitable for implementation.

## Accepted Risks

- The ADS-B package is parser evidence only; it does not yet write SemStreams graph state.
- OpenSky fixture shape may drift, so live-client work must re-check the official docs and auth/rate-limit behavior.
- The first ADS-B projection will need a separate ownership/indexing review before it writes high-rate aircraft state.
- SAPIENT may become available through non-public partner artifacts; until then, SemOps must not guess the schema.

## Follow-Up Tasks

- Add ADS-B projection contracts and readback only after the parser boundary is stable.
- Keep cross-source aircraft association out of the ADS-B adapter; make it fusion-owned evidence.
- Re-check SAPIENT artifacts before any implementation or conformance wording.
