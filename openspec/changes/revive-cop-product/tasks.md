## 1. OpenSpec And Planning Baseline

- [x] 1.1 Add SemOps OpenSpec project context and COP revival change set
- [x] 1.2 Record SemOps as the complete COP product owner and SemLink as basic-demo prior art
- [x] 1.3 Link the architecture note and tickets back to the OpenSpec change
- [x] 1.4 Run the first adversarial review with architect, reviewers, and technical-writer roles before implementation
  begins
- [x] 1.5 Triage SemOps GitHub issue #1 as an ADR-055/056 breaking-tag compliance gate

## 2. Modernize Framework Contract

- [x] 2.1 Update SemOps to the current SemStreams module path
- [x] 2.2 Align the Go toolchain with current SemStreams
- [x] 2.3 Delete no-reference StreamKit/BaseProcessor assumptions and hold only MAVLink extraction references
- [x] 2.4 Add a compile-time contract test importing current projection and graph mutation packages
- [x] 2.5 Run adversarial review on framework ownership and migration blast radius before broad rewiring
- [x] 2.6 Migrate SemOps runtime/projector wiring to SemStreams typed `ownership.OwnerToken` bind results

## 3. COP Model And Governance

- [x] 3.1 Define canonical COP entities: track, asset, hazard area, sensor footprint, alert, task, advisory
- [x] 3.2 Define SemOps predicate names and candidate upstream SemStreams vocabulary terms
- [x] 3.3 Define projection ownership contracts for strict feed owners, tolerant feed owners, and fusion owners
- [x] 3.4 Define provenance and confidence conventions for source facts and derived facts
- [x] 3.5 Add tests that reject overlapping replace-owned predicate claims
- [x] 3.6 Run adversarial review on entity boundaries, predicate ownership, provenance, and confidence claims

## 4. MAVLink Salvage

- [x] 4.1 Move MAVLink parser and generator behind a modern package boundary
- [x] 4.2 Move the SITL controller behind a modern package boundary or deliberately reject it
- [x] 4.3 Preserve heartbeat, global position, attitude, battery, and command coverage where tests prove it
- [x] 4.4 Replace old message/processor wiring with SemStreams projection writes
- [x] 4.5 Project decoded MAVLink current vehicle state into signal-profiled SemStreams mutation plans
- [x] 4.6 Add real-frame codec tests that do not mock away MAVLink decoding
- [ ] 4.7 Add PX4 SITL or MAVSDK smoke evidence after the current module/toolchain migration is stable
- [x] 4.8 Keep raw frames on a bounded lane and wire projection plans to live SemStreams graph writes
- [x] 4.9 Add a tested SemStreams graph request/reply writer boundary for MAVLink projection plans
- [x] 4.10 Add a bounded MAVLink raw frame lane with current-state source references
- [x] 4.11 Add generated/replay MAVLink live graph smoke before PX4/SITL fidelity work
- [x] 4.12 Add skipped-by-default MAVLink live graph smoke harness target
- [x] 4.13 Add MAVLink restart create-conflict reconciliation for existing asset/track births
- [x] 4.14 Add opt-in MAVLink UDP datagram transport reader for hosted runtime

## 5. Feed Validation And Indexing Ladder

- [x] 5.1 Add initial feed validation and indexing ladder documentation
- [x] 5.2 Record mock, simulator, replay, compliance, and sample-data evidence for each feed
- [x] 5.3 Define expected `indexing_profile` and cardinality risk for each projected entity type
- [ ] 5.4 Prove MAVLink parser, generator, ArduPilot SITL, and PX4 SITL evidence gates
- [ ] 5.5 Prove TAK/CoT UDP/TCP seed and replay gates before expanding CoT coverage
- [ ] 5.6 Prove CAP XML schema and NWS sample gates before adding loose-reader behavior
- [ ] 5.7 Verify or reject public SAPIENT compliance/fixture availability before phase commitment
- [ ] 5.8 Run a KLV/SemSource binary proof spike before claiming streaming-binary support
- [ ] 5.9 Run adversarial review for each feed before it enters the structural stack
- [x] 5.10 Treat ADR-055/056 live graph smoke as a MAVLink gate before simulator fidelity claims

## 6. Structural COP Stack

- [ ] 6.1 Add Compose stack for NATS, SemStreams, SemOps API, SemOps UI, and scenario runner
- [ ] 6.2 Add MAVLink, TAK/CoT, and CAP/EDXL structural adapters
- [ ] 6.3 Add service health and active state polling for long-running demo runs
- [ ] 6.4 Add scenario playback for a flood/evacuation and shared-airspace vignette
- [ ] 6.5 Add smoke test that verifies graph state from at least two feed types
- [ ] 6.6 Run adversarial review on demo credibility, monitoring, and graph/index cardinality before Phase 1 signoff
- [x] 6.7 Add an in-process MAVLink adapter harness with pollable health before Compose wiring
- [x] 6.8 Add durable MAVLink replay fixture storage behind the bounded raw lane
- [x] 6.9 Add SemStreams NATS request/retry adapter for graph mutation writers
- [x] 6.10 Add testable MAVLink structural wiring for NATS-backed graph writes
- [x] 6.11 Run breaking-tag compliance smoke for MAVLink graph writes
- [x] 6.12 Add live graph smoke harness that skips unless a SemStreams NATS URL is provided
- [x] 6.13 Add explicit SemOps COP owner registration and heartbeat smoke coverage
- [x] 6.14 Assert clean-stack graph-ingest foreign-edge, owner-token, and indexing-profile counter deltas
- [x] 6.15 Wire COP ownership registration into the hosted adapter or SemOps composition root
- [x] 6.16 Carry live graph metrics URL into the one-command hosted stack smoke
- [x] 6.17 Rerun one-command hosted stack smoke after typed-token migration and restart reconciliation
- [x] 6.18 Add hosted MAVLink UDP listener lifecycle wiring behind explicit runtime config
- [ ] 6.19 Expand Compose from graph scaffold to API, UI, scenario runner, and feed transports
- [x] 6.20 Add fixture-backed SemOps COP API, Svelte UI container, and Caddy same-origin ingress to the stack smoke
- [x] 6.21 Add hosted MAVLink UDP-to-Caddy-snapshot smoke for the first product-visible live graph path

## 7. COP UI

- [x] 7.1 Build Svelte 5 first screen as the usable COP, not a landing page
- [ ] 7.2 Add map layer for tracks, assets, hazard areas, footprints, alerts, and tasks
- [ ] 7.3 Add source and provenance lenses for selected entities
- [ ] 7.4 Validate whether topology/tier UI answers a real operator question before building it
- [ ] 7.5 Add component tests and accessibility checks for core operator flows
- [ ] 7.6 Run adversarial UX review before promoting topology, tier, or orchestration UI
- [x] 7.7 Inventory current UI state and decide clean-sheet COP UI over flow-runtime restoration
- [x] 7.8 Record MapLibre/deck.gl starting stack and dynamic ontology UI scope gate
- [x] 7.9 Add snapshot client tests and fixture fallback for the first SemOps API/UI contract
- [x] 7.10 Run adversarial review on the API/UI/Caddy spine before claiming graph-backed COP completeness
- [x] 7.11 Replace the static-only API provider with a graph-backed MAVLink snapshot provider and fixture fallback
- [x] 7.12 Add first MapLibre/deck.gl tactical layer for tracks, assets, hazards, labels, and picking

## 8. Tier Escalation And Egress

- [ ] 8.1 Add ADS-B and SAPIENT feed boundaries
- [ ] 8.2 Add statistical track association for ambiguous air tracks
- [ ] 8.3 Add KLV footprint extraction
- [ ] 8.4 Add SemConnect CS API egress as a standards-facing projection
- [ ] 8.5 Add semantic translation and anomaly explanation with provenance and trajectory
- [ ] 8.6 Run adversarial review before SAPIENT, KLV, semantic, or standards-conformance claims are demoed

## 9. Upstream SemStreams Asks

- [ ] 9.1 File manifest/tier placement ask only if the scope gate proves it is useful
- [ ] 9.2 File escalation event/status vocabulary ask only after inference evidence proves the need
- [ ] 9.3 File provenance/confidence convention ask after feed contracts stabilize
- [ ] 9.4 File spatial-temporal query helper asks from COP workflows
- [ ] 9.5 File raw-lane/current-state projection guidance after MAVLink and one non-MAVLink feed prove it
- [ ] 9.6 File indexing profile/cardinality helper asks only after mixed COP feeds prove the need
- [ ] 9.7 Run adversarial ownership review before filing each upstream SemStreams ask
- [x] 9.8 Feed back owner-token ergonomics and capture typed-token/evidence-declaration response
