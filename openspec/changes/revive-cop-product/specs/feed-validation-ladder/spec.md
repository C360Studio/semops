## ADDED Requirements

### Requirement: Feeds enter the COP through explicit validation gates

Each feed SHALL pass documented validation gates before it is treated as a SemOps product capability.

#### Scenario: Feed has parser and replay evidence before graph writes

- **WHEN** a new feed is proposed for the COP
- **THEN** its native parser and deterministic replay evidence are documented before governed graph writes are added

#### Scenario: Feed order is explicit

- **WHEN** the revival plan sequences feed work
- **THEN** it starts with MAVLink, then TAK/CoT, then CAP/EDXL before ADS-B, SAPIENT, KLV, or CS API egress

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

### Requirement: Every projected entity declares an indexing profile

Every feed projection SHALL declare an expected SemStreams `indexing_profile` and cardinality risk before writing
canonical graph state.

#### Scenario: High-rate telemetry stays signal shaped

- **WHEN** MAVLink, ADS-B, TAK position, SAPIENT detection, or KLV sensor-position data is projected
- **THEN** current-state graph entities use `signal` and raw packet or replay detail uses bounded lanes or `trace`
  references

#### Scenario: Textual advisories stay content shaped

- **WHEN** CAP advisory text, operator notes, chat text, or semantic translations are projected
- **THEN** text intended for meaning-based retrieval uses `content` instead of being hidden inside high-rate signal
  entities

#### Scenario: Durable operational state stays control shaped

- **WHEN** tasks, alerts, commands, feed health, scenario state, or egress lifecycle state are projected
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
