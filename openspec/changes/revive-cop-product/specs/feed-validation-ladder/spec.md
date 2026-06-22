## ADDED Requirements

### Requirement: Feeds enter the COP through explicit validation gates

Each feed SHALL pass documented validation gates before it is treated as a SemOps product capability.

#### Scenario: Feed has parser and replay evidence before graph writes

- **WHEN** a new feed is proposed for the COP
- **THEN** its native parser and deterministic replay evidence are documented before governed graph writes are added

#### Scenario: Feed order is explicit

- **WHEN** the revival plan sequences feed work
- **THEN** it starts with MAVLink, then TAK/CoT, then CAP before weather, DJI, ADS-B, SAPIENT, KLV, or CS API interop
- **AND** broader EDXL formats beyond CAP require separate product need, fixture, projection, and review gates before
  they enter Phase 1 scope

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

#### Scenario: CAP replay is not hosted CAP service evidence

- **WHEN** CAP lifecycle fixtures replay through parser, projector, scenario runner, or direct graph smoke paths
- **THEN** SemOps treats the result as parser/projection/readback evidence
- **AND** the first CAP HTTP poller, decoder, and projector components are treated as component-contract evidence, not
  default live NWS/IPAWS service evidence
- **AND** component health can report stale CAP polling when no fresh payload arrives within the configured
  `stale_after` threshold
- **AND** the opt-in runtime can capture provider-shaped CAP HTTP responses as replayable native CAP XML records
- **AND** SemOps does not claim hosted CAP polling, webhook, NWS/IPAWS integration, or alert feed service support until
  opt-in runtime wiring is backed by captured provider samples and alert lifecycle gates for that boundary

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

#### Scenario: Public compliance suite is verified but not yet run locally

- **WHEN** a feed such as SAPIENT has an official schema, sample corpus, or compliance harness available
- **THEN** SemOps records the authoritative sources, runtime constraints, and license boundary
- **AND** SemOps does not claim product support or conformance until a local parser gate and scoped harness result
  exist

#### Scenario: SAPIENT parser preflight is not conformance

- **WHEN** SemOps parses BSI Flex 335 v2 JSON fixtures derived from official SAPIENT sample shapes
- **THEN** the parser may be used as developer preflight evidence before graph projection
- **AND** SemOps may add descriptor-based binary protobuf preflight from official proto source without claiming
  full-message coverage, portable-suite authority, product support, or SAPIENT compliance
- **AND** SemOps may run SAPIENT raw ingress and decoder preflight as SemStreams input and processor components only
  when they publish raw/decoded streams and avoid graph mutation ports, owner claims, and product support wording
- **AND** the hosted app may compose that HTTP input -> decoder chain behind `SEMOPS_SAPIENT_ENABLED=true` only when
  URL, encoding, replay, and stale-source settings remain explicit preflight configuration
- **AND** SemOps does not claim compliance until scoped Dstl harness evidence exists
- **AND** SemOps does not add SAPIENT owner claims, graph projection, graph-producing hosted components, or product
  service support until projection ownership, indexing, service mode, and harness scope have been reviewed

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

#### Scenario: SAPIENT projection starts narrow

- **WHEN** SemOps adds a SAPIENT graph projector
- **THEN** the first projection uses an accepted SAPIENT source identity, entity model, ownership contract, and
  indexing profile
- **AND** absolute-location reports are handled before range/bearing detections unless sensor pose, reference frame,
  and uncertainty are available
- **AND** associated detections and cross-source links are expressed by fusion or evidence contracts rather than the
  SAPIENT adapter's source-owner contract

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
- **AND** the first OpenSky-compatible HTTP hosted ingress is expressed as SemStreams input and processor components
  with `HTTPClientPort`, `TimerPort`, payload-registry, stream-port, graph-request-port, health, and flow metrics
  coverage
- **AND** the hosted app can opt into that HTTP component chain with `SEMOPS_ADSB_ENABLED=true`, config-driven replay
  capture, raw-lane caps, stale-source settings, and `semops.feed.adsb` ownership registration
- **AND** structural scenario replay is opt-in, uses the hosted ADS-B adapter seam, and writes graph mutations only
  after `semops.feed.adsb` has a SemStreams-minted owner token
- **AND** COP API prefix discovery can read those aircraft tracks back from the graph
- **AND** default live OpenSky enablement, provider reliability, ASTERIX, raw receiver protocols, and cross-source
  aircraft association remain out of scope until separate gates approve them

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

#### Scenario: Synthetic binary fixture is labeled honestly

- **WHEN** SemSource or a SemOps sidecar uses a synthetic binary fixture before a real legal KLV/SKG sample exists
- **THEN** the fixture is labeled as storage/governance proof only
- **AND** it proves binary-by-reference, governed metadata, indexing profile, and memory-bound behavior
- **AND** it is not cited as KLV, STANAG 4609, SAPIENT, SKG, streaming-binary, or protocol conformance evidence

#### Scenario: KLV and STANAG behavior remains a SemOps product concern

- **WHEN** SemSource provides storage references or governed metadata for opaque binary artifacts
- **THEN** SemOps owns KLV/MISB/STANAG parser choice, derived-fact projection, and conformance claims
- **AND** SemSource substrate evidence is not treated as feed-specific protocol support

#### Scenario: KLV demux remains a SemOps product boundary for MVP

- **WHEN** SemOps consumes SemSource media references or native media ingress
- **THEN** the MVP demuxes KLV in a SemOps-owned component or sidecar
- **AND** future SemSource media-track extraction is treated as generic substrate support, not SemOps KLV/STANAG
  product support

#### Scenario: DJI media reinforces generic media references

- **WHEN** DJI video or recorded media enters SemOps
- **THEN** the media path uses generic media references and bounded metadata extraction where possible
- **AND** DJI telemetry, subtitles, or vendor metadata are not forced through the KLV/MISB decoder unless the source
  actually emits KLV
- **AND** the existence of DJI video does not move KLV/STANAG product claims into SemSource

#### Scenario: Weather layers are tracked separately

- **WHEN** SemOps adds weather to the COP
- **THEN** visual weather tiles, CAP/public alerts, and tactical weather telemetry have separate evidence gates
- **AND** tactical weather supports point, area, and route/trajectory query shapes before it influences routing or
  safety logic

#### Scenario: Public samples are smoke evidence only

- **WHEN** SemOps uses a public video-plus-KLV sample such as a widely circulated MPEG-TS KLV file
- **THEN** the sample has documented license and provenance before use
- **AND** the result is labeled as demux/parser smoke evidence, not deterministic correctness or conformance

#### Scenario: Deterministic KLV fixture proves engineering support

- **WHEN** SemOps claims MISB ST 0601 engineering support
- **THEN** a deterministic fixture traces truth JSON through encoded KLV and optional MPEG-TS wrapping to parsed output
- **AND** acceptance asserts the parsed field set and parsed numeric values against the original truth data within MISB
  integer quantization tolerances for the supported field subset
- **AND** MPEG-TS wrapping is generated locally from the truth fixture and skipped when FFmpeg tooling is unavailable,
  instead of vendoring large binary media into the repo

#### Scenario: First KLV parser spike is deterministic and local

- **WHEN** SemOps chooses the first KLV/MISB parser and demux strategy
- **THEN** the first spike uses a Go-native MISB ST 0601 local-set decoder against bounded deterministic packet bytes
- **AND** the first MPEG-TS demux spike uses FFmpeg/ffprobe behind the SemStreams demux component boundary
- **AND** public media samples and sidecar parser choices remain follow-up smoke or production paths
- **AND** parser-core tests do not write graph mutations

#### Scenario: KLV demux worker remains flow based

- **WHEN** the first demux worker path is enabled
- **THEN** it consumes registered `semops.klv_media_ref.v1` BaseMessages from the declared media-ref input
- **AND** it uses explicit data-stream selection and bounded byte extraction for local file URI fixtures
- **AND** it publishes registered `semops.klv_packet.v1` BaseMessages to the declared packet output
- **AND** it does not publish graph mutation requests

#### Scenario: KLV decoder worker remains flow based

- **WHEN** the first decoder worker path is enabled
- **THEN** it consumes registered `semops.klv_packet.v1` BaseMessages from the declared packet input
- **AND** it publishes registered `semops.klv_misb0601_frame.v1` BaseMessages to the declared frame output
- **AND** it does not publish graph mutation requests

#### Scenario: Engineering support is distinct from official conformance

- **WHEN** SemOps uses public examples and deterministic fixtures for KLV/MISB acceptance
- **THEN** demo and documentation language may claim engineering support for the tested subset
- **AND** official STANAG 4609 conformance or certification remains blocked until a funded validator or lab effort
  with proper access exists

#### Scenario: KLV worker uses SemStreams component flow

- **WHEN** SemOps promotes KLV/MISB beyond planning
- **THEN** it is composed as media-reference input, demux processor, MISB decode processor, projector processor, and
  optional interop processors
- **AND** every stage uses declared ports, registered payloads, health, flow metrics, and config schema
- **AND** graph writes occur only in the projector through declared SemStreams graph request ports

#### Scenario: Streaming-binary claim is blocked by memory-bound evidence

- **WHEN** binary storage requires reading whole video files into memory
- **THEN** SemOps does not claim streaming-binary support until the implementation is replaced or bounded by tests
