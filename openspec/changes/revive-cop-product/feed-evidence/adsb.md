# ADS-B Feed Evidence

Status: candidate Phase 2 air-picture feed with parser, replay, hosted-adapter seam, opt-in structural scenario
replay, projection, ownership registration, and COP readback evidence.

## Decision

ADS-B should enter after MAVLink, TAK/CoT, CAP, and the structural COP are stable. Start with OpenSky-shaped JSON
fixtures and deterministic replay. Treat ASTERIX, raw receiver protocols, and live OpenSky access as later expansion.

The current ADS-B scenario replay seam is intentionally not a SemStreams component package. It is an in-process,
deterministic harness for replaying snapshots through parser, projection, ownership, and graph-write contracts. When
SemOps adds live ADS-B ingress, that live boundary must be promoted into SemStreams input and processor components
with declared network, file, or request ports, registered payloads, health, flow metrics, and telemetry-driven
backpressure decisions. See
`openspec/changes/revive-cop-product/reviews/2026-06-20-adsb-component-promotion-review.md`.

## Local Evidence

- `pkg/adapters/adsb` parses OpenSky state-vector snapshots from bounded JSON fixtures.
- The first canonical model already includes `track`, which can represent aircraft state.
- The feed ladder assigns aircraft current state to `signal`, association evidence to `control`, and raw receiver or
  replay rows to `trace`.
- Parser tests preserve nullable callsign, position timestamp, longitude, latitude, altitude, velocity, track,
  vertical rate, receiver IDs, squawk, position source, and category fields before projection.
- `pkg/adapters/adsb` provides deterministic OpenSky snapshot fixture records plus JSONL replay load/store support.
- `internal/projectors/adsb` projects aircraft current state into source-partitioned ADS-B track entities with
  `indexing_profile=signal`, provenance, confidence, source references, and no cross-source association edge.
- `internal/projectors/adsb` has a graph writer boundary for SemStreams create/update mutation request/reply
  contracts.
- `internal/scenario` can replay ADS-B snapshot records through parse, projection, graph-plan writing, and born-state
  marking through the hosted adapter seam when a scenario opts into ADS-B.
- `internal/adapters/adsb` hosts an OpenSky-shaped snapshot ingest seam with bounded raw capture, JSONL replay
  append, projection writes, born-first reconciliation, and pollable health counters.
- `internal/stack.NewADSBAdapter` composes the adapter with either a SemStreams NATS requester or injected writer.
- `cmd/semops-scenario-runner` can opt into ADS-B replay with `SEMOPS_SCENARIO_ADSB_FIXTURE=true`; the Compose
  service passes the flag through but defaults it off.
- The scenario runner appends `semops.feed.adsb` ownership only for the opt-in ADS-B path; this is not a live OpenSky
  or receiver service claim.
- No `internal/components/adsb` package exists yet by design; component promotion waits for a real hosted ingress
  decision rather than wrapping deterministic fixtures.
- COP graph prefix discovery reads `c360.<platform>.cop.adsb.track.*` entities back into aircraft tracks and feed
  health without requiring a live ADS-B service.

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

- Replay produces deterministic current aircraft state, missing-position state, and non-ADS-B position-source state.
  [done]
- Fixture replay is the CI default; live OpenSky access is optional. [done]
- Replay records carry source refs without putting receiver rows directly into canonical track entities. [done]
- Scenario replay can process two OpenSky snapshots so repeated ICAO24 state updates after the first birth. [done]

### Projection Gate

Target command:

```bash
go test ./internal/projectors/adsb
```

Acceptance:

- Aircraft current state uses `indexing_profile=signal`. [done]
- Missing position data remains partial evidence and never emits fake coordinates. [done]
- Receiver/replay rows remain outside canonical track entities; future raw rows use bounded lanes or `trace`.
- Cross-source association with MAVLink/SAPIENT is separate fusion evidence, not an adapter side effect. [done]
- Position source is preserved as provenance/confidence evidence. [done]
- Restart reconciliation can seed known ADS-B track births before update-only projection. [done]

### Readback Gate

Target command:

```bash
go test ./internal/api/cop -run TestGraphProviderDiscoversADSBTracksByPrefix
```

Acceptance:

- Prefix discovery includes ADS-B track entities. [done]
- ADS-B track readback maps callsign/ICAO, position, velocity, source, provenance, and owner into the COP snapshot.
  [done]
- `feed.adsb` is live only when graph-backed ADS-B tracks are fresh; otherwise it remains planned/pending. [done]

### Hosted Adapter Seam Gate

Target command:

```bash
go test ./internal/adapters/adsb ./internal/stack -run ADSB
```

Acceptance:

- The adapter captures raw OpenSky-shaped snapshots on a bounded lane and appends replay records before projection.
  [done]
- The adapter projects snapshots through the ADS-B projector/writer seam and records graph mutation health. [done]
- Malformed snapshots are captured and replayed before parse failure, without graph writes. [done]
- `entity_already_exists` birth conflicts reconcile into update-only projection for already-born tracks. [done]
- Stack wiring can use SemStreams NATS request/reply with retry configuration or an injected writer. [done]

### Structural Scenario Gate

Target command:

```bash
SEMOPS_SCENARIO_ADSB_FIXTURE=true ./scripts/cop-stack-smoke.sh
```

Acceptance:

- The hosted scenario runner appends two deterministic OpenSky-shaped ADS-B snapshots only when explicitly enabled.
  [done]
- ADS-B scenario replay uses `internal/adapters/adsb.Adapter`, not a test-only projector/writer shortcut. [done]
- ADS-B graph writes are backed by a SemStreams-minted `semops.feed.adsb` owner token. [done]
- The default stack path remains MAVLink, TAK/CoT, and CAP so live ADS-B is not implied. [done]

### Live Mode Gate

Target command after optional live mode exists:

```bash
go test ./internal/feeds/adsb -run TestOpenSkyLiveSmoke
```

Acceptance:

- Test is opt-in and skips without credentials/network.
- Rate-limit and authentication behavior is explicit.
- Live mode never replaces fixture replay as the deterministic acceptance gate.
- Live mode uses SemStreams input and processor components rather than calling `internal/adapters/adsb` directly from a
  service loop.

## Known Gaps

- No ADS-B live client yet.
- No ADS-B component package yet; this is blocked on the live ingress choice.
- OpenSky is useful for samples but should not become a critical-path dependency.
- ASTERIX is not in the first ADS-B slice.
- Raw receiver/readsb/dump1090 paths are not implemented.
- Cross-source aircraft association is not implemented and remains fusion-owned.

## Adversarial Feed-Entry Questions

- Are we preserving source quality rather than treating every air track equally?
- Are nullable fields handled honestly?
- Are live API calls optional and rate-limit aware?
- Is cross-source identity resolution kept out of the adapter?
- Are high-cardinality receiver rows kept out of semantic indexing?

## Source Links

- OpenSky REST API: <https://openskynetwork.github.io/opensky-api/rest.html>
