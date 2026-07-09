# Weather Graph Contract Review

Date: 2026-06-23

Scope: tactical-weather `weather_observation` ownership contract and pure projector-plan gate.

## Decision

Accept `weather_observation` as the first governed graph shape for tactical weather.

The contract projects each bounded weather variable/time sample as source-partitioned `signal` evidence with provider,
query shape, query geometry, valid time, model time, freshness, value, unit, confidence, and source reference. It does
not own CAP hazard authority, route decisions, task state, alert state, or advisory content. The current slice is a
pure mutation-plan gate only; no graph writer, runtime component, UI layer, live provider, or route-safety rule is
enabled.

## Adversarial Notes

### High: Cardinality can explode

One entity per variable/time sample preserves tuple integrity, but it can become dangerous if SemOps ingests global
grids, long forecast horizons, or unbounded route sampling.

Mitigation:

- Keep tactical weather queries localized to active assets, incident areas, or planned routes.
- Require caps on query geometry count, forecast horizon, variables, and refresh cadence before runtime wiring.
- Use SemStreams `signal` indexing and monitor profile/cardinality pressure before UI promotion.

### High: Weather evidence can become decision authority by accident

Wind, visibility, precipitation, and pressure are operationally tempting. The graph contract publishes evidence only;
it does not recommend routes, ground drones, or assign tasks.

Mitigation:

- Keep route/weather decision state in a later `control` contract.
- Require explicit freshness, confidence, model-time, source, and operator-facing caveats before route/safety logic
  consumes weather observations.

### Medium: CAP hazard authority remains separate

Weather warnings and CAP alerts may describe hazards, but tactical weather observations must not overwrite CAP hazard
evidence or stricter hazard state.

Mitigation:

- Keep CAP/weather warning text on append-evidence or advisory contracts.
- Keep `weather_observation` free of `cop.hazard.*`, `cop.alert.*`, and `cop.task.*` predicates.

### Medium: Runtime binding is intentionally absent

The contract and pure projector prove the graph shape, but no owner is bound in the default runtime and no graph
writer publishes weather observations yet.

Mitigation:

- Add runtime owner binding, graph writer, cache/stale policy, and component telemetry as a separate gate.
- Do not claim tactical-weather UI, live provider support, or route-weather behavior from this slice.

## Verification

- `go test ./pkg/cop`
- `go test ./internal/projectors/weather`
- `go test ./...`

## Follow-Ups

- Add a weather graph writer and SemStreams projector component only after runtime caps and stale policy are accepted.
- Add UI readback after graph-backed weather observations exist.
- Add route/weather decision contracts separately as `control` state.
