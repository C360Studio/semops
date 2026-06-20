# ADS-B Feed Evidence

Status: candidate Phase 2 air-picture feed with initial OpenSky-shaped parser evidence.

## Decision

ADS-B should enter after MAVLink, TAK/CoT, CAP, and the structural COP are stable. Start with OpenSky-shaped JSON
fixtures and deterministic replay. Treat ASTERIX, raw receiver protocols, and live OpenSky access as later expansion.

## Local Evidence

- `pkg/adapters/adsb` parses OpenSky state-vector snapshots from bounded JSON fixtures.
- The first canonical model already includes `track`, which can represent aircraft state.
- The feed ladder assigns aircraft current state to `signal`, association evidence to `control`, and raw receiver or
  replay rows to `trace`.
- Parser tests preserve nullable callsign, position timestamp, longitude, latitude, altitude, velocity, track,
  vertical rate, receiver IDs, squawk, position source, and category fields before any graph projection is added.

## External Evidence

- OpenSky exposes REST endpoints for state vectors, flights, and tracks.
- `GET /states/all` returns state vectors with fields such as ICAO24, callsign, position timestamps, longitude,
  latitude, altitude, velocity, track, vertical rate, receivers, squawk, position source, and category.
- OpenSky state-vector calls for sensors other than the caller's own are rate limited.
- Position source values include ADS-B, ASTERIX, MLAT, and FLARM, which lets the adapter preserve source quality.
- Current OpenSky docs require OAuth2 client credentials for authenticated API use; live access is not part of this
  fixture/parser gate.

## Gates

### Parser Gate

Target command:

```bash
go test ./pkg/adapters/adsb
```

Acceptance:

- OpenSky state-vector fixtures decode with nullable fields preserved. [done]
- ICAO24, callsign, position time, last contact, position, velocity, track, vertical rate, and position source map
  into a typed intermediate model. [done]
- Missing position data produces partial evidence rather than fake coordinates. [done]
- Malformed rows fail before projection. [done]

### Replay Gate

Target artifact:

- A bounded OpenSky JSON fixture with at least one normal track, one stale track, one missing-position track, and one
  non-ADS-B position source.

Acceptance:

- Replay produces deterministic current aircraft state and stale-data transitions.
- Fixture replay is the CI default; live OpenSky access is optional.

### Projection Gate

Target command after SemOps graph contracts exist:

```bash
go test ./internal/projectors/adsb
```

Acceptance:

- Aircraft current state uses `indexing_profile=signal`.
- Receiver/replay rows use `indexing_profile=trace`.
- Cross-source association with MAVLink/SAPIENT is separate fusion evidence, not an adapter side effect.
- Position source is preserved as provenance/confidence evidence.

### Live Mode Gate

Target command after optional live mode exists:

```bash
go test ./internal/feeds/adsb -run TestOpenSkyLiveSmoke
```

Acceptance:

- Test is opt-in and skips without credentials/network.
- Rate-limit and authentication behavior is explicit.
- Live mode never replaces fixture replay as the deterministic acceptance gate.

## Known Gaps

- No ADS-B graph projector, hosted adapter, or COP readback path yet.
- OpenSky is useful for samples but should not become a critical-path dependency.
- ASTERIX is not in the first ADS-B slice.
- Raw receiver/readsb/dump1090 paths are not implemented.

## Adversarial Feed-Entry Questions

- Are we preserving source quality rather than treating every air track equally?
- Are nullable fields handled honestly?
- Are live API calls optional and rate-limit aware?
- Is cross-source identity resolution kept out of the adapter?
- Are high-cardinality receiver rows kept out of semantic indexing?

## Source Links

- OpenSky REST API: <https://openskynetwork.github.io/opensky-api/rest.html>
