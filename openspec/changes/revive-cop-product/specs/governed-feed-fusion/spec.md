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

#### Scenario: Fusion association projection preserves source ownership

- **WHEN** statistical track association produces derived evidence for two source-owned tracks
- **THEN** SemOps births a fusion-owned `association` entity with `control` indexing before later updates
- **AND** the create plan writes association status, confidence, algorithm identity, distance/time evidence,
  provenance, and strict primary/candidate track edges through SemStreams mutation request contracts
- **AND** update plans refresh association evidence without re-declaring strict source-track edges or mutating either
  source track entity
- **AND** COP readback discovers association entities by fusion prefix and exposes the evidence separately from track
  current state

#### Scenario: Fusion association processor uses SemStreams lifecycle

- **WHEN** SemOps hosts statistical track association
- **THEN** it runs as an opt-in SemStreams processor component with a declared NATS input port for bounded
  track-candidate batches and declared graph mutation request output ports
- **AND** the processor uses the payload registry to decode candidate batches before scoring or graph projection
- **AND** graph writes use the fusion owner token already minted by the COP ownership registration path
- **AND** entity-exists responses reconcile born state and reproject updates instead of falling back to auto-vivify or
  raw `triple.add` writes
- **AND** the component exposes `Health()` and `DataFlow()` so runtime telemetry can report throughput, errors, and
  last activity
- **AND** this hosted processor does not itself discover candidate tracks, merge identities, mutate source tracks, or
  expose operator merge controls until separate gates approve those workflows

#### Scenario: Fusion candidate producer uses bounded graph discovery

- **WHEN** SemOps produces statistical track-association candidates from governed graph state
- **THEN** it runs as an opt-in SemStreams processor component with a declared timer input port, graph prefix query
  request output port, and `semops.fusion.track_candidates` output stream port
- **AND** the producer queries only configured source-owned track prefixes and caps tracks per source, comparisons per
  batch, and batches per scan before publishing candidate payloads
- **AND** candidate batches use the SemStreams payload registry and preserve source track IDs, native IDs, positions,
  observed times, confidence, and source references for the downstream association processor
- **AND** source pairs are generated in a deterministic one-way order so the same association is not published with
  primary and candidate roles reversed during the same scan
- **AND** the producer exposes `Health()` and `DataFlow()` so runtime telemetry can report throughput, errors, and
  last activity
- **AND** the producer does not write graph state, mutate source tracks, merge identities, or enable default automatic
  demo association until adversarial operator review approves that posture

#### Scenario: Stack smoke proves hosted fusion association before default enablement

- **WHEN** the one-command COP stack smoke is run with `SEMOPS_COP_SMOKE_FUSION_ENABLED=true`
- **THEN** the hosted SemOps process enables the fusion candidate producer and fusion association projector as
  SemStreams lifecycle components
- **AND** the smoke seeds close-but-separate MAVLink and TAK/CoT source-owned tracks through their transport
  components rather than publishing fusion candidate subjects directly
- **AND** the candidate producer discovers those tracks through SemStreams graph prefix-query request ports and
  publishes bounded registered candidate batches
- **AND** the association projector consumes those batches, writes fusion-owned born-first association graph state,
  and exposes runtime and Prometheus flow telemetry
- **AND** the COP snapshot readback exposes a fusion association between the two source tracks without mutating either
  source-owned track
- **AND** the default stack keeps automatic demo association disabled until row-level UI language, mission thresholds,
  ambiguity policy, and operator acknowledge/challenge affordances are default-safe

#### Scenario: Hosted association scoring uses mission-profile config

- **WHEN** hosted fusion association is enabled
- **THEN** SemOps can configure maximum source-track distance, maximum source-track observation time delta, minimum
  confidence, ambiguity margin, maximum source-track observation age, and source-priority ordering without changing
  code
- **AND** invalid or unsafe scoring values fail startup before the fusion projector subscribes
- **AND** the fusion projector component exposes those scoring settings through its SemStreams config schema
- **AND** source priority is a deterministic tie-breaker for equal-score candidates rather than a replacement for
  geotemporal confidence
- **AND** stale-window filtering uses the hosted component's clock as the reference time
- **AND** operator review remains separate from scoring so acknowledge/challenge decisions do not change source-track
  state or association confidence

#### Scenario: Operator reviews association evidence without identity authority

- **WHEN** the COP snapshot contains fusion association evidence
- **THEN** the SemOps API accepts operator review decisions for `acknowledged` and `challenged`
- **AND** the API rejects unknown association IDs and unsupported review decisions
- **AND** the COP snapshot overlays the current operator review beside the association evidence
- **AND** the COP UI exposes acknowledge/challenge controls from the association inspector
- **AND** the API resolves the reviewer's audit label from `X-SemOps-Operator-ID`, then request-body `reviewed_by`,
  then `operator.local`
- **AND** the COP UI lets the operator set that local audit label without requiring an authentication provider
- **AND** operator review records expose `reviewer_role = operator.unverified`,
  `authority_scope = local.display_only`, and `conflict_policy = latest_review_wins_display_only`
- **AND** `X-SemOps-Operator-Role` and `X-SemOps-Authority-Scope` cannot escalate review records beyond those
  display-only semantics during the MVP
- **AND** hosted SemOps writes operator review as a fusion-owned `association_review` graph audit entity with a strict
  edge to the reviewed association evidence
- **AND** the graph audit entity owns the review decision, reviewer, reviewed time, reviewer role, authority scope,
  conflict policy, comment, and provenance predicates under the fusion owner
- **AND** the graph-backed COP snapshot discovers association-review audit state and overlays it on association
  evidence readback
- **AND** operator review state does not mutate source-owned tracks, merge identities, change the association status, or
  give feed adapters association authority
- **AND** fixture-only API mode may use a local memory overlay, but hosted review state must use the graph-backed audit
  path before review decisions can become command, identity, upstream CS API status, or compliance authority
- **AND** local MVP operator identity is not authentication and does not satisfy authenticated authority semantics
- **AND** SemOps may enable trusted-header operator identity only when an upstream authentication boundary owns
  `X-SemOps-Operator-Authenticated`, `X-SemOps-Operator-ID`, `X-SemOps-Operator-Role`,
  `X-SemOps-Authority-Scope`, and `X-SemOps-Authority-Domain`
- **AND** authenticated association-review records use `reviewer_role = operator.authenticated`,
  `authority_scope = association.review`, an explicit authority domain, and
  `conflict_policy = multi_authority_blocks_conflicts`
- **AND** matching authenticated decisions across authority domains may produce a consensus current review
- **AND** conflicting authenticated decisions across authority domains SHALL produce `decision = blocked_conflict`
  and `conflict_state = blocked_conflict` rather than letting latest-writer or local display review win
- **AND** display-only reviews cannot overwrite an authenticated authority-domain review for the same association
- **AND** `blocked_conflict` remains a hard stop for command execution, identity fusion, upstream CS API status, and
  compliance workflows until those workflows have their own authority tests

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

#### Scenario: CS API read-side egress stays standards-facing

- **WHEN** an external system needs to consume governed SemOps COP state through OGC Connected Systems API resources
- **THEN** SemOps SHALL prioritize read-side egress through a CS API gateway or projection boundary
- **AND** feature resources such as Systems, Procedures, Deployments, Sampling Features, Subsystems/Components, and
  Property Definitions map through curated COP projections rather than raw feed payloads
- **AND** dynamic read-side resources such as Datastreams, Observations, System Events, streaming, and snapshots map
  through ownership-aware state and evidence boundaries
- **AND** write-side ingress, ControlStreams, Commands, and Command Status remain stretch scope until command authority,
  TTL, priority, idempotency, local override, cancellation, and native safety gates are deliberately reopened
- **AND** native adapters for MAVLink, TAK/CoT, CAP, weather, DJI, ADS-B, SAPIENT, and KLV remain first-class paths
  when their native protocols carry product-critical semantics
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

- **WHEN** KLV, DJI video, imagery, video, or keyframe evidence is available to a feed adapter
- **THEN** graph state contains metadata and storage references instead of raw binary payloads

#### Scenario: Weather is split by operational shape

- **WHEN** weather enters the COP
- **THEN** alert/advisory evidence may reuse CAP-style append-evidence contracts
- **AND** tactical weather observations or forecasts are projected as bounded point, area, trajectory, or corridor
  evidence with provenance and freshness
- **AND** tactical weather variable/time samples are `signal`-profiled `weather_observation` entities rather than
  CAP hazard, route-decision, task, alert, or advisory authority
- **AND** graph-writing weather components use declared SemStreams graph request ports and per-payload caps rather
  than publishing directly to graph mutation subjects from transport or decode stages
- **AND** visual raster or tile layers may be browser-only unless an operator workflow requires graph state

#### Scenario: DJI video does not redefine KLV semantics

- **WHEN** DJI telemetry, media references, subtitles, or live video feeds enter SemOps
- **THEN** DJI-specific metadata and control semantics stay in a DJI feed boundary
- **AND** KLV/MISB decode remains a KLV/STANAG worker concern
- **AND** shared media infrastructure may provide generic media references or track extraction without claiming DJI or
  KLV product support

### Requirement: First canonical COP entities are stable

The first structural slice SHALL use a small canonical model before feed-specific expansion.

#### Scenario: First entity set covers the structural COP

- **WHEN** Phase 1 adapters project data
- **THEN** they target tracks, assets, hazard areas, sensor footprints, alerts, tasks, or advisories

#### Scenario: Feed-specific detail remains attached to source evidence

- **WHEN** a native protocol has fields outside the canonical model
- **THEN** those fields remain in source-specific evidence or raw references unless the COP needs them directly
