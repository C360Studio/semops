## ADDED Requirements

### Requirement: Feeds enter the COP through explicit validation gates

Each feed SHALL pass documented validation gates before it is treated as a SemOps product capability.

#### Scenario: Feed has parser and replay evidence before graph writes

- **WHEN** a new feed is proposed for the COP
- **THEN** its native parser and deterministic replay evidence are documented before governed graph writes are added

#### Scenario: Feed order is explicit

- **WHEN** the revival plan sequences feed work
- **THEN** it starts with MAVLink, then TAK/CoT, then CAP/EDXL before ADS-B, SAPIENT, KLV, or CS API interop

### Requirement: Feed roadmaps distinguish demo scope from full product scope

SemOps SHALL track the narrow demo path and the eventual full-product path for every feed so Phase 1 shortcuts do not
become dead-end architecture.

#### Scenario: Feed has two-lane roadmap before implementation

- **WHEN** SemOps starts or broadens a feed adapter
- **THEN** the feed record identifies the demo/MVP lane, the full product lane, the preserved boundary, and the claims
  that are still out of scope

#### Scenario: TAK Server is future service scope

- **WHEN** SemOps adds TAK/CoT support for the MVP demo
- **THEN** local CoT parsing and fixture replay remain separate from projection logic
- **AND** a SemStreams-backed SemOps TAK service is tracked as future product scope rather than an MVP service

#### Scenario: Feed server capabilities preserve service seams

- **WHEN** a feed's full product shape requires server, gateway, collaboration, session, or federation behavior
- **THEN** the MVP adapter keeps parser, transport, service state, and graph projection seams separate so SemOps can
  later promote that feed into a SemStreams-backed service without rewriting the governed projection contract

#### Scenario: Service promotion requires product forces

- **WHEN** SemOps proposes promoting a feed adapter into a SemOps-owned service or gateway
- **THEN** the proposal identifies the product force: external protocol exposure, auth/session/federation state,
  bidirectional command or tasking, scaling or placement isolation, durable collaboration or replay state, secrets,
  cost, or failure-domain isolation
- **AND** the promotion preserves the existing governed projection contract rather than moving ownership decisions
  into transport glue

### Requirement: Compliance claims require reproducible evidence

SemOps SHALL NOT claim protocol or standards conformance unless a reproducible local harness, official schema,
public conformance suite, or documented interoperability test backs the claim.

#### Scenario: Public compliance suite is available

- **WHEN** a public compliance suite or official schema exists for a feed
- **THEN** SemOps records how to run it or records why it is out of scope for the current phase

#### Scenario: Public compliance suite is not verified

- **WHEN** no public compliance suite is verified for a feed such as TAK/CoT or SAPIENT
- **THEN** SemOps records the gap and uses mock, replay, schema, or interoperability evidence without calling it
  conformance

#### Scenario: CS API conformance is standards-edge evidence

- **WHEN** SemOps maps governed COP state through OGC Connected Systems API
- **THEN** conformance evidence comes from SemConnect, an official schema, an official or accepted ETS, or a
  documented interoperability run
- **AND** the conformance result does not imply native MAVLink, TAK/CoT, CAP, ADS-B, SAPIENT, or KLV conformance

### Requirement: Every projected entity declares an indexing profile

Every feed projection SHALL declare an expected SemStreams `indexing_profile` and cardinality risk before writing
canonical graph state.

#### Scenario: High-rate telemetry stays signal shaped

- **WHEN** MAVLink, ADS-B, TAK position, SAPIENT detection, or KLV sensor-position data is projected
- **THEN** current-state graph entities use `signal` and raw packet or replay detail uses bounded lanes or `trace`
  references

#### Scenario: ADS-B starts with OpenSky fixture parsing and source-partitioned projection

- **WHEN** SemOps admits ADS-B into the feed ladder
- **THEN** the first executable gate parses bounded OpenSky-shaped state-vector fixtures before projection
- **AND** nullable callsign, position timestamp, longitude, latitude, altitude, velocity, track, vertical rate,
  receiver IDs, squawk, position source, and category fields remain explicit
- **AND** aircraft current state projects to source-partitioned ADS-B `track` entities with `signal` indexing,
  provenance, confidence, and source references
- **AND** deterministic OpenSky snapshot records can replay through the scenario runner without live network access
- **AND** a hosted snapshot-ingest seam captures bounded raw snapshots, appends replay, writes projection plans, and
  reports health without requiring live OpenSky
- **AND** structural scenario replay is opt-in, uses the hosted ADS-B adapter seam, and writes graph mutations only
  after `semops.feed.adsb` has a SemStreams-minted owner token
- **AND** COP API prefix discovery can read those aircraft tracks back from the graph
- **AND** live OpenSky polling, ASTERIX, raw receiver protocols, and cross-source aircraft association remain out of
  scope until separate gates approve them

#### Scenario: Textual advisories stay content shaped

- **WHEN** CAP advisory text, operator notes, chat text, or semantic translations are projected
- **THEN** text intended for meaning-based retrieval uses `content` instead of being hidden inside high-rate signal
  entities

#### Scenario: Durable operational state stays control shaped

- **WHEN** tasks, alerts, commands, feed health, scenario state, or standards bridge lifecycle state are projected
- **THEN** those entities use `control` unless their storage shape proves they are high-cardinality trace data

### Requirement: KLV remains a binary proof spike until proven

SemOps SHALL treat KLV/STANAG 4609 as a proof spike until small fixtures prove metadata extraction,
binary-by-reference storage, and memory-bounded handling.

#### Scenario: Binary payload is not written into graph triples

- **WHEN** a video, keyframe, or KLV payload is ingested
- **THEN** graph state contains metadata and storage references, not raw binary bytes

#### Scenario: Streaming-binary claim is blocked by memory-bound evidence

- **WHEN** binary storage requires reading whole video files into memory
- **THEN** SemOps does not claim streaming-binary support until the implementation is replaced or bounded by tests
