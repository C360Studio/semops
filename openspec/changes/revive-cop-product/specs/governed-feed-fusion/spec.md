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

### Requirement: Graph writes are born-first

SemOps adapters SHALL follow SemStreams ADR-055 and ADR-056. Entity creation must happen through typed graph birth
requests before triples or foreign edges are added.

#### Scenario: Adapter births entity before current-state update

- **WHEN** a feed adapter sees a new track, asset, hazard area, alert, task, advisory, or sensor footprint
- **THEN** it first creates the entity with `graph.CreateEntityWithTriplesRequest`, `MessageType`, and
  `IndexingProfile`
- **AND** later updates use `graph.UpdateEntityWithTriplesRequest` against an existing entity

#### Scenario: Adapter does not rely on auto-vivify

- **WHEN** a SemOps adapter wants to add triples for an entity that has not been born
- **THEN** the adapter must fail or birth the entity explicitly instead of relying on `triple.add` or
  `triple.add_batch` auto-vivify

#### Scenario: Graph writer uses SemStreams mutation request/reply contracts

- **WHEN** a projection plan is ready to commit governed graph state
- **THEN** creates are sent to `graph.mutation.entity.create_with_triples`
- **AND** updates are sent to `graph.mutation.entity.update_with_triples`
- **AND** mutation failures stop later writes in the plan
- **AND** committed-but-degraded SemStreams mutation responses are treated as committed writes, not retry prompts

#### Scenario: Foreign edges are declared through ADR-056

- **WHEN** a projection writes a relationship onto a different entity than the one it owns
- **THEN** its projection contract declares a foreign edge that derives a SemStreams `ForeignEdgeClaim` with producer,
  edge mode, predicate, and target pattern

#### Scenario: Strict source edge requires born target

- **WHEN** MAVLink or TAK projects a `cop.track.source` edge to an asset
- **THEN** the source asset is born first, and the edge contract uses `EdgeStrict` rather than implicit target creation

#### Scenario: Must-exist compliance is proven before feed-fidelity expansion

- **WHEN** SemOps prepares for a SemStreams ADR-055/056 must-exist breaking tag
- **THEN** generated or replay MAVLink frames must prove source asset birth, track birth, and update behavior against
  a live SemStreams graph path before PX4/SITL fidelity work is treated as blocking
- **AND** the smoke reports no `entity_not_found` failures for the MAVLink path
- **AND** clean-stack evidence asserts dropped foreign-edge and indexing-profile default counters when those signals
  are exposed by the SemStreams tag in use

#### Scenario: No-birth stub requires explicit review

- **WHEN** a SemOps projection has no independent producer for a relationship target
- **THEN** it may use `EdgeNoBirthStub` only after an adversarial review records why born-first is impossible

### Requirement: Raw feed data stays on bounded lanes

SemOps SHALL keep high-rate raw messages out of canonical graph entities unless a specific raw artifact must be
preserved as evidence.

#### Scenario: MAVLink raw frames are not graph entities

- **WHEN** MAVLink telemetry arrives at high frequency
- **THEN** raw frames remain on a bounded stream lane and current vehicle state is projected into signal-profiled
  graph entities

#### Scenario: Raw evidence can be linked when needed

- **WHEN** a QA, provenance, or replay workflow needs the native payload
- **THEN** the governed projection includes a `cop.provenance.source_ref` or equivalent source reference instead of
  copying every raw packet into the graph

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
