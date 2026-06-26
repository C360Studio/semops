## ADDED Requirements

### Requirement: Container boundaries follow deployment forces

SemOps SHALL containerize components when isolation, placement, scaling, protocol exposure, secrets, cost, or
failure domain justify a service boundary.

#### Scenario: Feed adapter owns an external protocol boundary

- **WHEN** an adapter listens for UDP, TCP, protobuf, video metadata, or public alert feeds
- **THEN** it may run as a service while keeping mapping logic testable as a library

#### Scenario: Deterministic mapper has no deployment force

- **WHEN** a mapper only transforms decoded data into canonical projections
- **THEN** it remains a library component unless placement, scaling, or failure isolation requires a service

### Requirement: Hosted feed services use SemStreams flow-based lifecycle

SemOps SHALL use SemStreams component lifecycle, flowgraph, port, payload-registry, and config-schema patterns for
hosted feed services instead of inventing a SemOps-local lifecycle or flow framework.

SemOps SHALL also prefer SemStreams shared utility packages for common runtime concerns, including `natsclient`,
`pkg/errs`, `pkg/cache`, and `pkg/buffer`, before creating SemOps-local equivalents.

#### Scenario: Flow backpressure is telemetry-driven

- **WHEN** SemOps wires input and processor components through NATS or JetStream ports
- **THEN** each component exposes SemStreams `Health()` and `DataFlow()` values for throughput, byte rate, error rate,
  and last activity
- **AND** hosted components expose Prometheus metrics through SemStreams metric helpers where the component runtime
  provides a metrics registry
- **AND** one-command stack smoke evidence scrapes SemOps component metrics through Caddy before any UI runtime-flow
  panel is treated as backed by live component telemetry
- **AND** durable stream backpressure, replay, redelivery, or ack behavior is represented as JetStream port
  configuration rather than hidden inside adapter code
- **AND** SemOps adds bounded `pkg/buffer` queues or `pkg/cache` only when telemetry shows queue growth, drop pressure,
  retry/redelivery pressure, smoothing need, or repeated reads across components

#### Scenario: Network servers are input components

- **WHEN** SemOps listens for native UDP, TCP, serial, file, webhook, or polling feed input
- **THEN** the listener is modeled as a SemStreams input component with declared network, file, or request ingress
  ports
- **AND** the input component emits registered raw-feed messages to declared SemStreams stream output ports
- **AND** the input component does not parse, project, or write graph state except for minimal envelope validation and
  transport health

#### Scenario: External polling clients remain component-visible

- **WHEN** SemOps polls public or vendor HTTP feeds such as NWS CAP, OpenSky ADS-B, or Apex/SAPIENT endpoints
- **THEN** the poller is modeled as a SemStreams input component with cadence, endpoint, auth, timeout, retry, cache,
  rate-limit, and stale-source behavior exposed through config schema and health
- **AND** the poller emits registered raw-feed messages to declared stream output ports before parser processors run
- **AND** the poller declares a SemStreams `HTTPClientPort` for outbound HTTP resource metadata
- **AND** timer-driven pollers declare a sibling `TimerPort` referenced by `HTTPClientPort.TriggerPort`

#### Scenario: Feed adapters are processor components

- **WHEN** SemOps decodes, validates, replays, maps, fuses, or projects native feed data
- **THEN** that behavior runs inside SemStreams processor components that subscribe to declared input ports and publish
  declared output ports
- **AND** parser or decoder processors emit decoded feed messages that other components can tap
- **AND** projection processors write governed graph mutations only through declared SemStreams NATS request ports

#### Scenario: Graph mutation failures use the SemStreams ADR-060 error contract

- **WHEN** SemOps writes governed graph mutations through SemStreams request/reply ports
- **THEN** requesters use classified request helpers that preserve stable SemStreams graph error codes
- **AND** feed writers reconcile `entity_already_exists`, `owner_lease_stale`, and future graph mutation failures from
  typed classified Go errors
- **AND** SemOps does not parse legacy text response bodies or `MutationResponse` failure fields for conflict handling

#### Scenario: Flowgraph defines feed topology

- **WHEN** SemOps composes hosted feed behavior
- **THEN** component connections are described through SemStreams `component/flowgraph` nodes, ports, and edges
- **AND** raw NATS subjects are port configuration, not hidden coupling between SemOps packages

#### Scenario: Flow payloads use SemStreams payload registry

- **WHEN** an input or processor component emits raw, decoded, or projected feed payloads on a stream port
- **THEN** the payload is wrapped in `message.BaseMessage`
- **AND** the payload type is registered in `payloadregistry.Registry`
- **AND** consumers decode through SemStreams message/payload-registry APIs rather than ad hoc JSON or subject naming

#### Scenario: Component config is schema-driven

- **WHEN** a hosted feed component accepts runtime configuration
- **THEN** SemOps exposes that configuration through SemStreams `component.ConfigSchema`
- **AND** the component config schema includes ownership, payload type, stream port, raw-lane bounds, transport, retry,
  and timeout settings that materially affect runtime behavior

#### Scenario: Codecs remain pure libraries

- **WHEN** a package only decodes, validates, replays, or maps native feed bytes
- **THEN** it remains a pure library and is not required to implement SemStreams lifecycle interfaces
- **AND** service lifecycle, ports, payload registry, flowgraph topology, config, health, and SemStreams client
  ownership stay in hosted input or processor components

#### Scenario: Legacy flow configs are not retained

- **WHEN** old StreamKit, BaseProcessor, or raw-subject flow files no longer describe the active runtime
- **THEN** they are deleted or quarantined outside the active repo context
- **AND** new flow/config artifacts derive from current SemStreams component metadata, flowgraph topology, registered
  payload types, ports, and config schemas

### Requirement: Product e2e evidence enters through feed boundaries

SemOps SHALL distinguish product e2e evidence from graph-contract evidence.

#### Scenario: Full-stack smokes cannot bypass feed components

- **WHEN** a smoke is used to claim product, operator UI, simulator-fidelity, provider-integration, CS API, or
  command-control behavior
- **THEN** test input enters through a supported external feed boundary or through a SemStreams input component that
  emits registered native/raw payloads
- **AND** the smoke exercises hosted component lifecycle, flowgraph ports, payload registry decoding, graph request
  ports, owner tokens, component health, `DataFlow()`, and Prometheus telemetry when those surfaces exist for the
  feed
- **AND** the smoke MUST NOT seed target COP state by publishing graph mutations, decoded payloads, or projected
  payloads directly around the hosted component graph
- **AND** direct graph smokes remain contract-only evidence for projection, SemStreams compatibility, restart
  reconciliation, indexing, and ownership behavior
- **AND** direct graph smokes MUST NOT satisfy product e2e, operator UI, simulator-fidelity, command-control,
  CS API, provider-integration, or standards-conformance gates

#### Scenario: One live owner incarnation writes each product e2e path

- **WHEN** a product e2e stack starts graph-writing components
- **THEN** each governed feed owner in the path under test is bound by one live process incarnation
- **AND** a second live process MUST NOT bind the same feed owner to seed demo state for the same product e2e path
- **AND** owner-token mismatch warnings, stale owner leases, or observe-only owner mismatch deltas fail product e2e
  evidence

### Requirement: Structural demo stack starts with one command

The first COP stack SHALL run locally with a single documented command after dependencies are installed.

#### Scenario: Phase 1 stack is healthy

- **WHEN** the structural stack starts
- **THEN** NATS, SemStreams, SemOps API, SemOps UI, scenario runner, and Phase 1 adapters expose health state
- **AND** SemStreams health probes use a dedicated non-conflicting listener port rather than racing the service-manager
  HTTP port

#### Scenario: Hosted scenario runner reports concrete playback state

- **WHEN** the local Compose stack starts `semops-scenario-runner`
- **THEN** the service reports the first HADR scenario playback state on startup
- **AND** `/healthz` remains unavailable until playback succeeds
- **AND** `/scenario/status` reports the scenario id, state, ingress mode, completed steps, failed steps, feed-boundary
  delivery count, mutation count, contract graph mutation-attempt count, and last error
- **AND** Caddy routes browser-facing `/scenario/status` to the hosted scenario runner
- **AND** the one-command smoke polls the Caddy-routed status endpoint by default rather than inferring scenario
  success from logs
- **AND** the smoke fails fast on explicit scenario failure and treats stale status progress as a wedged run with
  diagnostic output
- **AND** default product mode emits MAVLink and TAK/CoT through hosted UDP feed boundaries without opening a
  SemStreams client, binding graph owners, or writing graph mutations
- **AND** default product mode status reports `ingress_mode=feed-boundary`, zero `mutations`, zero
  `contract_graph_mutation_attempts`, and `feed_boundary_deliveries` equal to completed steps
- **AND** direct graph projection mode, when retained as `SEMOPS_SCENARIO_MODE=contract`, is labeled
  contract/replay infrastructure rather than product e2e

#### Scenario: Hosted scenario playback and CAP fixture polling are product-visible

- **WHEN** `semops-scenario-runner` reports succeeded in product mode through hosted MAVLink and TAK/CoT feed
  boundaries
- **THEN** the Caddy-routed COP snapshot contains the scenario MAVLink track
- **AND** the snapshot contains the scenario TAK/CoT task and advisory
- **AND** when CAP HTTP polling is enabled, the snapshot contains the local CAP fixture hazard with geometry and
  provenance from the hosted CAP component path
- **AND** this check runs before direct feed-specific live graph smokes so product visibility fails fast
- **AND** this check is not satisfied by scenario-runner direct graph writes
- **AND** the smoke checks SemStreams owner-token mismatch metrics before direct graph contract smokes run

#### Scenario: HADR shared-airspace playback is product-visible

- **WHEN** the local Compose smoke enables the local ADS-B HTTP fixture component alongside the HADR scenario runner
- **THEN** a single Caddy-routed COP snapshot contains scenario MAVLink/TAK state, hosted CAP fixture state, and an
  ADS-B aircraft track
- **AND** the ADS-B evidence comes from the hosted ADS-B component flow, not a second scenario-runner owner claim
- **AND** this remains local fixture evidence rather than a live OpenSky, ASTERIX, or receiver-support claim

#### Scenario: Browser ingress is same-origin in development

- **WHEN** the local COP stack exposes the operator UI
- **THEN** Caddy routes browser traffic to the Svelte UI and proxies SemOps API paths on the same origin
- **AND** the direct SemOps API port remains available for diagnostics and smoke checks

#### Scenario: Health checks use active polling

- **WHEN** a long-running demo or paid operation is supervised
- **THEN** SemOps polls authoritative health, graph, and scenario state instead of relying only on passive logs
- **AND** compose startup failures print NATS and SemStreams diagnostics so local Docker storage exhaustion is visible
  without a separate log hunt

#### Scenario: Feed adapter exposes active health state

- **WHEN** a feed adapter ingests native frames before the container stack exists
- **THEN** its library harness exposes pollable counters for received frames, captured frames, decoded packets, graph
  mutations, parse errors, projection drops, write errors, last raw reference, and last error
- **AND** invalid native frames and unsupported packets do not reach the graph writer silently

#### Scenario: Graph mutation writers use retry-aware request/reply

- **WHEN** a SemOps feed commits governed graph mutations through SemStreams NATS request/reply
- **THEN** it uses the framework mutation path with retry-aware request handling for transient responder startup races
- **AND** query-style non-retry requests are not used for mutation writers

#### Scenario: Feed service wiring is testable before Compose hosting

- **WHEN** SemOps builds a feed adapter service boundary
- **THEN** the structural wiring composes SemStreams input and processor component metadata, flowgraph edges, ports,
  payload registrations, config schema, requester, graph writer, raw lane, projector, adapter harness, and health
  state from config
- **AND** the same wiring is covered by tests without requiring the full container stack
- **AND** the hosted composition root registers COP ownership and passes registry-derived owner-token state into
  adapter wiring before any governed graph writes occur

#### Scenario: MAVLink live graph smoke precedes broad stack expansion

- **WHEN** the ADR-055/056 must-exist flip is pending in SemStreams
- **THEN** SemOps prioritizes a generated or replay MAVLink live graph smoke over PX4/SITL, UI, or second-feed
  expansion
- **AND** the smoke actively polls graph state and mutation failures rather than relying on quiet logs
- **AND** the clean-stack smoke registers COP ownership and uses registry-derived owner tokens
- **AND** the smoke asserts graph-ingest counter deltas for foreign-edge drops, owner-token mismatches, and
  indexing-profile defaults when a SemStreams metrics URL is provided

#### Scenario: Hosted product path proves live graph visibility

- **WHEN** the local Compose stack is used as the Phase 1 demo harness
- **THEN** the smoke sends generated MAVLink over the hosted SemOps UDP listener
- **AND** it sends CoT seed events over the hosted SemOps UDP listener
- **AND** it waits for the hosted scenario runner to report a succeeded HADR feed-boundary replay
- **AND** it waits for the Caddy-routed COP snapshot to expose the scenario's MAVLink, TAK/CoT, and CAP state from
  hosted component paths rather than scenario-runner direct graph writes
- **AND** it waits for the same-origin Caddy COP snapshot path to expose graph-backed MAVLink track state and
  TAK/CoT task/advisory state
- **AND** it can opt into a weather fixture check that waits for the Caddy-routed COP snapshot to expose graph-backed
  `weather_observation` evidence and component/runtime flow without enabling a default live weather provider
- **AND** it may run direct live graph smokes for MAVLink, TAK/CoT, and CAP contract evidence through SemStreams after
  product path checks pass
- **AND** it keeps the Svelte immutable asset cache check in place so Caddy remains a reverse proxy rather than a
  replacement static server

### Requirement: Scenario runner makes demos repeatable

SemOps SHALL include a scenario runner for deterministic HA/DR demo playback.

#### Scenario: In-process runner proves feed playback before service packaging

- **WHEN** the first HADR scenario fixture is replayed in tests
- **THEN** the runner sends generated MAVLink frames through the MAVLink adapter harness
- **AND** it sends deterministic TAK/CoT seed events through the CoT adapter harness
- **AND** it sends CAP lifecycle XML records through a feed-boundary or component path before CAP scenario output is
  cited as product e2e evidence
- **AND** direct CAP projector/graph-writer replay remains projection-contract evidence only
- **AND** it exposes a pollable run status with completed steps, failures, mutation counts, and last error
- **AND** missing feed sinks fail the run before it can report a false-green scenario result

#### Scenario: Flood and airspace vignette can replay

- **WHEN** the operator starts the Phase 1 scenario
- **THEN** the runner emits feed events for flood/evacuation, responder assets, drones, alerts, and COP updates

#### Scenario: Native feed replay has durable fixtures

- **WHEN** the scenario runner replays MAVLink input
- **THEN** it can load durable raw-frame fixture records rather than depending only on in-memory raw-lane state
- **AND** replay fixtures remain native-frame artifacts until projection writes current COP state

#### Scenario: Playback has a controllable clock

- **WHEN** the scenario is paused, resumed, accelerated, or reset
- **THEN** feed emission and expected COP state remain deterministic enough for tests and stage demos

### Requirement: Edge/core split is a later deployment mode

SemOps SHALL start with a compact single-stack demo and split edge/core placement only after structural behavior is
stable.

#### Scenario: Edge node runs structural behavior

- **WHEN** Phase 4 edge/core mode is enabled
- **THEN** edge services run feed boundaries, raw lanes, structural projection, and deterministic fusion

#### Scenario: Core node runs inference tiers

- **WHEN** Phase 4 edge/core mode is enabled
- **THEN** statistical and semantic services run in the core and write governed evidence back through SemStreams
