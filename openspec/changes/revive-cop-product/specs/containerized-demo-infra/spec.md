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

### Requirement: Scenario runner makes demos repeatable

SemOps SHALL include a scenario runner for deterministic HA/DR demo playback.

#### Scenario: Flood and airspace vignette can replay

- **WHEN** the operator starts the Phase 1 scenario
- **THEN** the runner emits feed events for flood/evacuation, responder assets, drones, alerts, and COP updates

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
