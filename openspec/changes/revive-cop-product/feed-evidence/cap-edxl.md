# CAP/EDXL Feed Evidence

Status: initial Phase 1 parser/projection/readback slice exists, with a skipped-by-default live graph smoke for
born-first append-evidence behavior. Hosted CAP polling, NWS fixture capture, XML schema validation, and
consumer-rule coverage remain open.

## Decision

CAP should be the third feed because it proves loose civilian warning ingestion early. It should not be treated like a
strict tactical source. CAP writes hazard, advisory, expiry, and provenance evidence; it does not overwrite stricter
track or asset facts.

## Local Evidence

- `pkg/adapters/cap` parses CAP alert, info, area, polygon, circle, resource, geocode, and parameter fields used by
  the first civilian-warning fixtures.
- `internal/projectors/cap` births source-partitioned `hazard_area` entities and appends CAP evidence through the
  `semops.cop.hazard.cap-evidence` contract.
- `internal/api/cop` reads CAP hazard evidence JSON into the COP hazard overlay view model.
- `internal/smoke/cap` writes CAP create and update plans through a live SemStreams graph stack, polls
  `graph.query.prefix`, and checks that CAP evidence did not claim authoritative hazard predicates.
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

Target command after fixture tooling exists:

```bash
go test ./internal/feeds/cap
```

Acceptance:

- Local fixtures cover active alert, update, cancel/expire, polygon, circle, resource link, and multilingual info.
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

### COP Readback Gate

Current target command:

```bash
go test ./internal/api/cop
```

Acceptance:

- A graph-backed CAP `hazard_area` with `cop.hazard.evidence` renders as a COP hazard overlay.
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

Target artifact:

- A fixture pack with alert, update, cancel, expire, polygon, circle, and malformed CAP messages.

Acceptance:

- Replaying the fixture yields deterministic hazard/advisory state and visible stale/expiry transitions.

## Known Gaps

- EDXL beyond CAP is not scoped for Phase 1.
- NWS is a useful public source, but live NWS calls should not be required for deterministic CI.
- CAP conformance should be stated as schema/consumer-rule evidence until we implement a proper consumer profile.
- The current CAP slice does not host a poller/webhook service, fetch NWS alerts, or model update/cancel/expire
  lifecycle beyond the parsed evidence document.
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
