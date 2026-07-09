# Weather Stack Readback Review

Scope: graph-backed weather observation COP API readback and opt-in Caddy-routed stack smoke.

## Decision

Accept weather readback as fixture-backed stack evidence. The COP snapshot can discover `weather_observation` entities
by prefix, expose provider, variable/value/unit, query geometry, model/valid/freshness time, provenance, and claim
posture, and the one-command smoke can opt into the check with `SEMOPS_COP_SMOKE_WEATHER_ENABLED=true`.

This remains source/provenance evidence only. It does not add a live weather provider, cache/stale policy,
tactical-weather map layer, weather-routing authority, or OGC conformance claim.

## Findings

### High: API readback can be mistaken for tactical weather UX

Exposing weather observations in the COP snapshot is useful evidence, but it is not yet an operator layer. A map layer
needs source selection, freshness semantics, legend/range design, and route-safety copy before it belongs in the
operator surface.

Resolution: expose `weather_observations` for API/smoke evidence only and keep UI semantics deferred.

### High: Freshness must use weather metadata, not generic feed age only

Weather observations carry `fresh_until`. A model timestamp can be older than the generic feed freshness window while
the forecast is still valid for the local fixture proof.

Resolution: feed health treats weather as live when at least one observation is still `fresh` according to
weather-specific metadata.

### Medium: Stack smoke remains opt-in

The smoke proves local fixture graph projection through Caddy, but enabling it by default would make the Phase 1 stack
look like it includes a weather service.

Resolution: keep `SEMOPS_COP_SMOKE_WEATHER_ENABLED=false` unless explicitly requested.

## Verification

- `go test ./internal/api/cop ./internal/smoke/cop`
- `npm test -- --run src/lib/cop/sourceHealth.test.ts src/lib/cop/client.test.ts src/lib/cop/selection.test.ts src/lib/cop/mapLayers.test.ts`
- `SEMOPS_COP_SMOKE_WEATHER_ENABLED=true bash scripts/cop-stack-smoke.sh`

## Follow-Ups

- Add a tactical weather layer only after operator semantics, source/freshness policy, and visual encoding are reviewed.
- Add live provider polling only after cache, stale-source, rate-limit, and replay policy are accepted.
