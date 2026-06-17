# CAP/EDXL Feed Evidence

Status: candidate Phase 1 feed, blocked from implementation by `COP-002`, `COP-007`, and `COP-008`.

## Decision

CAP should be the third feed because it proves loose civilian warning ingestion early. It should not be treated like a
strict tactical source. CAP writes hazard, advisory, expiry, and provenance evidence; it does not overwrite stricter
track or asset facts.

## Local Evidence

- No SemOps CAP adapter exists in the current checkout.
- The COP model already reserves `hazard_area`, `alert`, and `advisory` as first-slice entities.
- The feed ladder already assigns CAP alert lifecycle to `control`, advisory text to `content`, and fetch/replay
  detail to `trace`.

## External Evidence

- CAP 1.2 is an OASIS Standard with a declared XML namespace, normative alert structure, XML schema, and
  conformance sections for messages, producers, and consumers.
- CAP is designed for exchanging all-hazard emergency alerts and public warnings.
- NWS exposes forecasts, alerts, and observations through `api.weather.gov`.
- NWS supports `application/cap+xml` responses and active-alert filters such as `/alerts/active?area={state}`.

## Gates

### Parser Gate

Target command after the SemOps CAP package exists:

```bash
go test ./internal/cap
```

Acceptance:

- OASIS CAP examples parse into alert, info, resource, and area structures.
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

Target command after SemOps graph contracts exist:

```bash
go test ./internal/projectors/cap
```

Acceptance:

- Alert lifecycle and hazard area state use `indexing_profile=control`.
- Advisory/instruction text uses `indexing_profile=content`.
- Poll history, raw fetches, and replay steps use `indexing_profile=trace`.
- CAP evidence appends to source-owned predicates and does not replace stricter feed state.
- Expired alerts become stale or inactive in the COP instead of silently remaining active.

### Replay Gate

Target artifact:

- A fixture pack with alert, update, cancel, expire, polygon, circle, and malformed CAP messages.

Acceptance:

- Replaying the fixture yields deterministic hazard/advisory state and visible stale/expiry transitions.

## Known Gaps

- No SemOps-local CAP parser or projector exists yet.
- EDXL beyond CAP is not scoped for Phase 1.
- NWS is a useful public source, but live NWS calls should not be required for deterministic CI.
- CAP conformance should be stated as schema/consumer-rule evidence until we implement a proper consumer profile.

## Adversarial Feed-Entry Questions

- Are we treating CAP as advisory evidence instead of authoritative track or asset truth?
- Does the parser preserve enough area/resource structure for provenance and operator inspection?
- Are text-heavy fields indexed as `content` without poll/replay noise leaking into semantic indexing?
- Are update/cancel/expire semantics visible in the COP?
- Is live NWS access optional and replayable?

## Source Links

- OASIS CAP 1.2: <https://docs.oasis-open.org/emergency/cap/v1.2/CAP-v1.2-os.pdf>
- NWS API: <https://www.weather.gov/documentation/services-web-api>
