# ADS-B Projection Readback Review

Date: 2026-06-20
Scope: `COP-007` ADS-B current-state projection/readback gate

## Decision

Accept the ADS-B slice as parser plus projection-plan plus COP API readback evidence. This approves
source-partitioned ADS-B aircraft tracks with `signal` indexing, provenance, confidence, and source references. It
does not approve a hosted ADS-B service, live OpenSky polling, receiver protocols, ASTERIX, or cross-source aircraft
association.

## Objections Reviewed

- OpenSky is a convenient JSON feed shape, not ADS-B protocol coverage. The slice must stay fixture-first and
  optional-live later.
- Nullable state-vector fields make false coordinates easy. The projector now omits `cop.track.position` when a
  state vector lacks latitude or longitude.
- `position_source` can be absent. The projector treats absent source quality as lower-confidence unknown evidence
  instead of inheriting enum zero as ADS-B.
- Aircraft association is tempting but dangerous in the adapter. ADS-B projection does not emit `cop.track.source`
  or any cross-source association edge; correlation remains fusion-owned.
- High-cardinality receiver rows should not become semantic graph noise. Current-state tracks are projected; raw
  rows remain future bounded-lane or trace evidence.
- Runtime owner registration should not imply hosted support. `semops.feed.adsb` has a contract and projector, but
  it is not part of first-phase hosted owner registration until a hosted adapter exists.

## Evidence

- `pkg/cop` declares `OwnerADSB` and `ADSBTrackContract` with `replace-owned` signal predicates and no foreign
  edges.
- `internal/projectors/adsb` creates or updates `c360.<platform>.cop.adsb.track.<icao24>` entities with typed
  message metadata, owner tokens, source refs, confidence, and restart born-state seeding.
- `internal/api/cop` discovers `c360.<platform>.cop.adsb.track.*` prefixes and maps fresh aircraft tracks to
  `feed.adsb` health and ADS-B owner provenance.
- Targeted evidence command: `go test ./pkg/cop ./internal/projectors/adsb ./internal/api/cop -count=1`.

## Follow-Ups

- Add deterministic ADS-B fixture replay before a hosted adapter.
- Add a hosted adapter seam with explicit health counters before any live OpenSky or receiver mode.
- Re-check current OpenSky auth/rate-limit behavior before live mode work.
- Keep ASTERIX, raw receiver protocols, and statistical aircraft association behind separate adversarial reviews.
