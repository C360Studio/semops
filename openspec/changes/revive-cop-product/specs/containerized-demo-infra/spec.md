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

#### Scenario: Feed adapters are processor components

- **WHEN** SemOps decodes, validates, replays, maps, fuses, or projects native feed data
- **THEN** that behavior runs inside SemStreams processor components that subscribe to declared input ports and publish
  declared output ports
- **AND** parser or decoder processors emit decoded feed messages that other components can tap
- **AND** projection processors write governed graph mutations only through declared SemStreams NATS request ports

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

### Requirement: Structural demo stack starts with one command

The first COP stack SHALL run locally with a single documented command after dependencies are installed.

#### Scenario: Phase 1 stack is healthy

- **WHEN** the structural stack starts
- **THEN** NATS, SemStreams, SemOps API, SemOps UI, scenario runner, and Phase 1 adapters expose health state
- **AND** SemStreams health probes use a dedicated non-conflicting listener port rather than racing the service-manager
  HTTP port

#### Scenario: Hosted scenario runner reports concrete playback state

- **WHEN** the local Compose stack starts `semops-scenario-runner`
- **THEN** the service replays the first HADR scenario into the live SemStreams graph on startup
- **AND** `/healthz` remains unavailable until playback succeeds
- **AND** `/scenario/status` reports the scenario id, state, completed steps, failed steps, mutation count, and last
  error
- **AND** the one-command smoke polls the status endpoint rather than inferring scenario success from logs

#### Scenario: Hosted scenario playback is product-visible

- **WHEN** `semops-scenario-runner` reports succeeded in the local Compose stack
- **THEN** the Caddy-routed COP snapshot contains the scenario MAVLink track
- **AND** the snapshot contains the scenario TAK/CoT task and advisory
- **AND** the snapshot contains the scenario CAP hazard with geometry and provenance
- **AND** this check runs before direct feed-specific live graph smokes so product visibility fails fast

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
- **AND** it waits for the hosted scenario runner to report a succeeded HADR replay
- **AND** it waits for the Caddy-routed COP snapshot to expose the scenario runner's MAVLink, TAK/CoT, and CAP state
- **AND** it waits for the same-origin Caddy COP snapshot path to expose graph-backed MAVLink track state and
  TAK/CoT task/advisory state
- **AND** it runs direct live graph smokes for MAVLink, TAK/CoT, and CAP evidence through SemStreams
- **AND** it keeps the Svelte immutable asset cache check in place so Caddy remains a reverse proxy rather than a
  replacement static server

### Requirement: Scenario runner makes demos repeatable

SemOps SHALL include a scenario runner for deterministic HA/DR demo playback.

#### Scenario: In-process runner proves feed playback before service packaging

- **WHEN** the first HADR scenario fixture is replayed in tests
- **THEN** the runner sends generated MAVLink frames through the MAVLink adapter harness
- **AND** it sends deterministic TAK/CoT seed events through the CoT adapter harness
- **AND** it sends CAP lifecycle XML records through the CAP projector and graph writer
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
