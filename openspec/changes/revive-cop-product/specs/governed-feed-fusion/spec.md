## ADDED Requirements

### Requirement: Feed adapters canonicalize native data at the boundary

Each feed adapter SHALL decode, validate or tolerate, and project native messages into canonical COP state.

#### Scenario: Strict feed rejects malformed native messages

- **WHEN** a strict feed such as MAVLink, SAPIENT, CoT, or OGC receives malformed native data
- **THEN** the adapter rejects the malformed message before it reaches governed graph state

#### Scenario: Loose feed writes best-effort evidence

- **WHEN** a loose feed such as CAP evidence or raw ADS-B has partial but usable data
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

#### Scenario: Hosted adapters use registry-derived owner tokens

- **WHEN** a SemOps process hosts a governed feed adapter
- **THEN** the process registers the feed's projection contracts through SemStreams ownership before composing the
  adapter
- **AND** owner-token state is derived from the registry or bind result rather than static operator configuration

#### Scenario: Strict source edge requires born target

- **WHEN** MAVLink or TAK projects a `cop.track.source` edge to an asset
- **THEN** the source asset is born first, and the edge contract uses `EdgeStrict` rather than implicit target creation

#### Scenario: TAK projection separates signal, control, and content state

- **WHEN** TAK/CoT operator dots, air tracks, markers, and GeoChat messages are projected
- **THEN** operator dots and air tracks become `signal` track state
- **AND** markers and task-like map control state become `control` task state
- **AND** GeoChat text becomes `content` advisory state
- **AND** native CoT XML remains on bounded raw or replay lanes referenced by provenance source refs

#### Scenario: TAK readback proves graph state before broader feed expansion

- **WHEN** the hosted SemOps stack receives seed CoT events over the configured UDP listener
- **THEN** the live COP snapshot reads TAK/CoT track, task, and advisory entities back from SemStreams graph state
- **AND** feed health reflects live or stale state from graph observation timestamps
- **AND** the result remains a seed-UID readback gate, not a claim of TAK Server-equivalent behavior

#### Scenario: ADS-B projection does not own aircraft association

- **WHEN** an ADS-B-shaped state vector is projected into SemStreams graph state
- **THEN** SemOps births a source-partitioned ADS-B `track` entity with `signal` indexing before update-only writes
- **AND** nullable position data stays partial evidence instead of fake coordinates
- **AND** position-source quality contributes provenance/confidence evidence
- **AND** the ADS-B adapter does not emit `cop.track.source` or cross-source aircraft association edges
- **AND** deterministic snapshot replay can drive parse, projection, graph-plan writing, and born-state marking
- **AND** hosted snapshot ingest preserves bounded raw refs, replay append, health counters, and born-first write
  reconciliation before any live-source promotion
- **AND** the hosted app can opt into the OpenSky-compatible HTTP poller -> decoder -> projector component flow with
  `semops.feed.adsb` ownership minted only for that enabled runtime
- **AND** the COP snapshot may read ADS-B aircraft tracks by prefix discovery while default live provider enablement,
  receiver protocols, ASTERIX, and statistical association remain separate gates

#### Scenario: COP readback uses SemStreams discovery primitives

- **WHEN** SemOps needs to hydrate the operator snapshot from graph state
- **THEN** it uses SemStreams `graph.query.prefix` to discover source-partitioned COP entities by 5-part graph prefix
- **AND** configured seed IDs are compatibility fallback, not the product-grade discovery model
- **AND** SemOps does not bypass SemStreams with direct KV scans for normal browser readback

#### Scenario: CAP evidence readback stays append-only

- **WHEN** a CAP alert fixture is parsed and projected into SemStreams graph state
- **THEN** SemOps births a source-partitioned `hazard_area` entity before appending CAP advisory and geometry
  evidence
- **AND** the projection uses the CAP append-evidence contract without claiming authoritative
  `cop.hazard.geometry`, `cop.hazard.severity`, or `cop.hazard.status`
- **AND** the COP snapshot may render the evidence as a hazard overlay while preserving provenance and source
  confidence
- **AND** the result remains a parser/projection/readback gate, not a claim of hosted CAP polling, NWS integration,
  or full CAP consumer conformance

#### Scenario: CS API interop remains bidirectional and standards-facing

- **WHEN** an external system can publish or consume OGC Connected Systems API resources
- **THEN** SemOps SHALL preserve a path to map that state into or out of the governed COP model through a CS API
  gateway or projection boundary
- **AND** feature resources such as Systems, Procedures, Deployments, Sampling Features, Subsystems/Components, and
  Property Definitions map through curated COP projections rather than raw feed payloads
- **AND** dynamic resources such as Datastreams, Observations, ControlStreams, Commands, Command Status, System
  Events, streaming, and snapshots map through ownership-aware state, evidence, and command-authority boundaries
- **AND** command ingress is asynchronous: the CS API boundary records governed intent or desired state with TTL,
  priority, authority, idempotency, and cancellation semantics before native drivers reconcile actual execution
- **AND** native adapters for MAVLink, TAK/CoT, CAP, ADS-B, SAPIENT, and KLV remain first-class paths when their
  native protocols carry product-critical semantics
- **AND** CS API interop MUST preserve SemStreams ownership, provenance, freshness, and indexing-profile decisions
  rather than becoming a bypass around governed feed fusion

#### Scenario: Must-exist compliance is proven before feed-fidelity expansion

- **WHEN** SemOps prepares for a SemStreams ADR-055/056 must-exist breaking tag
- **THEN** generated or replay MAVLink frames must prove source asset birth, track birth, and update behavior against
  a live SemStreams graph path before PX4/SITL fidelity work is treated as blocking
- **AND** the smoke reports no `entity_not_found` failures for the MAVLink path
- **AND** clean-stack evidence registers COP owners and uses registry-derived owner tokens
- **AND** metrics-enabled evidence asserts dropped foreign-edge, owner-token mismatch, and indexing-profile default
  counter deltas for SemOps message types when those signals are exposed by the SemStreams tag in use

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
