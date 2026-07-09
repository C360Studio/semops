# MapLibre/deck.gl Layer Adversarial Review

Date: 2026-06-19

## Decision

Accept the first MapLibre/deck.gl tactical layer as the Phase 1 map-rendering path for tracks, assets, and hazards.
Do not treat it as completion of the full COP geospatial surface.

## Evidence Checked

- `ui/src/lib/cop/TacticalMap.svelte` mounts MapLibre only in the browser, adds a deck.gl `MapboxOverlay`, and renders
  point, polygon, and label layers from the curated snapshot model.
- `ui/src/lib/cop/mapLayers.ts` keeps entity-to-layer projection pure and covered by `mapLayers.test.ts`.
- The route-level placeholder coordinate math is removed from `ui/src/routes/+page.svelte`; selection now flows through
  deck picking or icon buttons.
- `ui/vite.config.ts` keeps deck/luma/math/probe renderer dependencies in one production chunk to avoid the luma
  circular re-export warning seen in the first build.
- Browser smoke at `http://localhost:5174/` rendered two canvases, visible labels, selectable entities, no map load
  error, and no 390px horizontal overflow.

## Objections

- The MapLibre style is intentionally local and empty. It proves renderer integration without proving basemap tile,
  terrain, offline cache, attribution, or airspace/incident cartography policy.
- The deck.gl surface currently renders tracks, assets, hazards, and labels only. Footprints, alert geometry, tasks,
  trails, and temporal scrubbing remain unproved.
- The first fixture places an asset and track at the same coordinate. Label offsets make it readable, but denser real
  feeds will need clustering, decluttering, priority rules, and selection-state styling.
- The production build still reports large map renderer chunks. The circular execution-order warning is fixed, but we
  should not ignore load-time cost once the first viewport grows.
- Browser evidence covered visual runtime and responsive layout, not keyboard-only map selection or screen-reader
  navigation.

## Accepted Risks

- Keep the local no-tile style until SemOps chooses an online/offline basemap source and cache policy.
- Keep icon selection controls as the accessible fallback for deck picking while richer e2e accessibility checks are
  added.
- Accept the renderer chunk size for now because the first screen is map-first and the map packages are dynamically
  imported after mount.

## Follow-Up Tasks

- Add footprint, alert geometry, task, and trail layers behind their own tests and browser evidence.
- Add keyboard/accessibility e2e coverage for selection, inspector updates, source state, and provenance fields.
- Decide tile source/cache policy before adding external basemap dependencies to Caddy or Compose.
- Revisit renderer chunking and lazy-load strategy after the first live mixed-feed scenario is visible.
