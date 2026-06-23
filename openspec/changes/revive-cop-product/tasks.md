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
- [x] 2.7 Add SemStreams flowgraph, lifecycle, port, payload-registry, and config-schema contract guard for feed
      components
- [x] 2.8 Wrap live ADS-B ingress, hosted CAP if promoted, and future SAPIENT feed boundaries as SemStreams input and
      processor components
- [x] 2.9 Add the first concrete MAVLink input/decoder/projector component package using SemStreams payload registry
      and ports
- [x] 2.10 Add the first concrete TAK/CoT input/decoder/projector component package using SemStreams payload registry
      and ports
- [x] 2.11 Bump SemOps to SemStreams `v1.0.0-beta.114` and add the `HTTPClientPort` contract guard
- [x] 2.12 Add the first concrete CAP HTTP poller input and decoder processor components using SemStreams
      `HTTPClientPort`, `TimerPort`, payload registry, and ports
- [x] 2.13 Add the first concrete CAP graph projector processor component using SemStreams request ports and
      born-first restart reconciliation
- [x] 2.14 Wire the CAP HTTP poller -> decoder -> graph-projector chain into the app runtime as an explicit opt-in
      SemStreams component flow
- [x] 2.15 Add CAP HTTP poller stale-source health and provider-contact debug state to the SemStreams component
      contract surface
- [x] 2.16 Add optional CAP replay-store capture to the opt-in runtime component chain
- [x] 2.17 Add the first concrete ADS-B OpenSky HTTP poller, decoder, and projector component package using
      SemStreams ports, payload registry, health, and flow metrics
- [x] 2.18 Add preflight-only SAPIENT HTTP input and decoder components without graph request ports or owner claims
- [x] 2.19 Wire the ADS-B OpenSky-compatible HTTP poller -> decoder -> projector component chain into app runtime
      behind explicit opt-in config
- [x] 2.20 Wire the SAPIENT preflight HTTP input -> decoder component chain into app runtime without graph ownership
      claims
- [x] 2.21 Add local HTTP fixture-provider service evidence for ADS-B and SAPIENT component-flow smoke tests
- [x] 2.22 Add the first concrete weather graph projector processor component using SemStreams request ports,
      payload registry, health, flow metrics, and born-first reconciliation for point forecasts
- [x] 2.23 Wire the opt-in hosted weather fixture input -> decoder -> projector flow through SemStreams component
      lifecycle, ports, payload registry, owner tokens, and component metrics
- [x] 2.24 Add the first concrete SAPIENT graph projector processor component behind explicit graph opt-in using
      SemStreams request ports, payload registry, health, flow metrics, and born-first reconciliation for
      absolute-location detections

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
- [x] 4.15 Add skipped-by-default external PX4/MAVSDK/SITL telemetry smoke harness against the hosted COP snapshot

## 5. Feed Validation And Indexing Ladder

- [x] 5.1 Add initial feed validation and indexing ladder documentation
- [x] 5.2 Record mock, simulator, replay, compliance, and sample-data evidence for each feed
- [x] 5.3 Define expected `indexing_profile` and cardinality risk for each projected entity type
- [ ] 5.4 Prove MAVLink parser, generator, ArduPilot SITL, and PX4 SITL evidence gates
- [x] 5.5 Prove TAK/CoT UDP/TCP seed and replay gates before expanding CoT coverage
- [ ] 5.6 Prove CAP XML schema, NWS sample, and lifecycle gates before CAP conformance or service claims
- [x] 5.7 Verify or reject public SAPIENT compliance/fixture availability before phase commitment
- [ ] 5.8 Run a KLV/SemSource binary proof spike before claiming streaming-binary support
- [ ] 5.9 Run adversarial review for each feed before it enters the structural stack
- [x] 5.10 Treat ADR-055/056 live graph smoke as a MAVLink gate before simulator fidelity claims
- [x] 5.11 Add SemOps-local TAK/CoT native parser gate before UDP/TCP replay or graph projection
- [x] 5.12 Track demo/MVP versus full-product roadmap boundaries for every feed
- [x] 5.13 Add SemOps-owned TAK/CoT UDP/TCP fixture replay harness before graph projection
- [x] 5.14 Add TAK/CoT projection planner for born-first tracks, marker control state, and GeoChat content state
- [x] 5.15 Add CAP parser fixtures and append-evidence projection gate before hosted CAP service work
- [x] 5.16 Add CAP live graph smoke for born-first append-evidence readback
- [x] 5.17 Add CAP lifecycle-status readback without claiming authoritative hazard status
- [x] 5.18 Add CAP raw XML lifecycle fixture replay for scenario-runner input
- [x] 5.19 Add service-promotion matrix and current CS API source anchors without claiming bridge implementation
- [x] 5.20 Add initial ADS-B OpenSky-shaped parser fixture gate without claiming live ADS-B support
- [x] 5.21 Add ADS-B projection/readback fixture gate without claiming hosted ADS-B service
- [x] 5.22 Add deterministic ADS-B OpenSky fixture replay through the scenario-runner seam
- [x] 5.23 Add hosted ADS-B adapter seam without claiming live ADS-B service
- [x] 5.24 Add opt-in ADS-B structural scenario replay through the hosted adapter seam
- [x] 5.25 Record ADS-B component-promotion gate so scenario replay is not mistaken for a hosted feed service
- [x] 5.26 Record official SAPIENT GOV.UK, BSI Flex 335 v2, Dstl protobuf, test-harness, and Apex middleware anchors
- [ ] 5.27 Run or qualify the Dstl BSI Flex 335 v2 Test Harness before SAPIENT compliance claims
- [x] 5.28 Add parser-only SAPIENT BSI Flex 335 v2 JSON preflight fixture gate before graph projection
- [x] 5.29 Evaluate a portable Linux/CI-friendly SAPIENT preflight suite as an ecosystem contribution
- [x] 5.30 Add SAPIENT BSI Flex 335 v2 descriptor-based binary payload decode gate
- [x] 5.31 Decide whether generated SAPIENT Go bindings are needed beyond dynamic descriptors
- [x] 5.32 Add bounded SAPIENT raw lane and replay for JSON/binary preflight payloads
- [x] 5.33 Run SAPIENT projection ownership/indexing review before graph writes
- [x] 5.34 Record CAP component-promotion gate so scenario replay is not mistaken for hosted polling/webhook service
- [x] 5.35 Add CAP HTTP poller/decoder component gate without claiming live NWS service or CAP conformance
- [x] 5.36 Add CAP graph-projector component gate without claiming default hosted CAP service
- [x] 5.37 Record opt-in CAP runtime composition while keeping default live NWS/IPAWS provider claims gated
- [x] 5.38 Add deterministic CAP poller stale-source health evidence without claiming provider-specific stale policy
- [x] 5.39 Add deterministic provider-shaped CAP HTTP capture/replay evidence before live NWS sample capture
- [x] 5.40 Add ADS-B OpenSky HTTP component-promotion gate without claiming default live OpenSky service support
- [x] 5.41 Add SAPIENT preflight component gate without claiming product support, conformance, or graph projection
- [x] 5.42 Add ADS-B app-runtime component gate without claiming default live OpenSky reliability or receiver support
- [x] 5.43 Add SAPIENT app-runtime preflight gate without claiming product support, conformance, or graph projection
- [x] 5.44 Record SemSource binary fixture handoff as synthetic storage/governance proof until a real KLV/SKG fixture exists
- [ ] 5.45 Identify a legal representative KLV/SKG binary fixture before protocol conformance or streaming-binary claims
- [x] 5.46 Accept SemSource beta.114 binary boundary while keeping KLV/MISB/STANAG details in SemOps
- [x] 5.47 Design the SemOps-owned KLV/MISB worker boundary from storage references to governed derived facts
- [x] 5.48 Record public-sample versus deterministic fixture strategy for KLV/MISB
- [ ] 5.49 Verify public KLV sample license/provenance before vendoring, downloading, or using in CI
- [x] 5.50 Choose parser and demux strategy for the first KLV/MISB worker spike
- [x] 5.51 Define first `semops.klv_*` payload registry schemas with SemStreams BaseMessage round-trip tests
- [x] 5.52 Define SemStreams component port skeletons for media-ref input, demux, decode, and projector stages
- [x] 5.53 Add an opt-in KLV worker spike against a local deterministic fixture without graph writes
- [ ] 5.54 Add adversarial review before any MISB/STANAG engineering-support language appears in demo copy
- [x] 5.55 Add a deterministic MISB ST 0601 local-set decoder core test without graph writes
- [x] 5.56 Add DJI sensor/telemetry and weather as first-class feed/layer roadmap entries
- [x] 5.57 Record that DJI video reinforces generic media-reference handling without moving KLV demux into SemSource
- [x] 5.58 Add weather parser/source evidence for OGC API EDR, Open-Meteo, NWS CAP alerts, and visual tile sources
- [x] 5.59 Add DJI feed evidence for telemetry, command authority, media references, and replay fixtures
- [x] 5.60 Add fixture-grade FFmpeg/ffprobe KLV demux worker path with explicit stream selection and bounded output
- [x] 5.61 Add deterministic MISB ST 0601 truth JSON to generated-packet to decoded-frame fixture acceptance
- [x] 5.62 Add optional deterministic truth-to-MPEG-TS demux smoke using local FFmpeg tooling
- [x] 5.63 Add KLV sensor/frame-center projection contract without footprint polygon claims
- [x] 5.64 Add bounded KLV packet splitting and opt-in storage-ref materialization gate
- [x] 5.65 Add bounded KLV packet storage-ref materialization for decoder workers
- [x] 5.66 Add opt-in public KLV sample smoke gate without vendoring media
- [x] 5.67 Add DJI fixture input and decoder components using SemStreams payload registry and ports without graph
      writes or live bridge claims
- [x] 5.68 Add weather fixture input and decoder components using SemStreams payload registry and ports without graph
      writes or live provider claims
- [x] 5.69 Add OGC EDR-shaped position fixture and decoder gate before standards-facing tactical weather claims
- [x] 5.70 Add OGC EDR-shaped area, trajectory, and corridor parser gates before route-weather claims
- [x] 5.71 Add tactical-weather graph contract and pure projector-plan gate before UI or route-safety claims
- [x] 5.72 Add tactical-weather graph writer and projector component gate for point forecasts without claiming live
      provider support, spatial runtime payloads, UI semantics, or route-safety authority
- [x] 5.73 Add opt-in hosted tactical-weather point-fixture runtime evidence without claiming live provider support,
      cache/stale policy, spatial runtime payloads, UI semantics, or route-safety authority
- [x] 5.74 Add Caddy-routed COP API readback evidence for opt-in weather point fixtures without claiming tactical
      weather UI semantics or route-safety authority
- [x] 5.75 Add narrow SAPIENT absolute-location detection projection and readback evidence without claiming
      conformance, hosted graph production, tasking, association, UTM conversion, or range/bearing support
- [x] 5.76 Add opt-in SAPIENT absolute-location runtime graph component evidence without claiming conformance,
      product service support, tasking, association, UTM conversion, or range/bearing support
- [x] 5.77 Add opt-in SAPIENT absolute-location stack-smoke assertions without claiming conformance, product service
      support, tasking, association, UTM conversion, or range/bearing support

## 6. Structural COP Stack

- [x] 6.1 Add Compose stack for NATS, SemStreams, SemOps API, SemOps UI, and scenario runner
- [x] 6.2 Add MAVLink, TAK/CoT, and CAP structural adapters while keeping broader EDXL out of Phase 1
- [x] 6.3 Add service health and active state polling for long-running demo runs
- [x] 6.4 Add scenario playback for a flood/evacuation and shared-airspace vignette
- [x] 6.5 Add smoke test that verifies graph state from at least two feed types
- [x] 6.6 Run adversarial review on demo credibility, monitoring, and graph/index cardinality before Phase 1 signoff
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
- [x] 6.19 Expand Compose from graph scaffold to API, UI, scenario runner, and feed transports
- [x] 6.20 Add fixture-backed SemOps COP API, Svelte UI container, and Caddy same-origin ingress to the stack smoke
- [x] 6.21 Add hosted MAVLink UDP-to-Caddy-snapshot smoke for the first product-visible live graph path
- [x] 6.22 Add TAK/CoT graph writer, adapter graph path, stack constructor, and opt-in hosted listener wiring
- [x] 6.23 Add TAK/CoT live graph smoke and hosted UDP-to-Caddy-snapshot readback
- [x] 6.24 Add CAP born-first graph writer path for hazard evidence plans
- [x] 6.25 Add SemStreams prefix discovery for graph-backed COP readback
- [x] 6.26 Add CAP live graph smoke to the one-command COP stack
- [x] 6.27 Add in-process HADR scenario runner core for MAVLink, TAK/CoT, and CAP lifecycle replay
- [x] 6.28 Add hosted scenario-runner service with active status polling to the one-command stack smoke
- [x] 6.29 Assert hosted scenario-runner graph writes are visible through the COP snapshot
- [x] 6.30 Move SemStreams dedicated health to a non-conflicting container port and print NATS/SemStreams diagnostics
      on compose startup failure
- [x] 6.31 Remove required CoT/CAP seed IDs from the COP snapshot path and rely on prefix discovery for hosted
      state
- [x] 6.32 Add source/type cardinality diagnostics for prefix-discovered snapshot state
- [x] 6.33 Promote prefix discovery pressure into source-health warning alerts
- [x] 6.34 Adopt SemStreams typed prefix-query cursor pagination for graph-backed COP discovery
- [x] 6.35 Bind ADS-B ownership for structural replay while keeping live ADS-B service out of the MVP default
- [x] 6.36 Delete stale StreamKit-era robotics flow config so raw-subject topology does not pollute current context
- [x] 6.37 Move hosted feed service composition onto SemStreams input/processor component metadata, flowgraph,
      registered-payload, and port/config surfaces
- [x] 6.38 Add concrete MAVLink UDP input, decoder processor, and projector processor components before app runtime
      rewiring
- [x] 6.39 Rewire hosted MAVLink UDP ingestion onto the SemStreams input -> decoder processor -> projector processor
      flow
- [x] 6.40 Record SemStreams utility package reuse as a SemOps framework-compliance expectation
- [x] 6.41 Record flow backpressure and Prometheus telemetry as component-boundary design gates
- [x] 6.42 Add concrete TAK/CoT UDP input, TCP input, decoder processor, and projector processor components
- [x] 6.43 Rewire hosted TAK/CoT UDP/TCP ingestion onto the SemStreams input -> decoder processor -> projector
      processor flow
- [x] 6.44 Record that ADS-B scenario replay remains an adapter harness until live ingress forces a component package
- [x] 6.45 Record SAPIENT hosted component gate and SemStreams `HTTPClientPort` adoption
- [x] 6.46 Add initial CAP hosted HTTP poller and raw decoder component package while keeping default-stack live
      provider wiring open
- [x] 6.47 Add initial CAP born-first graph projector component package while keeping default-stack hosting open
- [x] 6.48 Wire CAP HTTP polling through the hosted app composition root behind `SEMOPS_CAP_ENABLED=false` by default
- [x] 6.49 Thread CAP `stale_after` runtime config into the opt-in Compose/runtime chain
- [x] 6.50 Thread optional `SEMOPS_CAP_REPLAY_PATH` into the opt-in Compose/runtime chain
- [x] 6.51 Add initial ADS-B OpenSky HTTP poller, decoder, and projector components while keeping runtime hosting
      and receiver paths gated
- [x] 6.52 Add initial SAPIENT preflight HTTP input and decoder components while keeping graph projection and product
      service claims gated
- [x] 6.53 Wire ADS-B HTTP components into hosted app composition behind `SEMOPS_ADSB_ENABLED=false` by default
- [x] 6.54 Wire SAPIENT preflight HTTP components into hosted app composition behind `SEMOPS_SAPIENT_ENABLED=false`
      by default
- [x] 6.55 Add a local ADS-B/SAPIENT HTTP fixture-provider service to the Compose stack without live network
      dependencies
- [x] 6.56 Add one-command stack smoke coverage for opt-in ADS-B HTTP graph readback and SAPIENT decoded-stream
      preflight
- [x] 6.57 Fix hosted component runtime lifecycle so startup/connect context deadlines do not cancel long-running
      input and processor components
- [ ] 6.58 Split broader EDXL beyond CAP into a later feed-validation gate if product needs justify it
- [x] 6.59 Run full one-command stack smoke after Playwright, active-status polling, shared-airspace, and CAP scope
      updates
- [x] 6.60 Expose hosted component health and flow through Prometheus `/metrics` and assert Caddy-routed
      `semops_component_*` samples in the one-command stack smoke
- [x] 6.61 Add a UI-facing COP runtime facade derived from SemStreams component health and flow sources
- [x] 6.62 Wire opt-in KLV local media-ref -> demux -> decode -> projector hosted runtime with ownership and
      component metrics
- [x] 6.63 Add a one-command opt-in KLV stack smoke after fixture provenance and runtime-resource posture are clean
- [x] 6.64 Add initial weather point-forecast graph projector component while keeping hosted runtime, live provider,
      cache/stale policy, and route-weather behavior gated
- [x] 6.65 Wire opt-in weather point-fixture input -> decoder -> projector hosted runtime with ownership and component
      metrics while keeping weather disabled by default
- [x] 6.66 Add opt-in one-command weather stack smoke for fixture-backed graph projection, COP snapshot readback, and
      component/runtime flow evidence
- [x] 6.67 Wire SAPIENT decoded-message graph projection through hosted runtime behind
      `SEMOPS_SAPIENT_GRAPH_ENABLED=false` by default, with SAPIENT ownership registered only when graph mode is
      enabled
- [x] 6.68 Add opt-in one-command SAPIENT graph smoke for fixture-backed detection projection, COP snapshot readback,
      component metrics, and runtime flow evidence

## 7. COP UI

- [x] 7.1 Build Svelte 5 first screen as the usable COP, not a landing page
- [x] 7.2 Add map layer for tracks, assets, TAK tasks/advisories, and hazard areas
- [x] 7.3 Add source and provenance lenses for selected entities
- [x] 7.4 Validate whether topology/tier UI answers a real operator question before building it
- [x] 7.5 Add component tests and accessibility checks for core operator flows
- [x] 7.6 Run adversarial UX review before promoting topology, tier, or orchestration UI
- [x] 7.7 Inventory current UI state and decide clean-sheet COP UI over flow-runtime restoration
- [x] 7.8 Record MapLibre/deck.gl starting stack and dynamic ontology UI scope gate
- [x] 7.9 Add snapshot client tests and fixture fallback for the first SemOps API/UI contract
- [x] 7.10 Run adversarial review on the API/UI/Caddy spine before claiming graph-backed COP completeness
- [x] 7.11 Replace the static-only API provider with a graph-backed MAVLink snapshot provider and fixture fallback
- [x] 7.12 Add first MapLibre/deck.gl tactical layer for tracks, assets, hazards, labels, and picking
- [x] 7.13 Add tested selection reconciliation and pressed-state affordances for map/alert controls
- [x] 7.14 Add TAK/CoT task and advisory readback to the COP API/UI contract
- [x] 7.15 Add sensor footprints and alert geometry after their graph contracts have evidence
- [x] 7.16 Add CAP hazard evidence readback to the COP API contract
- [x] 7.17 Record SemConnect/SemLink graph visualization prior art for future graph lenses
- [x] 7.18 Surface derived CAP hazard lifecycle status in the COP view model
- [x] 7.19 Surface compact prefix-discovery counts in source cards
- [x] 7.20 Surface prefix discovery pressure in the alert list
- [x] 7.21 Surface ADS-B prefix-discovered aircraft tracks in the COP API contract
- [x] 7.22 Add fixture-backed Playwright browser smoke for API-backed ADS-B feed/discovery readback and selection
      provenance
- [x] 7.23 Surface SemStreams component runtime health and flow in source cards without adding topology controls
- [x] 7.24 Add KLV sensor-footprint COP API readback for sensor point, frame center, ray geometry, media/packet
      provenance, and claim posture
- [x] 7.25 Add KLV sensor-footprint map layer, selected-entity provenance inspector, and Playwright smoke without
      footprint polygon or video-player claims
- [x] 7.26 Add weather-observation COP API readback with source/provenance claim posture while deferring tactical
      weather map layers and route-safety UI
- [x] 7.27 Add weather-observation point evidence to the COP UI with source diagnostics, provenance inspector, and
      Playwright smoke while deferring visual weather tiles, live providers, and route-safety decisions
- [x] 7.28 Add SAPIENT graph-prefix COP API readback for absolute-location detections while deferring dedicated
      SAPIENT UI semantics

## 8. Tier Escalation And Egress

- [x] 8.1 Add ADS-B and SAPIENT feed boundaries
- [ ] 8.2 Add statistical track association for ambiguous air tracks
- [ ] 8.3 Add KLV footprint polygon extraction beyond current sensor/frame-center projection
- [ ] 8.4 Add SemConnect CS API bidirectional interop as a standards-facing bridge after the command-impedance gate is
      reviewed
- [ ] 8.5 Add semantic translation and anomaly explanation with provenance and trajectory
- [ ] 8.6 Run adversarial review before SAPIENT, KLV, semantic, or standards-conformance claims are demoed
- [x] 8.7 Record CS API tasking impedance gate for TTL, priority, authority, local override, and async status edge
      cases before command/control implementation
- [x] 8.8 Run SAPIENT projection ownership/indexing review for absolute-location detections while keeping hosted
      service, tasking, association, UTM, and range/bearing gates closed

## 9. Upstream SemStreams Asks

- [ ] 9.1 File manifest/tier placement ask only if the scope gate proves it is useful
- [ ] 9.2 File escalation event/status vocabulary ask only after inference evidence proves the need
- [ ] 9.3 File provenance/confidence convention ask after feed contracts stabilize
- [ ] 9.4 File spatial-temporal query helper asks from COP workflows
- [ ] 9.5 File raw-lane/current-state projection guidance after MAVLink and one non-MAVLink feed prove it
- [ ] 9.6 File indexing profile/cardinality helper asks only after mixed COP feeds prove the need
- [ ] 9.7 Run adversarial ownership review before filing each upstream SemStreams ask
- [x] 9.8 Feed back owner-token ergonomics and capture typed-token/evidence-declaration response
- [x] 9.9 Reconcile SemStreams prefix-discovery issue #302 and adopt typed cursor pagination locally
- [x] 9.10 File SemStreams component backpressure telemetry issue #309 from hosted feed migration pressure
- [x] 9.11 File SemStreams external HTTP polling/client port metadata issue #310
- [x] 9.12 Adopt SemStreams `HTTPClientPort` from `v1.0.0-beta.114` for external polling/client feed boundaries
- [x] 9.13 File SemStreams `TimerPort` flowgraph cadence-boundary issue #312 from the CAP HTTP poller component
