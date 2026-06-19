# COP UI Stack And Experience Baseline

## Status

Starting point for the SemOps COP UI, recorded on 2026-06-17. First UI/API/Caddy implementation slice added on
2026-06-19.

This is a product baseline, not a final design. It captures the first implementation direction and the scope gates for
ideas that are promising but risky for a Phase 1 demo.

The current implementation is intentionally narrow:

- `ui` contains the clean-sheet Svelte 5/SvelteKit COP surface.
- `internal/api/cop` exposes `GET /api/cop/snapshot` through a graph-backed provider for configured MAVLink systems
  and TAK/CoT seed UIDs, with the fixture provider retained only as a development fallback before live graph state
  exists.
- `compose.cop.yml` runs the UI behind Caddy so `/api/*` is same-origin with the operator surface.
- The UI renders a MapLibre GL JS canvas with deck.gl tactical overlays for tracks, assets, TAK/CoT tasks,
  TAK/CoT advisories, hazards, labels, and picking, plus alert, feed state, and provenance panels.

This is the first full-stack spine, not the final map implementation. Bounded deltas, real basemap/terrain sources,
footprints, alert geometry, index-backed CoT discovery, and scenario playback remain next gates.

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
the API and graph spine are still moving. deck.gl currently owns point, polygon, label, and picking overlays for the
snapshot's tracks, assets, TAK/CoT tasks, TAK/CoT advisories, and hazard areas. That proves the rendering integration
and selection path, not a finished cartographic basemap, temporal trail layer, alert geometry model, or full tasking
workflow. Vite pins deck/luma/math/probe packages into a single renderer chunk to avoid luma's circular re-export
warning in production builds; the remaining large renderer chunks are accepted while the first screen is inherently
map-first.

## Browser Contract

The browser should not connect directly to NATS in Phase 1. SemOps API owns the browser contract and should expose:

- a snapshot endpoint for current COP state;
- a delta stream using WebSocket, SSE, or GraphQL subscriptions after the API contract decision is made;
- bounded view models for tracks, assets, hazard areas, footprints, alerts, tasks, advisories, feed health,
  provenance, source evidence, and timeline state.

The UI consumes curated COP view models. Native packets, raw frames, SemStreams graph mutation details, and high-rate
trace events stay behind the SemOps API unless an operator workflow proves they need a visible diagnostic lens.

The first live snapshot provider queries SemStreams `graph.query.entity` for configured MAVLink source asset/track IDs
and known TAK/CoT seed UIDs. It maps owned triples into the COP view model and uses SemStreams classified query errors
when available so not-found graph state is handled deliberately instead of being mis-decoded as success. The fixture
snapshot remains a fallback for local development and cold-start demos; it is not graph-compliance evidence. Seed UID
readback is the current proof path; index-backed discovery is a product-grade follow-up.

In local development, Caddy is the browser-facing entrypoint. It serves the Svelte UI and proxies `/api/*` plus
`/healthz` to SemOps API so CORS behavior matches the expected deployment shape. The direct API port stays exposed for
diagnostics and smoke tests.

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
- Topology, tier, or orchestration panels without an adversarial review proving the operator job they improve.

The short version: ontology hydrates the inspector; SemOps owns the view.

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
