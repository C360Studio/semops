## ADDED Requirements

### Requirement: Feed adapters canonicalize native data at the boundary

Each feed adapter SHALL decode, validate or tolerate, and project native messages into canonical COP state.

#### Scenario: Strict feed rejects malformed native messages

- **WHEN** a strict feed such as MAVLink, SAPIENT, CoT, or OGC receives malformed native data
- **THEN** the adapter rejects the malformed message before it reaches governed graph state

#### Scenario: Loose feed writes best-effort evidence

- **WHEN** a loose feed such as CAP/EDXL or raw ADS-B has partial but usable data
- **THEN** the adapter writes best-effort governed evidence with provenance and confidence

### Requirement: Ownership prevents silent clobbering

Feed projections MUST use SemStreams ownership contracts so one producer cannot silently replace another producer's
authoritative predicates.

#### Scenario: Civilian alert does not overwrite authoritative track facts

- **WHEN** a CAP hazard or advisory references a vehicle, unit, or infrastructure asset also known by a strict feed
- **THEN** CAP evidence is appended or written under CAP-owned predicates instead of replacing strict feed state

#### Scenario: Fusion writes derived facts as a separate owner

- **WHEN** deterministic fusion correlates two source facts or raises an alert
- **THEN** the fusion owner writes derived predicates or evidence without hiding the original source facts

### Requirement: Raw feed data stays on bounded lanes

SemOps SHALL keep high-rate raw messages out of canonical graph entities unless a specific raw artifact must be
preserved as evidence.

#### Scenario: MAVLink raw frames are not graph entities

- **WHEN** MAVLink telemetry arrives at high frequency
- **THEN** raw frames remain on a bounded stream lane and current vehicle state is projected into signal-profiled
  graph entities

#### Scenario: Raw evidence can be linked when needed

- **WHEN** a QA, provenance, or replay workflow needs the native payload
- **THEN** the governed projection includes a source reference instead of copying every raw packet into the graph

#### Scenario: Binary media is referenced instead of embedded

- **WHEN** KLV, imagery, video, or keyframe evidence is available to a feed adapter
- **THEN** graph state contains metadata and storage references instead of raw binary payloads

### Requirement: First canonical COP entities are stable

The first structural slice SHALL use a small canonical model before feed-specific expansion.

#### Scenario: First entity set covers the structural COP

- **WHEN** Phase 1 adapters project data
- **THEN** they target tracks, assets, hazard areas, sensor footprints, alerts, tasks, or advisories

#### Scenario: Feed-specific detail remains attached to source evidence

- **WHEN** a native protocol has fields outside the canonical model
- **THEN** those fields remain in source-specific evidence or raw references unless the COP needs them directly
