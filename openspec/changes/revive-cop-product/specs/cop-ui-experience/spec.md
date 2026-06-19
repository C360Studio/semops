## ADDED Requirements

### Requirement: COP UI uses a product-owned map-first stack

SemOps SHALL build the first operator UI as a Svelte 5/SvelteKit COP surface using MapLibre GL JS for the basemap and
deck.gl for high-rate tactical overlays.

#### Scenario: First screen is the usable COP

- **WHEN** the operator opens SemOps UI
- **THEN** the first screen shows the operating picture with tracks, assets, hazards, footprints, alerts, feed health,
  source state, and provenance access rather than a landing page or flow-runtime dashboard

#### Scenario: Map stack is explicit

- **WHEN** SemOps implements tactical geospatial layers
- **THEN** MapLibre owns the basemap/camera/terrain/label surface and deck.gl owns high-rate overlays, picking,
  filtering, trails, polygons, footprints, and temporal visualization

#### Scenario: First tactical map layer is bounded

- **WHEN** the first MapLibre/deck.gl implementation is present
- **THEN** it renders snapshot tracks, assets, TAK/CoT tasks, TAK/CoT advisories, and hazard areas through deck.gl
  point, polygon, label, and picking overlays
- **AND** it does not claim finished basemap tiles, terrain, track trails, footprints, alert geometry, full task
  workflow geometry, or temporal scrubbing until those layers have their own evidence

### Requirement: Browser state comes through SemOps API

The browser SHALL consume curated SemOps COP view models instead of connecting directly to NATS or raw feed subjects in
Phase 1.

#### Scenario: UI consumes snapshot and delta contracts

- **WHEN** the UI loads or receives live updates
- **THEN** it reads a SemOps API snapshot plus a bounded delta stream for tracks, assets, hazards, footprints, alerts,
  tasks, advisories, feed health, provenance, source evidence, and timeline state

#### Scenario: First snapshot path tolerates backend absence

- **WHEN** the first Svelte COP surface cannot reach the SemOps snapshot API during local development
- **THEN** it renders the fixture snapshot and marks the source as fixture rather than presenting an empty COP
- **AND** the fixture path remains a development fallback, not evidence that the graph-backed COP contract is complete

#### Scenario: First live snapshot path reads governed graph state

- **WHEN** configured MAVLink source asset/track entities, TAK/CoT seed UID entities, or CAP hazard-evidence entities
  exist in SemStreams
- **THEN** `GET /api/cop/snapshot` maps their governed triples into the COP track, asset, task, advisory, hazard,
  feed-health, freshness, confidence, and provenance view model
- **AND** graph query not-found responses are handled as cold-start state rather than silently decoded as successful
  entity data

#### Scenario: Native trace stays behind the API

- **WHEN** a feed emits native packets, raw frames, graph mutations, or replay trace events
- **THEN** those details stay behind SemOps API unless a specific operator or diagnostic workflow exposes them through
  a deliberate lens

### Requirement: Ontology hydrates inspectors, not the whole UI

SemOps MUST NOT treat dynamic ontology-generated UI as a Phase 1 feature.

#### Scenario: Product-owned views remain static

- **WHEN** new predicates, entity classes, or source-specific properties appear
- **THEN** SemOps may use metadata to populate inspector fields, provenance explanations, legends, filters, and
  confidence/freshness badges without automatically creating new operator layers or workflows

#### Scenario: Dynamic layer population is allowed

- **WHEN** feeds, scenarios, filters, or time windows change
- **THEN** existing product-owned layer types may dynamically populate, hide, style, or animate data according to the
  curated COP view model

#### Scenario: Dynamic UI requires a future review

- **WHEN** a proposal wants ontology metadata to create layouts, workflows, alerting behavior, command controls, or
  new operator layer types automatically
- **THEN** the proposal is deferred until an adversarial UX review proves operator value and failure behavior

### Requirement: 3D detail is gated by operator value

SemOps SHALL treat Threlte/Three.js as a detail-inspection surface, not as the default COP renderer.

#### Scenario: 3D detail is selected-entity focused

- **WHEN** the UI uses Threlte or custom Three.js scenes
- **THEN** the scene is tied to a selected platform, sensor, frustum, payload, or similar inspection workflow that a
  2D/2.5D map layer cannot answer clearly

#### Scenario: 3D spectacle is rejected

- **WHEN** 3D terrain, models, or camera effects make stale state, provenance, alerts, or source confidence harder to
  read
- **THEN** the UI favors simpler map layers and operator state over visual novelty
