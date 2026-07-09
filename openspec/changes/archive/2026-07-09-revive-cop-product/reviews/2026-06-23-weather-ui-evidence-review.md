# Weather UI Evidence Review

Scope: selectable weather-observation point evidence in the COP UI.

## Decision

Accept a narrow weather UI proof: graph-backed `weather_observation` view models may render as selectable localized
point evidence with source-card discovery/runtime context and inspector provenance. This gives the demo visible
weather evidence without promoting weather into route safety, visual tiles, live-provider support, or standards
conformance.

## Findings

### High: A weather marker can be mistaken for tactical weather product support

Rendering weather on the map is useful, but a single point observation is not a radar layer, incident-area weather
product, route-weather model, or stale/cache policy.

Resolution: the UI labels the selected entity through provider, value, query shape/geometry, freshness, provenance,
and claim posture. Documentation and OpenSpec call this point-observation evidence only.

### High: Weather evidence must keep model and freshness context visible

Weather values are unsafe without time context. An observation value can be stale, model-derived, or scoped to a
single query point even when it looks visually current.

Resolution: the inspector exposes valid time, model time, fresh-until time, updated freshness, source, confidence,
owner, and source reference. The weather marker is selectable evidence, not a decision rule.

### Medium: Fixture fallback should not masquerade as live provider support

The UI fixture now contains weather evidence for browser development and e2e coverage. That fixture is not live
Open-Meteo, OGC EDR, NWS, MSC GeoMet, or tile-service integration.

Resolution: the fixture claim posture explicitly says there is no live provider, weather tile, route-safety, or OGC
conformance claim. Live provider work remains a separate feed/runtime promotion gate.

## Verification

- `npm --prefix ui run test -- --run src/lib/cop/mapLayers.test.ts src/lib/cop/selection.test.ts src/lib/cop/sourceHealth.test.ts src/lib/cop/client.test.ts`
- `npm --prefix ui run check`
- `npm --prefix ui run test:e2e`

## Follow-Up

- Design a visual weather tile/raster layer only after source, cache, license, legend, and operator workflow review.
- Keep route-weather and weather-derived decisions as governed control/fusion state with explicit authority and TTL.
