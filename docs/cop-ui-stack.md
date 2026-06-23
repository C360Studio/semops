# COP UI Stack And Experience Baseline

## Status

Starting point for the SemOps COP UI, recorded on 2026-06-17. First UI/API/Caddy implementation slice added on
2026-06-19.

This is a product baseline, not a final design. It captures the first implementation direction and the scope gates for
ideas that are promising but risky for a Phase 1 demo.

The current implementation is intentionally narrow:

- `ui` contains the clean-sheet Svelte 5/SvelteKit COP surface.
- `internal/api/cop` exposes `GET /api/cop/snapshot` through a graph-backed provider that discovers MAVLink,
  TAK/CoT, and CAP entities by SemStreams prefix query, with configured seed IDs retained only as family-scoped
  compatibility fallback.
- `internal/api/cop` also exposes `GET /api/cop/runtime`, a read-only runtime view derived from running SemStreams
  component health and flow sources.
- `compose.cop.yml` runs the UI behind Caddy so `/api/*` is same-origin with the operator surface.
- The UI renders a MapLibre GL JS canvas with deck.gl tactical overlays for tracks, assets, TAK/CoT tasks,
  TAK/CoT advisories, hazards, KLV sensor/frame-center rays, labels, and picking, plus alert, feed state, runtime
  flow, and provenance panels.

This is the first full-stack spine, not the final map implementation. Bounded deltas, real basemap/terrain sources,
footprint polygons, alert geometry, discovery total-count tuning, and scenario playback remain next gates.

## Direction

SemOps should build a clean-sheet operator COP, not restore the old flow-runtime UI and not ship a generic dashboard.
The first screen should answer:

- What is happening?
- What changed?
- What is stale?
- What source said it?
- What needs action?

The UI should be map-first, source-aware, and provenance-rich. It should reduce cognitive load rather than expose every
feed, predicate, service, and inference tier at once.

## Starting Stack

| Layer | Starting choice | Role |
| --- | --- | --- |
| App shell | Svelte 5 / SvelteKit | Product shell, panels, stores, API subscriptions, tests |
| Basemap | MapLibre GL JS | Open WebGL vector basemap, camera, terrain, labels, map controls |
| Tactical overlays | deck.gl | High-rate tracks, points, polygons, footprints, picking, filtering, temporal layers |
| Format loaders | loaders.gl | Optional parser helpers for GeoJSON, WKT/WKB, glTF, 3D Tiles, and imagery |
| Detail 3D | Threlte | Gated 3D detail views for selected platforms, sensors, frustums, or payload previews |

MapLibre plus deck.gl is the default COP map path. Threlte is not the default tactical map renderer; use it only when a
selected entity needs a richer 3D inspection surface than a map symbol, footprint, or trail can provide.

The first implemented map layer uses a local empty MapLibre style so the demo does not depend on external tiles while
the API and graph spine are still moving. deck.gl currently owns point, polygon, line, label, and picking overlays for
the snapshot's tracks, assets, TAK/CoT tasks, TAK/CoT advisories, hazard areas, and KLV sensor/frame-center evidence.
Selecting an alert can highlight the referenced map entity when `entity_id` points at a track, asset, task, advisory,
hazard, or sensor footprint; the alert itself remains the selected inspector object. That proves the rendering and
selection path, not a finished cartographic basemap, temporal trail layer, independent alert geometry model, full
footprint polygon, or full tasking workflow. Vite pins deck/luma/math/probe packages into a single renderer chunk to
avoid luma's circular re-export warning in production builds; the remaining large renderer chunks are accepted while
the first screen is inherently map-first.

## Browser Contract

The browser should not connect directly to NATS in Phase 1. SemOps API owns the browser contract and should expose:

- a snapshot endpoint for current COP state;
- a runtime endpoint for component-derived source health and flow;
- a delta stream using WebSocket, SSE, or GraphQL subscriptions after the API contract decision is made;
- bounded view models for tracks, assets, hazard areas, footprints, alerts, tasks, advisories, feed health,
  provenance, source evidence, and timeline state.

The UI consumes curated COP view models. Native packets, raw frames, SemStreams graph mutation details, and high-rate
trace events stay behind the SemOps API unless an operator workflow proves they need a visible diagnostic lens.

The first live snapshot provider prefers SemStreams `graph.query.prefix` over seed-only point lookups. It discovers
MAVLink, TAK/CoT, and CAP entities by their 5-part COP prefixes, maps owned triples into the COP view model, and keeps
seeded `graph.query.entity` reads as family-scoped compatibility fallback when prefix discovery is disabled,
unavailable, or empty for that feed family. CoT/CAP seed IDs are optional in the normal Compose path when discovery is
enabled. That makes SemStreams responsible for graph discovery while SemOps owns the curated operator view. The fixture
snapshot remains a fallback for local development and cold-start demos; it is not graph-compliance evidence. CAP hazard
lifecycle status is derived in the view model from advisory evidence and freshness; distinct expired/cancelled/stale map
symbology is a future UX gate.

The snapshot uses SemStreams typed `graph.PrefixQueryRequest`/`graph.PrefixQueryResponse` envelopes and follows opaque
`NextCursor` values until a source/type prefix is exhausted or the configured SemOps discovery cap is reached. Its
diagnostics report org, platform, source, entity type, returned count, query limit, cap-truncated pressure, and
prefix-query error text when a partial read fails. The UI surfaces those counts compactly in the source cards and
promotes truncation/error diagnostics into warning alerts so large mixed-feed demos can show index-pressure evidence
without exposing raw graph triples as an operator workflow.

The COP API also exposes graph-backed weather observations for source/provenance evidence and stack smoke readback.
That is not yet a tactical weather map layer: visual weather tiles, route-weather semantics, stale/cache policy, and
operator-facing legends remain separate product gates.

`GET /api/cop/runtime` rolls up SemStreams component `Health()` and `DataFlow()` into feed-level status,
throughput, healthy component counts, and last activity. The source cards merge this runtime evidence with snapshot
feed state, so the UI can show whether a hosted MAVLink, TAK/CoT, CAP, ADS-B, SAPIENT, or KLV component flow is
active, idle, stale, or degraded. Prometheus remains the operational metrics standard; the browser runtime endpoint is
a curated product view and should not grow into a topology editor or orchestration shell.

In local development, Caddy is the browser-facing entrypoint. It serves the Svelte UI and proxies `/api/*` plus
`/healthz` to SemOps API so CORS behavior matches the expected deployment shape. The direct API port stays exposed for
diagnostics and smoke tests.

The browser e2e gate is fixture-backed Playwright coverage in `ui/e2e`. It intercepts `GET /api/cop/snapshot` and
`GET /api/cop/runtime`, serves API-shaped ADS-B discovery plus runtime-flow evidence, and verifies the operator
surface renders source cards, prefix-discovery counts, runtime flow, map selection controls, keyboard selection,
selected-entity provenance, alert-to-map target highlighting, and a narrow viewport without horizontal overflow. This
complements the Docker stack smoke: Playwright proves the browser contract and interaction path, while
`scripts/cop-stack-smoke.sh` proves the live SemOps/SemStreams/Caddy plumbing.

## KLV Sensor-Footprint UI Gate

The first KLV product-visible slice proves binary-derived evidence through the graph and COP API, not by showing a raw
video file in the browser. The current KLV projector contract can write source-partitioned `sensor_footprint` state for
sensor position, frame center, azimuth/elevation, media reference, packet reference, platform designation, and
provenance. `GET /api/cop/snapshot` now reads that governed state back before any richer media surfaces exist.

The first visible layer should include:

- a sensor-position point;
- a frame-center point;
- a ray or line between the sensor and frame center;
- selected-entity provenance with frame time, observed time, platform designation, decoded field inventory, warnings,
  media reference, packet reference, source hash/provenance when available, and component-flow status.

This is the "eye candy with teeth" gate: the operator sees something spatial and selectable, but every visible detail
is tied back to governed graph state, packet/media references, and the validation ladder. Public sample evidence should
be labeled as smoke only. Deterministic fixtures may support engineering-support language only for the tested MISB ST
0601 subset.

The implemented UI renders the sensor-position point, frame-center point, and ray as a selectable deck.gl layer. The
selected inspector shows KLV evidence, media reference, packet reference, decoded-field inventory, warning evidence,
claim posture, and provenance. Playwright covers the selector, inspector, source card, runtime flow, and narrow
viewport path. The one-command Docker smoke can additionally opt into the hosted KLV local-media flow with
`SEMOPS_COP_SMOKE_KLV_ENABLED=true`, proving generated deterministic MPEG-TS media through the SemStreams component
chain and Caddy-routed COP readback without enabling KLV in the default stack.

Do not add a video player, thumbnail strip, 3D frustum, footprint polygon, or STANAG 4609 conformance language as part
of this gate. Those remain separate gates because each adds a different failure mode: media serving and cache policy,
operator attention load, footprint computation policy, and formal standards evidence.

## Dynamic UI Scope Gate

Dynamic ontology-generated UI is a research idea, not a Phase 1 feature.

Accepted Phase 1 behavior:

- Product-owned views and layer types are statically designed.
- Layer population is dynamic based on feed capabilities, scenario state, filters, time window, and source health.
- Ontology and projection metadata hydrate inspector fields, provenance explanations, filter labels, legends, and
  confidence/freshness badges.
- Unknown or new predicates can appear in a technical evidence panel when useful for debugging.

Deferred behavior:

- Automatic creation of new operator layers because a new predicate or entity class appears.
- Automatic layout, workflow, alerting, or command controls generated from ontology structure.
- Topology, tier, or orchestration panels without an adversarial review proving the operator job they improve. The
  current MVP review explicitly defers them because source health, runtime flow, provenance, alerts, and scenario
  state answer the known operator questions with less visual and control-surface risk.

The short version: ontology hydrates the inspector; SemOps owns the view.

Graph/source visualization prior art should come from C360's existing stack before SemOps invents a new surface.
SemConnect and SemLink both contain useful graph-lens patterns for source-aware inspection and standards bridge
debugging. SemOps should reuse or adapt those ideas when a graph visualization answers an operator or diagnostic
question that the tactical map cannot answer cleanly.

## UX Principles

- Prefer 2D/2.5D map clarity before 3D spectacle.
- Make freshness, confidence, and source provenance visible at selection time and in compact layer state.
- Keep raw/replay detail behind drill-down affordances; do not let trace volume become the main experience.
- Represent stale or missing feeds as state, not silence.
- Use timeline controls only when they help compare current state, replay, and expected scenario state.
- Avoid topology/tier controls until source health and provenance views are proven insufficient.

## External References

- MapLibre GL JS: https://maplibre.org/maplibre-gl-js/docs/
- deck.gl: https://deck.gl/docs
- deck.gl Mapbox/MapLibre overlay: https://deck.gl/docs/api-reference/mapbox/mapbox-overlay
- loaders.gl: https://loaders.gl/docs
- loaders.gl 3D Tiles notes: https://loaders.gl/docs/modules/3d-tiles
- Threlte: https://threlte.xyz/docs/learn/getting-started/introduction/
