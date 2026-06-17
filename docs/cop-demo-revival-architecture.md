# SemOps COP Demo Revival Architecture

## Status

Draft baseline for the SemOps revival, created on 2026-06-17.

Inputs:

- Claude's SemOps COP demo plan in `SemOps-COP-Demo-Plan.md`.
- Current SemOps checkout at `/Users/coby/Code/c360/semops`.
- Current SemStreams checkout at `/Users/coby/Code/c360/semstreams`.
- Current SemLink checkout at `/Users/coby/Code/c360/semlink`.
- Feed validation and indexing ladder in `docs/feed-validation-and-indexing-ladder.md`.

## Executive Direction

SemOps should come back as a greenfield data-fusion common operating picture and integration lab. It should be
large, bold, and allowed to break old assumptions. The old SemOps code is not the architecture; it is salvage.

Product boundary:

- SemOps owns the COP product, HA/DR scenario, feed adapters, domain vocabulary, fusion behavior, operator UI,
  scenario playback, and product-scoped governance.
- SemStreams owns the substrate: NATS/JetStream, graph mutation/query contracts, projection contracts,
  ownership claims, indexing profiles, rule processing, and tiered structural/statistical/semantic services.
- SemConnect owns the standards-facing OGC Connected Systems API egress path.
- SemLink remains useful prior art for the modern GCS UI pattern, source-aware graph lens, TAK bridge, CS API bridge,
  and bounded raw telemetry lane. SemLink should stay a basic demo unless explicitly rechartered; SemOps owns the
  complete COP product going forward.

SemOps should act as both consumer and producer for SemStreams improvement. Build product-specific pieces here,
then upstream generic manifest, governance, tiering, indexing, and provenance needs only once the demo proves them.

OpenSpec source: `openspec/changes/revive-cop-product`.

Scope gate: orchestration, topology panels, and tier controls are hypotheses, not accepted Phase 1 features.

Feed gate: feeds are added one at a time. MAVLink and TAK/CoT are first, CAP follows, and KLV stays a proof spike
until video/KLV fixtures prove binary-by-reference and memory-bounded handling.

Review gate: adversarial reviews are required at key stage boundaries. Reviews should challenge product value,
framework ownership, evidence quality, compliance language, index-profile decisions, and demo credibility before the
next implementation tranche begins.

UI gate: the frontend starts as a clean-sheet Svelte 5/SvelteKit COP using MapLibre GL JS for the basemap and deck.gl
for high-rate tactical overlays. Dynamic ontology-generated UI is not a Phase 1 feature. Ontology and projection
metadata should hydrate inspectors, provenance, filters, legends, and confidence/freshness badges; SemOps owns the
operator views.

## Live Repo Findings

SemOps started materially stale; the first revival slices are correcting that:

- SemOps now declares Go `1.26.3` and imports current `github.com/c360studio/semstreams`.
- `go test ./...` passes for the active product compile path.
- `cmd/semops/main.go` is a lifecycle stub with TODOs for configuration, SemStreams clients, adapters, API, and
  monitoring.
- `configs/robotics-flow.json` describes an old StreamKit-style flow and not the current SemStreams graph ingest
  and projection contract surface.
- Old EntityStore, ObjectStore, StreamKit, and BaseProcessor product paths have been removed from the active build.
- The current checkout and reachable Git history contain no frontend tree. The old flow-runtime UI idea should be
  treated as historical context, not a surface to restore.

SemOps has salvageable MAVLink depth:

- It contains a MAVLink v1/v2 parser with registered message specs for heartbeat, global position, attitude, battery,
  COMMAND_LONG, and COMMAND_ACK.
- It contains a test message generator, parser tests, command codec tests, and raw-lane tests.
- Non-reference StreamKit, BaseProcessor-era MAVLink code, and ignored SITL references have been removed after useful
  command encoding and ACK parsing moved into the active adapter.
- A bounded MAVLink raw frame lane now stores copied frames under record and byte caps and annotates decoded packets
  with source references for governed current-state projections.

SemLink has the more current product pattern:

- Raw high-rate MAVLink frames stay on a bounded stream lane.
- Current vehicle state is collapsed into one signal-profiled graph entity per vehicle.
- Alerts and commands are control-profiled graph entities.
- Projection contracts declare ownership and indexing profiles before writing through SemStreams graph mutation
  subjects.
- A Svelte 5 dashboard and CS API bridge already prove the operator and standards-projection shape.

## COP UI Baseline

The starting UI stack is recorded in `docs/cop-ui-stack.md` and
`openspec/changes/revive-cop-product/specs/cop-ui-experience/spec.md`.

The product direction is:

- Svelte 5/SvelteKit for the product shell, panels, stores, subscriptions, and component tests.
- MapLibre GL JS for the open WebGL basemap, camera, terrain, labels, and map controls.
- deck.gl for tactical overlays: high-rate tracks, trails, hazards, footprints, picking, filtering, and temporal
  layers.
- loaders.gl as an optional parser helper for formats such as GeoJSON, WKT/WKB, glTF, 3D Tiles, and imagery.
- Threlte only for selected-entity 3D detail views where a 2D/2.5D map layer is insufficient.

The browser should consume SemOps API snapshots and bounded deltas, not connect directly to NATS in Phase 1. Native
packets, raw frames, graph mutation detail, and replay trace events stay behind SemOps API unless a deliberate
operator or diagnostic lens exposes them.

Dynamic UI is scoped narrowly:

- Accepted: dynamic population, styling, filtering, and timeline behavior inside product-owned layer types.
- Accepted: ontology and projection metadata in inspector fields, provenance explanations, legends, filters, and
  confidence/freshness badges.
- Deferred: automatic operator layouts, workflows, alerting, command controls, or new map layers generated from
  ontology structure.

The short rule is: ontology hydrates the inspector; SemOps owns the view.

## Design Principles

1. Raw feeds stay raw at the boundary. The graph gets current state, durable events, provenance, confidence, and
   relationship evidence, not one entity per packet.
2. Every adapter writes through SemStreams ADR-055/056 projection and ownership contracts. Entity birth is explicit,
   foreign edges derive `ForeignEdgeClaim` records, and no feed silently clobbers another feed's predicates.
3. Loose feeds use tolerant readers at the boundary and strict governed writes into the graph.
4. Structural is the default operating mode. Statistical and semantic inference are recorded as evidence with
   visible justification before they become any kind of control surface.
5. Container boundaries follow deployment concerns: independent placement, scaling, external network protocols,
   secrets, expensive inference, or different failure domains.
6. Library/component boundaries follow code reuse concerns: codecs, canonical mappers, entity models, vocabulary,
   projection contracts, and deterministic fusion rules.
7. SemOps can evaluate tier-placement and escalation behavior, but only after a concrete operator-value case exists.
8. Adversarial reviews are part of the delivery plan. A stage is not ready because it is plausible; it is ready after
   architect, reviewer, and technical-writer roles have tried to break the assumptions and recorded the result.

## Born-First Graph Discipline

SemOps accepts the SemStreams breaking-change direction before rebuilding feed adapters:

- New entities are born with `graph.CreateEntityWithTriplesRequest`, `MessageType`, and `IndexingProfile`.
- Current-state changes use `graph.UpdateEntityWithTriplesRequest` against entities that already exist.
- Adapters must not rely on `triple.add` or `triple.add_batch` auto-vivify.
- Cross-entity relationships must be declared in projection contracts so SemStreams derives
  `ownership.ForeignEdgeClaim` values.
- MAVLink and TAK `cop.track.source` edges are strict born-first edges. The source `asset` entity must exist before
  the track edge is written.
- `EdgeNoBirthStub` is a reviewed exception for targets that have no independent producer, not a general fallback.

## Adversarial Review Gates

Run adversarial reviews before:

- Modernizing the SemStreams contract, to catch old StreamKit assumptions and accidental framework drift.
- Declaring the COP entity/predicate model stable, to catch ownership conflicts, born-first gaps, and product-only
  vocabulary.
- Adding each Phase 1 feed to the stack, to catch missing parser, replay, compliance, and indexing-profile evidence.
- Promoting orchestration, topology, or tier UI, to prove operator value and avoid building a footgun.
- Starting SAPIENT or KLV product work, to verify authoritative fixtures and honest compliance/binary claims.
- Filing upstream SemStreams issues, to separate product-specific pressure from reusable framework requirements.

Each review should leave a short record: decision, objections, evidence checked, accepted risks, and follow-up tasks.

## System View

```mermaid
flowchart LR
    subgraph Edge["Edge node"]
        MAV["MAVLink adapter"]
        COT["TAK/CoT adapter"]
        CAP["CAP/EDXL adapter"]
        RAW["Bounded raw lanes"]
        PROJ["Structural projectors"]
    end

    subgraph Core["Core node"]
        SS["SemStreams structural graph"]
        RULES["Rules and governance"]
        ASSOC["Track association evidence"]
        SEM["Semantic translation evidence"]
        API["SemOps COP API"]
        UI["SemOps Svelte COP"]
        CS["SemConnect CS API egress"]
    end

    subgraph Ops["Operations"]
        MAN["Deployment metadata"]
        POL["Escalation evidence"]
        OBS["Prometheus/Otel/logs"]
    end

    MAV --> RAW
    COT --> RAW
    CAP --> RAW
    RAW --> PROJ
    PROJ --> SS
    RULES --> SS
    SS --> API
    API --> UI
    SS --> CS
    MAN --> API
    POL --> API
    API --> ASSOC
    API --> SEM
    ASSOC --> SS
    SEM --> SS
    SS --> OBS
    API --> OBS
```

## Containerized Services

Use services where deployment isolation matters. The first structural demo should start with a single Compose
stack, then split edge/core only after the deployment metadata has real value.

| Service | Owner | Why it is a service | First phase |
| --- | --- | --- | --- |
| `nats` | Infra | Durable streams, KV buckets, request/reply, observability port | Phase 0 |
| `semstreams-structural` | SemStreams | Graph ingest, graph query, rule processor, structural indexes | Phase 0 |
| `semops-api` | SemOps | COP snapshot API, SSE, commands, source/provenance views | Phase 1 |
| `semops-ui` | SemOps | Svelte COP product surface; may be served by `semops-api` | Phase 1 |
| `semops-scenario-runner` | SemOps | Scripted HA/DR feed playback, deterministic demo clock | Phase 1 |
| `semops-adapter-mavlink` | SemOps | External UDP/TCP/SITL boundary and high-rate raw lane producer | Phase 1 |
| `semops-adapter-cot` | SemOps | TAK UDP/TCP/XML boundary and operator/marker/message projection | Phase 1 |
| `semops-adapter-cap` | SemOps | Tolerant CAP/EDXL reader and hazard/advisory projection | Phase 1 |
| `semops-adapter-sapient` | SemOps | Protobuf boundary and strict detection/track projection | Phase 2 |
| `semops-adapter-adsb` | SemOps | Air-track source, raw JSON first, ASTERIX later | Phase 2 |
| `semops-adapter-klv` | SemOps | Video metadata/footprint extraction from STANAG 4609 KLV subset | Phase 3 |
| `semops-track-association` | SemOps | Statistical tier for ambiguous cross-source track association | Phase 2 |
| `semops-translation-agent` | SemOps | Semantic tier for civilian advisory translation and explanations | Phase 3 |
| `semconnect-csapi` | SemConnect | Standards egress and conformance surface | Phase 3 |
| `observability` | Infra | Prometheus/Otel/log aggregation for active demo monitoring | Phase 0 |

Do not make each deterministic mapper its own service by default. A mapper becomes a service only when it owns an
external protocol boundary, needs separate placement, or has a different failure/scaling profile.

## SemOps Components

These belong inside the SemOps codebase even when a container hosts them.

| Component | Role | Notes |
| --- | --- | --- |
| `pkg/adapters/mavlink` | MAVLink codec, raw lane, and command helpers | Active parser/generator extracted |
| `pkg/cop` | COP model, predicates, projection contracts | Track, alert, asset, hazard, footprint, task, advisory |
| `internal/projectors/mavlink` | Decoded MAVLink packets to graph mutation plans | Born-first current-state planner |
| `internal/projectors/*` | Boundary payload to graph projection mappers | One projection owner per feed or flow |
| `internal/fusion` | Structural fusion and deterministic correlation | Geofence, dedupe, stable-ID match, warnings |
| `internal/deployment` | Deployment metadata and health state | Build only after operator-value review |
| `internal/inference` | Inference evidence and transition records | Evidence first, UI later |
| `internal/scenario` | HA/DR scripted playback and demo clock | Keeps the stage demo repeatable |
| `ui` | Clean-sheet Svelte 5/SvelteKit COP product surface | MapLibre, deck.gl, source lens, provenance lens, alerts |

## First Canonical Entity Set

Keep the first model small and strong:

- `track`: moving thing with source evidence, position, velocity, identity, and confidence.
- `asset`: responder, platform, vehicle, sensor, infrastructure, or resource.
- `hazard_area`: flood, fire, plume, debris, exclusion zone, or evacuation polygon.
- `sensor_footprint`: observed area from drone/video/sensor metadata.
- `alert`: rule or source alert with severity, active state, and affected entities.
- `task`: requested action or operator intent.
- `advisory`: semantic-tier translation meant for civilian or cross-agency consumption.

Each feed should own only its source-specific predicate group. Cross-source fusion should append evidence or write
separate derived predicates under a fusion owner.

## SemStreams Framework Pressure

The SemOps revival should produce concrete upstream asks, not vague "platform needs":

- A reusable deployment metadata schema only if service placement becomes operator-relevant.
- A reusable escalation event/status vocabulary only if inference transitions generalize.
- Better first-class provenance and confidence conventions for projection contracts and graph triples.
- Indexing profile and cardinality guard improvements only after mixed COP feeds prove current `signal`, `control`,
  `content`, and `trace` profiles are insufficient with clean entity boundaries.
- Spatial and temporal query helpers tuned for COP workflows: polygon intersection, nearest track, stale track,
  and moving object windows.
- A documented raw-lane plus current-state projection pattern for high-rate telemetry.
- Edge/core sync guidance for structural edge nodes and inference-heavy core nodes.
- Governance helpers for tolerant-reader adapters that append evidence without replacing owned predicates.

SemOps should stress current indexing behavior deliberately:

- MAVLink, ADS-B, TAK position events, SAPIENT detections, and KLV sensor positions are high-rate `signal`.
- Alerts, tasks, commands, feed health, scenario state, and egress state are durable `control`.
- CAP advisory text, operator notes, chat text, and semantic explanations are `content`.
- Replay steps, native packet references, and decode logs are `trace`.

If those boundaries fail, file a SemStreams ask with a failing SemOps fixture rather than inventing SemOps-only
profile semantics.

## Phased Execution

### Phase 0: Stabilize The Contract

- Move SemOps to current Go and current `github.com/c360studio/semstreams` module path.
- Quarantine or remove old StreamKit processor assumptions that do not match the current framework surface.
- Add a small compile-time test for projection contracts and ownership claims.
- Define the first canonical COP entity set and feed ownership matrix.

### Phase 1: Structural COP

- Build the structural stack with NATS, SemStreams, SemOps API, SemOps UI, and scripted feeds.
- Use MAVLink, TAK/CoT, and CAP first because they prove high-rate telemetry, operator COP, and loose civilian
  alerts.
- Show deterministic fusion: hazard polygon intersects an asset, stale track detection, low-battery alert, and
  source-aware provenance.

### Phase 2: Air Picture And Statistical Escalation

- Add ADS-B and SAPIENT feed boundaries.
- Add a statistical track association service for ambiguous air tracks.
- Write association evidence back to the graph; add UI only if it helps operator decisions.

### Phase 3: Semantic Translation And Standards Egress

- Add KLV footprint extraction and CS API egress through SemConnect.
- Add the semantic translation service for civilian advisories and anomaly explanation.
- Expose provenance and trajectory for every semantic answer.

### Phase 4: Edge/Core Split

- Run structural feeds and deterministic fusion at the edge.
- Run statistical and semantic tiers at the core.
- Use deployment metadata only where it helps edge/core operation.
- Add scripted failover/offline behavior only after the single-stack demo is stable.

## Open Decisions

- Exact entity ID scheme for SemOps COP entities.
- Predicate ownership matrix for each feed and derived fusion owner.
- Whether to reuse SemLink UI components directly or port only the patterns into a new SemOps product surface.
- Whether deployment metadata or tier UI is a value add or a footgun.
- How much SAPIENT and KLV to implement for demo-grade fidelity before claiming conformance.
