# Weather Projector Component Review

Date: 2026-06-23

Scope: weather graph writer and SemStreams point-forecast projector component.

## Decision

Accept the weather graph writer and point-forecast projector component as the next tactical-weather evidence gate.

The slice is valuable because it moves weather from parser/component preflight into the same SemStreams graph request
shape used by the other governed feeds: decoded forecast BaseMessages in, bounded `weather_observation` mutation plans
out, declared graph request ports, owner-token-fenced writes, and born-first create-conflict reconciliation.

This is not live provider support, not hosted runtime wiring, not OGC EDR conformance, not spatial runtime payload
support, not a weather UI layer, and not route-safety authority.

## Adversarial Notes

### High: Weather cardinality can explode quietly

Point forecasts already multiply samples by variables. Area, corridor, route, ensemble, grid, or high-frequency
provider payloads can turn into thousands of graph mutations if accepted without gates.

Mitigation:

- Keep `max_observations` as a component config gate and reject oversized decoded payloads before graph writes.
- Treat broader OGC EDR query shapes and gridded products as separate runtime-payload reviews.
- Keep raw provider responses in bounded replay/storage lanes instead of expanding every provider dimension by
  default.

### High: Freshness is evidence metadata, not routing truth

The component can stamp `fresh_until`, but that does not prove a provider stale policy, source reliability, or route
safety decision.

Mitigation:

- Keep live provider cadence, cache, stale-source health, and rate-limit behavior as separate gates.
- Do not let route scoring consume weather until model time, valid time, source, confidence, and stale behavior are
  visible to operators.

### Medium: Spatial parser evidence is not spatial runtime projection

The projector can plan observations from spatial forecasts, but the current component consumes decoded point forecast
payloads.

Mitigation:

- Keep OGC EDR area, trajectory, and corridor payload promotion open until payload schemas, UI semantics, and route
  use cases are reviewed.
- Do not claim route-weather support from point-payload graph writes.

### Medium: Reconciliation hides only one class of restart issue

Born-first reconciliation handles create conflicts when SemStreams reports an entity already exists. It does not solve
identity churn, provider duplicate semantics, stale owner tokens, or cross-source weather association.

Mitigation:

- Keep entity IDs deterministic from provider/query-shape/geometry/valid-time/variable.
- Continue using SemStreams-minted owner tokens in runtime composition.
- Add live graph smoke only when a hosted weather runtime is intentionally enabled.

## Verification

- `go test ./internal/projectors/weather`
- `go test ./internal/components/weather`
- `go test ./internal/contracts`
- `go test ./...`

## Follow-Ups

- Decide hosted weather runtime shape: SemStreams HTTP poller component versus a SemOps weather gateway.
- Add spatial decoded payload schemas before projecting area, trajectory, or corridor forecasts in runtime.
- Add COP API/UI weather readback only after operator semantics and stale/freshness labels are accepted.
