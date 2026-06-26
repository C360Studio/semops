# ADS-B Feed Evidence

Status: candidate Phase 2 air-picture feed with parser, replay, hosted-adapter seam, OpenSky-shaped HTTP component
package, opt-in app-runtime wiring, opt-in structural scenario replay, projection, ownership registration, and COP
readback evidence.

## Decision

ADS-B should enter after MAVLink, TAK/CoT, CAP, and the structural COP are stable. Start with OpenSky-shaped JSON
fixtures and deterministic replay. Treat ASTERIX, raw receiver protocols, and live OpenSky access as later expansion.

The current ADS-B scenario replay seam remains an in-process deterministic harness for replaying snapshots through
parser, projection, ownership, and graph-write contracts. SemOps now also has the first real hosted ingress shape:
`internal/components/adsb` provides an OpenSky-compatible HTTP poller input component, raw decoder processor, and
born-first graph projector processor with declared `HTTPClientPort`, `TimerPort`, stream ports, request ports,
registered payloads, health, flow metrics, replay capture, and provider-shaped local HTTP tests. `cmd/semops` can now
wire that poller -> decoder -> projector chain behind `SEMOPS_ADSB_ENABLED=true`, minting `semops.feed.adsb`
ownership only for the enabled runtime. This does not make live OpenSky part of the default MVP stack and does not
cover readsb/dump1090, receiver TCP/UDP, ASTERIX, or provider reliability. See
`openspec/changes/revive-cop-product/reviews/2026-06-20-adsb-component-promotion-review.md` and
`openspec/changes/revive-cop-product/reviews/2026-06-21-adsb-component-promotion-review.md` plus
`openspec/changes/revive-cop-product/reviews/2026-06-21-adsb-runtime-flow-review.md`.

## Local Evidence

- `pkg/adapters/adsb` parses OpenSky state-vector snapshots from bounded JSON fixtures.
- The first canonical model already includes `track`, which can represent aircraft state.
- The feed ladder assigns aircraft current state to `signal`, association evidence to `control`, and raw receiver or
  replay rows to `trace`.
- Parser tests preserve nullable callsign, position timestamp, longitude, latitude, altitude, velocity, track,
  vertical rate, receiver IDs, squawk, position source, and category fields before projection.
- `pkg/adapters/adsb` provides deterministic OpenSky snapshot fixture records plus JSONL replay load/store support.
- `fixtures/adsb/opensky-hadr.jsonl` is the committed portable replay fixture for this OpenSky-shaped ADS-B gate. It
  is manifest-listed, generated from `OpenSkyFixtureRecords`, and not captured OpenSky, raw ADS-B, receiver, or
  ASTERIX evidence.
- `internal/projectors/adsb` projects aircraft current state into source-partitioned ADS-B track entities with
  `indexing_profile=signal`, provenance, confidence, source references, and no cross-source association edge.
- `internal/projectors/adsb` has a graph writer boundary for SemStreams create/update mutation request/reply
  contracts and ADR-060 classified graph mutation errors.
- `internal/scenario` can replay ADS-B snapshot records through parse, projection, graph-plan writing, and born-state
  marking through the hosted adapter seam when a scenario opts into ADS-B.
- `internal/adapters/adsb` hosts an OpenSky-shaped snapshot ingest seam with bounded raw capture, JSONL replay
  append, projection writes, born-first reconciliation, and pollable health counters.
- `internal/components/adsb` promotes OpenSky-compatible HTTP polling into SemStreams input/processor components with
  `HTTPClientPort`, `TimerPort`, registered `message.BaseMessage` payloads, stream ports, graph request ports,
  replay capture, stale-source health, and local provider-shaped HTTP fixture tests.
- `cmd/semops` can opt into the ADS-B HTTP poller -> decoder -> graph-projector chain with `SEMOPS_ADSB_ENABLED=true`,
  `SEMOPS_ADSB_HTTP_URL`, stale/source replay settings, and config-driven raw-lane caps.
- App-runtime ADS-B ownership appends `semops.feed.adsb` only when the hosted ADS-B flow is enabled, matching the
  scenario-runner opt-in owner-token discipline.
- `internal/stack.NewADSBAdapter` composes the adapter with either a SemStreams NATS requester or injected writer.
- `internal/stack.NewADSBPlanWriter` exposes the same SemStreams graph request writer for component runtime wiring
  without calling projector internals from the app layer.
- `cmd/semops-scenario-runner` can opt into ADS-B replay with `SEMOPS_SCENARIO_ADSB_FIXTURE=true`; the Compose
  service passes the flag through but defaults it off.
- The scenario runner appends `semops.feed.adsb` ownership only for the opt-in ADS-B path; this is not a live OpenSky
  or receiver service claim.
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
- The committed replay JSONL matches the generator and fixture manifest. [done]
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
- Typed `entity_already_exists` birth conflicts reconcile into update-only projection for already-born tracks. [done]
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

### Component Promotion Gate

Target command:

```bash
go test ./internal/components/adsb ./internal/contracts -run ADSB
```

Acceptance:

- OpenSky-compatible HTTP polling is represented as a SemStreams input component with an `HTTPClientPort`,
  `TimerPort`, raw stream output, config schema, health, flow metrics, and provider-contact debug state. [done]
- Raw and decoded ADS-B snapshots use registered `message.BaseMessage` payloads and tappable stream subjects. [done]
- The decoder captures provider-shaped raw JSON into the ADS-B replay store before publishing decoded snapshots.
  [done]
- Malformed snapshots are captured and replayed before parse failure, without graph writes. [done]
- The projector writes source-partitioned ADS-B track plans through SemStreams create/update request ports and
  reconciles already-born track births into update-only projection. [done]
- Component tests use local HTTP fixture servers and do not claim live OpenSky reliability or credentials. [done]

### App Runtime Component Gate

Target command:

```bash
go test ./internal/app -run ADSB
```

Acceptance:

- The hosted app composes ADS-B as SemStreams input -> decoder processor -> projector processor only when
  `SEMOPS_ADSB_ENABLED=true`. [done]
- Runtime ownership includes `semops.feed.adsb` only for the enabled ADS-B component flow. [done]
- `SEMOPS_ADSB_REPLAY_PATH`, raw-lane bounds, HTTP URL, poll interval, stale-after, contact policy, and write timeout
  are config/env-driven. [done]
- Local provider-shaped HTTP fixtures can drive poller -> decoder -> projector writes and append replay without live
  network access. [done]
- Compose passes ADS-B runtime env through but defaults `SEMOPS_ADSB_ENABLED=false`. [done]

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

- Default ADS-B runtime enablement remains off; opt-in OpenSky-compatible HTTP polling is not a provider reliability
  or credentialed-live-service claim.
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
