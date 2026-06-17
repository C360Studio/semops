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

### Requirement: Structural demo stack starts with one command

The first COP stack SHALL run locally with a single documented command after dependencies are installed.

#### Scenario: Phase 1 stack is healthy

- **WHEN** the structural stack starts
- **THEN** NATS, SemStreams, SemOps API, SemOps UI, scenario runner, and Phase 1 adapters expose health state

#### Scenario: Health checks use active polling

- **WHEN** a long-running demo or paid operation is supervised
- **THEN** SemOps polls authoritative health, graph, and scenario state instead of relying only on passive logs

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
- **THEN** the structural wiring composes SemStreams requester, graph writer, raw lane, projector, adapter harness,
  and health state from config
- **AND** the same wiring is covered by tests without requiring the full container stack

#### Scenario: MAVLink live graph smoke precedes broad stack expansion

- **WHEN** the ADR-055/056 must-exist flip is pending in SemStreams
- **THEN** SemOps prioritizes a generated or replay MAVLink live graph smoke over PX4/SITL, UI, or second-feed
  expansion
- **AND** the smoke actively polls graph state and mutation failures rather than relying on quiet logs
- **AND** the later clean-stack smoke asserts graph-ingest counters for foreign-edge drops and indexing-profile
  defaults

### Requirement: Scenario runner makes demos repeatable

SemOps SHALL include a scenario runner for deterministic HA/DR demo playback.

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
