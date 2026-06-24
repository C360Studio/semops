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
- **THEN** it renders snapshot tracks, assets, TAK/CoT tasks, TAK/CoT advisories, hazard areas, and KLV
  sensor/frame-center footprints, plus localized weather observation points, through deck.gl point, polygon, line,
  label, and picking overlays
- **AND** when an alert references an existing spatial entity, selecting the alert highlights the referenced map
  geometry while keeping the alert itself selected in the inspector
- **AND** it does not claim finished basemap tiles, terrain, track trails, an independent alert geometry model, full
  footprint polygons, full task workflow geometry, or temporal scrubbing until those layers have their own evidence

### Requirement: Core operator loop is test guarded

SemOps SHALL keep the first COP operator loop covered by focused component/helper tests and browser checks.

#### Scenario: Source state has component-level coverage

- **WHEN** source cards combine snapshot feed health, runtime component flow, and prefix-discovery diagnostics
- **THEN** tests cover live, idle, stale, degraded, runtime-only, and truncation states without requiring the full page
  to render
- **AND** runtime evidence does not erase the feed's source/provenance message

#### Scenario: Browser flow remains keyboard and narrow-viewport usable

- **WHEN** Playwright exercises the first COP screen
- **THEN** it verifies named source cards, map entity controls, alert controls, selected-entity provenance, keyboard
  activation, alert-to-map target highlighting, and a narrow viewport without horizontal overflow

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

#### Scenario: First live snapshot path discovers governed graph state

- **WHEN** MAVLink source asset/track entities, TAK/CoT entities, or CAP hazard-evidence entities exist in
  SemStreams under configured COP graph discovery scopes
- **THEN** `GET /api/cop/snapshot` maps their governed triples into the COP track, asset, task, advisory, hazard,
  feed-health, freshness, confidence, and provenance view model
- **AND** the primary read path uses SemStreams prefix discovery before falling back to configured seed entity IDs only
  for feed families with disabled, unavailable, or empty discovery
- **AND** CoT/CAP snapshot state does not require configured seed UID or alert-ID lists when graph discovery is enabled
- **AND** the snapshot follows typed SemStreams prefix cursors until each source/type prefix is exhausted or the
  configured SemOps discovery cap is reached
- **AND** the snapshot exposes source/type discovery diagnostics with returned count, query limit, cap-truncated
  pressure, and partial prefix-read error state
- **AND** truncation or error diagnostics become active warning alerts tied to the affected source feed
- **AND** graph query not-found responses are handled as cold-start state rather than silently decoded as successful
  entity data

#### Scenario: Runtime flow is exposed as a curated API view

- **WHEN** SemOps hosts feed input and processor components with SemStreams `Health()` and `DataFlow()` surfaces
- **THEN** `GET /api/cop/runtime` maps those components into feed-level runtime status, throughput, health counts,
  last activity, and component evidence
- **AND** the browser consumes that curated view instead of scraping Prometheus, connecting to NATS, or inferring raw
  subject topology
- **AND** an unavailable runtime view does not replace the snapshot fallback or make the first screen empty

#### Scenario: Native trace stays behind the API

- **WHEN** a feed emits native packets, raw frames, graph mutations, or replay trace events
- **THEN** those details stay behind SemOps API unless a specific operator or diagnostic workflow exposes them through
  a deliberate lens

#### Scenario: KLV sensor-footprint UI proves binary-derived evidence

- **WHEN** KLV decoded-frame state has been projected into governed `sensor_footprint` graph entities
- **THEN** the SemOps COP API exposes a curated sensor-footprint view model with sensor position, frame center,
  sensor-to-frame-center ray geometry, frame time, observed time, confidence, freshness, source, media reference,
  packet reference, and claim posture
- **AND** the browser renders the sensor point, frame-center point, and ray as a selectable tactical layer
- **AND** the selected-entity inspector shows platform designation, decoded-field inventory, warnings, media
  provenance, packet provenance, source hash/provenance when available, and component-flow evidence
- **AND** the UI labels public-sample evidence as smoke only and deterministic fixtures as engineering-support
  evidence for the tested MISB ST 0601 subset
- **AND** the UI does not show a footprint polygon, video player, thumbnail wall, 3D frustum, or STANAG conformance
  claim until those gates have separate graph, media, and review evidence

#### Scenario: Weather observation UI proves localized source evidence

- **WHEN** weather observations have been projected into governed `weather_observation` graph entities
- **THEN** the SemOps COP API exposes a curated weather view model with provider, query shape, query geometry,
  variable/value/unit, valid time, model time, freshness, confidence, provenance, and claim posture
- **AND** the browser renders the localized weather observation as a selectable point evidence layer
- **AND** the selected-entity inspector shows provider, value, query geometry, valid/model/freshness time, provenance,
  and claim posture
- **AND** the UI does not show visual weather tiles, incident-area weather products, route-safety decisions, live
  provider reliability, or OGC conformance claims until those gates have separate evidence and review

#### Scenario: Command lifecycle UI is read-only status evidence

- **WHEN** command-intent task state has been projected into governed `command` task entities
- **THEN** the SemOps COP API discovers those tasks by command prefix and maps latest lifecycle status, target,
  authority, priority, expiry, requested-by, correlation, desired state, provenance, and owner into the curated task
  view model
- **AND** the browser renders command tasks as selectable task rows with selected-entity inspector details and source
  card discovery evidence
- **AND** command tasks without geometry remain inspectable without pretending they are map points
- **AND** the UI does not expose execute, cancel, retry, arbitration override, CS API tasking, or native transmit
  controls until those gates have separate safety and adversarial UX review

#### Scenario: Fusion association UI is evidence, not identity merge

- **WHEN** fusion-owned track association evidence has been projected into governed graph entities
- **THEN** the SemOps COP API discovers association entities by fusion prefix and maps source tracks, confidence,
  status, algorithm identity, distance/time evidence, source references, owner, and claim posture into the curated
  snapshot
- **AND** the browser renders association records as selectable evidence rows and map-selector affordances without
  drawing new merged-track geometry
- **AND** the selected-entity inspector shows primary track, candidate track, algorithm, metrics, reason, provenance,
  and the explicit no-merge/no-identity-authority posture
- **AND** row-level copy and badges use candidate/possible association language when the evidence has not been
  operator-reviewed or promoted by an explicit identity policy
- **AND** the UI does not expose merge, split, override, source-track mutation, or identity-authority controls until
  identity policy, mission thresholds, ambiguity policy, and adversarial operator review prove those workflows are safe

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

#### Scenario: Source cards show bounded discovery evidence

- **WHEN** SemOps renders source health for graph-backed snapshot state
- **THEN** the source cards may show compact prefix-discovery counts by source/type
- **AND** those cards highlight cap-truncated pressure without exposing raw graph triples or native packet payloads

#### Scenario: Source cards show runtime flow evidence

- **WHEN** SemOps renders source cards with runtime component data available
- **THEN** the cards show compact feed-level component status, message rate, healthy component count, and last-flow
  freshness
- **AND** runtime-only preflight feeds such as SAPIENT may appear as component-flow evidence without claiming graph
  projection or product feed support
- **AND** this remains a source-health view, not an orchestration, topology, or flow-control UI

#### Scenario: Discovery pressure becomes source-health alerts

- **WHEN** prefix discovery stops at its configured cap with continuation state available or a prefix read fails
- **THEN** the snapshot emits active warning alerts tied to the affected source feed
- **AND** those alerts explain the pressure or partial read failure without presenting the condition as authoritative
  fusion state

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
