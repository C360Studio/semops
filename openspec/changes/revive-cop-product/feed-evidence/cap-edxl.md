# CAP/EDXL Feed Evidence

Status: initial Phase 1 parser/projection/readback slice exists, with deterministic raw XML lifecycle fixture replay,
derived lifecycle-status readback, a skipped-by-default live graph smoke for born-first append-evidence behavior, and
the first hosted HTTP poller, decoder, and graph-projector component package. The app runtime can compose that chain
behind `SEMOPS_CAP_ENABLED=true`, and `SEMOPS_CAP_REPLAY_PATH` can capture provider-shaped raw CAP XML replay records,
while default live-provider enablement remains off. Live NWS fixture capture, XML schema validation, and consumer-rule
coverage remain open.

## Decision

CAP should be the third feed because it proves loose civilian warning ingestion early. It should not be treated like a
strict tactical source. CAP writes hazard, advisory, expiry, and provenance evidence; it does not overwrite stricter
track or asset facts.

The current CAP slice now has the first SemStreams component package for a hosted HTTP poller, raw-alert decoder, and
born-first graph projector. That is a component-contract and deterministic local-polling gate, not a default live NWS
service claim. CAP remains parser, projection, scenario-replay, readback, opt-in runtime, component stale-source
health, and live graph smoke evidence until SemOps proves provider fixtures, webhook ingestion,
NWS/IPAWS/vendor integration, provider lifecycle behavior, and alert-source operations. SemStreams
`v1.0.0-beta.114` provides `HTTPClientPort` for CAP/NWS-style outbound HTTP pollers, while SemStreams issue #309
tracks richer component backpressure telemetry and issue #312 tracks first-class `TimerPort` flowgraph cadence
semantics.

## Local Evidence

- `pkg/adapters/cap` parses CAP alert, info, area, polygon, circle, resource, geocode, and parameter fields used by
  the first civilian-warning fixtures.
- `pkg/adapters/cap` stores replayable raw XML CAP records and provides a HA/DR flood lifecycle fixture covering
  alert, update, cancel, and expired-alert records.
- `internal/projectors/cap` births source-partitioned `hazard_area` entities and appends CAP evidence through the
  `semops.cop.hazard.cap-evidence` contract.
- `internal/api/cop` reads CAP hazard evidence JSON into the COP hazard overlay view model and derives
  operator-facing status from CAP `msgType`, `status`, `expires`, and read-time freshness.
- `internal/smoke/cap` writes CAP create and update plans through a live SemStreams graph stack, polls
  `graph.query.prefix`, and checks that CAP evidence did not claim authoritative hazard predicates.
- `internal/components/cap` adds a SemStreams lifecycle `HTTPPollerComponent` with `HTTPClientPort` plus sibling
  `TimerPort`, a raw-alert decoder processor, a born-first graph-projector processor, registered raw/decoded
  `message.BaseMessage` payloads, graph request ports, config schemas, health, and flow metrics.
- The CAP HTTP poller records provider contact separately from fresh payload data, preserves `ETag`/`Last-Modified`,
  treats `304 Not Modified` as contact without duplicate raw publish, and reports `stale` health after `stale_after`
  elapses without fresh CAP XML.
- The hosted app runtime can attach a CAP replay store with `SEMOPS_CAP_REPLAY_PATH`; local HTTP tests prove poller
  output flows through the decoder into native `RawAlertRecord` JSONL and then parses back as CAP XML.
- The COP model reserves `hazard_area`, `alert`, and `advisory` as first-slice entities.
- The feed ladder assigns current CAP evidence to `content`, future authoritative alert lifecycle to `control`, and
  fetch/replay detail to `trace`.

## External Evidence

- CAP 1.2 is an OASIS Standard with a declared XML namespace, normative alert structure, XML schema, and
  conformance sections for messages, producers, and consumers.
- CAP is designed for exchanging all-hazard emergency alerts and public warnings.
- NWS exposes forecasts, alerts, and observations through `api.weather.gov`.
- NWS supports `application/cap+xml` responses and active-alert filters such as `/alerts/active?area={state}`.

## Gates

### Parser Gate

Current target command:

```bash
go test ./pkg/adapters/cap
```

Acceptance:

- Local CAP examples parse into alert, info, resource, and area structures.
- Polygon and circle areas are preserved as geometry evidence.
- Message, producer, and consumer conformance rules are represented as tests where practical.
- Malformed XML and invalid required fields fail before graph writes.

### Sample Source Gate

Current target command:

```bash
go test ./pkg/adapters/cap
```

Acceptance:

- Local fixtures cover active alert, update, cancel, expired alert, polygon, circle, resource link, and parser
  rejection cases.
- NWS samples are captured into fixtures rather than required live for CI.
- Optional live mode respects NWS User-Agent guidance, caching, and rate-limit behavior.

### Projection Gate

Current target command:

```bash
go test ./internal/projectors/cap
```

Acceptance:

- CAP advisory and geometry evidence uses `indexing_profile=content` through the append-evidence contract.
- Future authoritative alert lifecycle and hazard status state will use `indexing_profile=control`.
- Poll history, raw fetches, and replay steps use `indexing_profile=trace`.
- CAP evidence appends to source-owned predicates and does not replace stricter feed state.
- Expired alerts become stale or inactive in the COP instead of silently remaining active.

### Component Gate

Current target command:

```bash
go test ./internal/components/cap ./internal/contracts
```

Acceptance:

- The HTTP poller declares a SemStreams `HTTPClientPort` for method, URL pattern, auth reference, contact policy, and
  raw-alert interface metadata.
- The poller declares a sibling `TimerPort` referenced by `HTTPClientPort.TriggerPort` so polling cadence is visible
  as a component contract.
- The poller config schema exposes `stale_after`; component health degrades to `stale` after no fresh provider payload
  arrives within that threshold.
- `304 Not Modified` provider responses update provider-contact debug state without publishing duplicate raw CAP XML.
- Raw and decoded CAP payloads register with the SemStreams `PayloadRegistry` and round-trip through
  `message.BaseMessage`.
- Deterministic local HTTP tests publish raw CAP XML without calling live NWS.
- The decoder processor parses raw CAP XML, appends replay evidence when configured, and emits decoded CAP alerts on
  a stream port.
- The projector processor consumes decoded CAP alerts, writes born-first graph mutation requests, and reconciles
  restart-time create conflicts without relying on graph auto-vivify.

### COP Readback Gate

Current target command:

```bash
go test ./internal/api/cop
```

Acceptance:

- A graph-backed CAP `hazard_area` with `cop.hazard.evidence` renders as a COP hazard overlay.
- CAP alert, update, cancel, expire, stale, and non-operational test evidence maps to explicit hazard view-model
  status without writing `cop.hazard.status`.
- When multiple CAP evidence triples exist on one hazard, the newest evidence and source reference drive readback.
- CAP feed health is live or stale based on graph observation timestamps.
- Provenance identifies `semops.feed.cap` and the source reference.
- Missing CAP graph state is treated as cold-start or fallback state, not as a successful empty decode.

### Live Graph Gate

Current target command:

```bash
SEMOPS_CAP_LIVE_GRAPH_NATS_URL=<nats-url> go test ./internal/smoke/cap -run TestLiveGraphCAPBornFirstSmoke -v
```

Acceptance:

- CAP creates a `hazard_area` entity before appending update evidence.
- The update path does not fail with `entity_not_found` or `foreign_edge_dropped`.
- Prefix discovery can read the CAP hazard entity through `graph.query.prefix`.
- CAP evidence includes update provenance while leaving `cop.hazard.geometry`, `cop.hazard.severity`, and
  `cop.hazard.status` unowned.

### Replay Gate

Current target command:

```bash
go test ./pkg/adapters/cap ./internal/components/cap ./internal/app ./internal/projectors/cap
```

Acceptance:

- Replaying the raw XML lifecycle fixture yields deterministic alert/update/cancel/expired alert parse output.
- Projecting the lifecycle fixture births the first hazard, appends update/cancel evidence to that hazard, and births
  a separate expired hazard without relying on auto-vivify.
- Provider-shaped HTTP CAP responses can be captured through the poller and decoder into replay JSONL without live NWS
  access.
- The hosted runtime honors `SEMOPS_CAP_REPLAY_PATH` for opt-in CAP replay capture while keeping
  `SEMOPS_CAP_ENABLED=false` by default.

## Known Gaps

- EDXL beyond CAP is not scoped for Phase 1.
- NWS is a useful public source, but live NWS calls should not be required for deterministic CI.
- CAP conformance should be stated as schema/consumer-rule evidence until we implement a proper consumer profile.
- The initial `internal/components/cap` package is wired into the app runtime and Compose as an explicit opt-in, but
  `SEMOPS_CAP_ENABLED` defaults to `false` and does not fetch live NWS alerts by default.
- `SEMOPS_CAP_REPLAY_PATH` is optional and empty by default; setting it captures decoded CAP raw XML records from the
  opt-in runtime path.
- Captured NWS update/cancel/expire fixture replay is still missing.
- Provider-specific stale-source review is still needed after captured NWS/IPAWS/vendor fixtures exist; the current
  stale-source gate is component-level health behavior with deterministic local HTTP tests.
- Default live-provider CAP polling is still not enabled in the default Compose stack.
- The current projector intentionally does not own `cop.hazard.geometry`, `cop.hazard.severity`, or
  `cop.hazard.status`.
- The live graph smoke is SemStreams graph-contract evidence, not CAP consumer conformance evidence.

## Adversarial Feed-Entry Questions

- Are we treating CAP as advisory evidence instead of authoritative track or asset truth?
- Does the parser preserve enough area/resource structure for provenance and operator inspection?
- Are text-heavy fields indexed as `content` without poll/replay noise leaking into semantic indexing?
- Are update/cancel/expire semantics visible in the COP?
- Is live NWS access optional and replayable?

## Source Links

- OASIS CAP 1.2: <https://docs.oasis-open.org/emergency/cap/v1.2/CAP-v1.2-os.pdf>
- NWS API: <https://www.weather.gov/documentation/services-web-api>
