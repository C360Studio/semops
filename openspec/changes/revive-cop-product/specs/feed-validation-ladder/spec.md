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

### Requirement: Feed evidence tiers cannot bypass product ingress

SemOps SHALL keep parser, projection-contract, component-flow, simulator, provider, standards, and product e2e
evidence separate for every feed.

#### Scenario: Product e2e enters through native or component ingress

- **WHEN** SemOps cites a feed as product-visible or end-to-end in the COP
- **THEN** input reaches the product through a supported native protocol boundary, fixture provider boundary, file
  boundary, HTTP polling boundary, or other declared SemStreams input component
- **AND** downstream parsing, decoding, projection, graph requests, component health, data flow, and Prometheus
  metrics are exercised when implemented for that feed
- **AND** a test helper MUST NOT inject graph mutations, decoded payloads, projected payloads, or raw NATS messages
  around the hosted component graph to satisfy the claim

#### Scenario: Direct graph smokes remain contract-only

- **WHEN** a direct live graph smoke writes feed projection plans through SemStreams
- **THEN** the evidence may clear born-first, ownership, indexing, restart-reconciliation, classified-error, and graph
  readback gates
- **AND** it MUST NOT clear product e2e, simulator-fidelity, live command-control, CS API interop, provider-service,
  or standards-conformance gates

#### Scenario: Duplicate owner incarnations fail product evidence

- **WHEN** a feed owner writes current COP state during product e2e
- **THEN** one live component or service incarnation owns that feed path
- **AND** owner-token mismatch warnings, stale leases, or observe-only owner mismatch deltas fail the product e2e
  claim even if graph readback eventually contains the expected entity

### Requirement: Feed fixtures declare promotion tier and provenance

SemOps SHALL track every feed fixture by promotion tier so live evidence, committed demo data, and synthetic story
data are not confused.

#### Scenario: Ignored live captures are evidence sources

- **WHEN** SemOps captures data from a live public, partner, simulator, or vendor feed
- **THEN** the raw capture first lands in an ignored local path
- **AND** SemOps records source URL or connection description, capture time, capture command, contact or credential
  posture, SHA-256, observed message shape, and claim scope
- **AND** ignored live captures may support evidence notes but do not travel with the demo until promoted

#### Scenario: Cleared committed fixtures travel with the demo

- **WHEN** a feed fixture is committed to the repo for portable demos or CI
- **THEN** a fixture manifest records source, provenance or license posture, capture or derivation time, SHA-256,
  size, retention decision, claim scope, and review decision
- **AND** committed fixtures are intentionally small, deterministic, and replayable without live network access
- **AND** committed fixtures do not imply product, standards, or provider conformance beyond their manifest scope

#### Scenario: Fixture manifest is executable evidence

- **WHEN** a fixture manifest entry is added or changed
- **THEN** an automated manifest test verifies the manifest version, unique fixture IDs, required provenance and claim
  scope fields, allowed promotion tier and commit status values, and readable review files
- **AND** committed fixture artifacts must match the manifest SHA-256 and byte size
- **AND** ignored live captures remain optional local files, but when present they must match the manifest SHA-256 and
  byte size
- **AND** portable files under `fixtures/` must not bypass the manifest unless they are explicitly ignored local
  captures, generated media, cache files, or non-fixture documentation

#### Scenario: Derived story fixtures are labeled synthetic

- **WHEN** SemOps derives a coherent demo lifecycle from live captures, public examples, or hand-authored truth data
- **THEN** the fixture is labeled as derived or synthetic story data
- **AND** the manifest identifies which fields came from observed evidence and which were edited, generated, or
  time-shifted for replay
- **AND** derived story fixtures may support demo narrative and regression tests but must not be cited as captured
  provider lifecycle, official compliance, or interoperability evidence

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
- **AND** the local fixture-provider CAP HTTP poller, decoder, and projector components can satisfy product e2e only
  for local fixture ingress through hosted SemStreams components
- **AND** the same evidence is not live NWS/IPAWS service evidence
- **AND** component health can report stale CAP polling when no fresh payload arrives within the configured
  `stale_after` threshold
- **AND** the opt-in runtime can capture provider-shaped CAP HTTP responses as replayable native CAP XML records
- **AND** CAP parser preflight rejects wrong or missing CAP 1.2 namespaces and invalid CAP 1.2 consumer-rule fields
  before graph projection
- **AND** the portable derived CAP lifecycle fixture is committed as replay JSONL, listed in the fixture manifest, and
  tested against the deterministic generator
- **AND** local CAP schema/sample smoke may validate developer-supplied CAP XML files or replay JSONL against a
  developer-supplied CAP 1.2 XSD with `xmllint`, then parse the same samples through SemOps, while skipping by default
  when no local schema is configured
- **AND** SemOps does not claim CAP webhook, NWS/IPAWS integration, or live alert feed service support until runtime
  wiring is backed by captured provider samples and alert lifecycle gates for that boundary

#### Scenario: Broader EDXL is not CAP evidence

- **WHEN** SemOps considers EDXL formats beyond CAP
- **THEN** each selected EDXL family requires a separate product need, legal fixture or captured provider sample,
  parser/schema evidence, component boundary, projection contract, indexing profile, replay behavior, and adversarial
  review
- **AND** CAP parser, CAP HTTP polling, and CAP hazard projection evidence must not be cited as broader EDXL support

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

#### Scenario: SAPIENT harness is qualified as out of MVP runtime scope

- **WHEN** the Dstl BSI Flex 335 v2 Test Harness is public but requires a Windows-focused runtime, .NET 6,
  PostgreSQL 12, manual configuration, and phase-specific test planning
- **THEN** SemOps MAY close the harness task only as a non-compliance MVP qualification
- **AND** SemOps SHALL record that the official harness was not run for the current phase
- **AND** SAPIENT compliance, product support, local harness success, and portable-suite authority language remain
  blocked until a scoped Dstl harness result or accepted authority result is recorded
- **AND** SemOps MAY continue JSON/protobuf parser preflight, raw replay, SemStreams component-flow, and opt-in
  absolute-location graph projection as developer evidence only

#### Scenario: MAVLink simulator telemetry is opt-in evidence

- **WHEN** SemOps adds PX4, MAVSDK, ArduPilot SITL, or hardware-adjacent MAVLink evidence
- **THEN** the simulator or hardware source must be explicit and must feed the hosted SemOps MAVLink component path
  rather than a test-only projector shortcut
- **AND** the smoke may observe simulator telemetry through the Caddy-routed COP snapshot without injecting generated
  MAVLink frames from the assertion process
- **AND** the smoke must assert live MAVLink feed health, source-owned provenance, non-empty source reference, position,
  velocity or equivalent state evidence, and repeated simulator updates before it is cited as simulator-fidelity
  evidence
- **AND** focused or stack evidence must record an explicit simulator family (`px4`, `ardupilot`, `mavsdk`,
  `hardware`, or `other`) before it is cited as parity evidence
- **AND** passing telemetry evidence for one simulator family SHALL NOT imply ArduPilot parity, MAVSDK/offboard parity,
  live command transmit, or broader command/control authority
- **AND** live command transmit, mission state, command ACK reconciliation, serial/TCP transports, signed links, and
  hardware behavior remain separate gates until reviewed

#### Scenario: MAVLink command ACK readback is not outbound authority

- **WHEN** SemOps projects MAVLink COMMAND_ACK packets
- **THEN** the ACK SHALL be written as `control`-profiled command readback task state
- **AND** the command task SHALL use a strict born-first `cop.task.target` edge to the MAVLink source asset
- **AND** known command-task updates SHALL NOT repeat the strict target edge after the task is born
- **AND** SemOps SHALL NOT treat ACK readback projection as evidence of live command transmission, mission execution,
  command priority arbitration, TTL-window policy, CS API tasking reconciliation, or native safety interlocks

#### Scenario: Command intent is governed before native execution

- **WHEN** SemOps accepts desired tasking state from a future local operator, CS API bridge, automation, or replay
- **THEN** the desired command SHALL be written through a product-owned `control`-profiled command-intent contract
- **AND** the command intent SHALL include authority, priority, expiry or TTL-derived deadline, correlation ID,
  idempotency key, requested-by, desired-state, status, provenance, and a strict born-first target asset edge
- **AND** the command-intent planner SHALL reject malformed or expired desired state before producing graph mutations
- **AND** command-intent status values SHALL be constrained to a documented lifecycle vocabulary with deterministic
  terminal-status and transition validation before handler or native-driver code can invent new states
- **AND** cancellation requests SHALL update the existing command intent to `cancel_requested` with a matching target,
  request authority, correlation, idempotency, provenance, and desired cancel state before any native cancellation is
  attempted
- **AND** the command-intent planner SHALL NOT birth target assets or transmit native feed commands
- **AND** admission SHALL reject unresolved target assets before producing graph mutations
- **AND** admission SHALL collapse duplicate idempotency keys before producing graph mutations
- **AND** command arbitration SHALL deterministically select at most one active command per target for native execution
  by local override, authority rank, priority, observation time, and native ID
- **AND** command arbitration SHALL surface superseded command intents as status decisions rather than handing losing
  intents to native feed drivers
- **AND** guarded batch projection SHALL admit commands before arbitration, project accepted and superseded status as
  command-intent graph state, and expose only accepted decisions as native execution candidates
- **AND** native status reconciliation SHALL map protocol readback evidence into constrained command-intent lifecycle
  status updates without rewriting desired-state, authority, priority, or target edges
- **AND** deadline reconciliation SHALL map unaccepted expired commands to `expired` and accepted, executing, or
  cancel-requested commands past deadline to `timeout` without rewriting desired-state, authority, priority, or target
  edges
- **AND** portable command lifecycle replay fixtures SHALL be synthetic, listed in the fixture manifest, tested against
  their deterministic generator, and capable of exercising requested, accepted, executing, cancel-requested,
  cancelled, timeout, and expired policy states without live native transmit
- **AND** native feed drivers SHALL publish ACK/status evidence separately rather than owning desired command intent
- **AND** live command transmission SHALL remain blocked until safety interlocks, local override, stale-command
  rejection, cancellation, supersession, and async status reconciliation are reviewed

#### Scenario: MAVLink command-control preflight is fail-closed before native transmit

- **WHEN** SemOps runs a MAVLink command-control preflight before a reviewed native transmitter exists
- **THEN** the helper SHALL require a named simulator source, simulator family, born-first command target, command
  action, and command safety profile
- **AND** the helper SHALL require explicit local-override, ACK-correlation, and post-command state-polling
  attestations before it writes command-control evidence
- **AND** the preflight helper SHALL reject any configured native transmitter or transmit-enabled flag because
  preflight is non-transmitting
- **AND** the helper SHALL write blocked evidence rather than passing the gate
- **AND** the blocked preflight SHALL NOT be cited as live command/control, mission execution, or native command
  authority evidence

#### Scenario: MAVLink live simulator command gate requires ACK and post-state evidence

- **WHEN** SemOps runs a MAVLink live simulator command-control gate
- **THEN** the helper SHALL require a named simulator source, simulator family, born-first command target, command
  action, simulator-scoped safety profile, local override posture, abort readiness, ACK requirement, and post-command
  state-polling requirement
- **AND** the helper SHALL reject hardware-family command evidence in this simulator gate
- **AND** the helper SHALL require an explicitly reviewed transmitter command and explicit transmit enablement before
  running any command
- **AND** the helper SHALL run the reviewed transmitter only after the COP stack and simulator are already live
- **AND** the helper SHALL poll the COP snapshot for graph-visible MAVLink `COMMAND_ACK` task evidence owned by the
  MAVLink feed
- **AND** the helper SHALL poll the COP snapshot for post-command MAVLink track refresh after the command start time
- **AND** the helper SHALL write blocked evidence rather than passing if the transmitter command, expected ACK task,
  or expected post-command track is missing
- **AND** a blocked simulator command gate SHALL NOT be cited as command authority, mission execution, or hardware
  safety evidence

#### Scenario: MAVLink MVP command scope stays read-side

- **WHEN** SemOps provides an MVP MAVLink simulator transmitter helper
- **THEN** the helper SHALL require simulator-only confirmation before sending any frame
- **AND** the helper SHALL allow only `MAV_CMD_REQUEST_MESSAGE` for `AUTOPILOT_VERSION` unless a later review expands
  the command scope
- **AND** the helper SHALL print command metadata and the expected ACK task suffix in dry-run mode
- **AND** the helper SHALL NOT include mission upload, mode change, arm/disarm, offboard control, or hardware command
  authority in MVP
- **AND** passing the helper dry-run SHALL NOT close live command/control until `command-live-sim` observes the ACK
  task and post-command track refresh through the COP snapshot

#### Scenario: MAVLink simulator readiness is not simulator evidence

- **WHEN** the external SITL smoke skips because no COP snapshot URL is configured or local PX4/MAVSDK/ArduPilot
  tooling is unavailable
- **THEN** SemOps records that as readiness-gap evidence only
- **AND** SemOps SHALL NOT close PX4/MAVSDK/SITL evidence gates or claim simulator fidelity until the observer-only
  smoke passes against an explicit simulator source feeding the hosted UDP component path
- **AND** the future pass records simulator name, version, launch command, system ID, UDP route, SemOps commit,
  pass/fail result, and any motion requirement

#### Scenario: MAVLink SITL helper requires simulator attestation

- **WHEN** SemOps provides a helper for the external SITL gate
- **THEN** preflight mode MAY run local readiness checks and the guarded skip test without simulator attestation
- **AND** focused or stack mode SHALL require a named simulator source before running the evidence gate
- **AND** focused or stack mode SHALL require a simulator family before running the evidence gate
- **AND** focused or stack mode SHALL require local simulator tooling for the declared simulator family or an explicit
  remote-source override
- **AND** any ArduPilot-specific helper mode SHALL stamp `simulator_family=ardupilot`, default to motion-required
  telemetry, and block unless `sim_vehicle.py`, an ArduPilot-family Docker image, or an explicit remote-source override
  is present
- **AND** any MAVSDK/offboard-specific helper mode SHALL stamp `simulator_family=mavsdk`, default to motion-required
  telemetry, and block unless `mavsdk_server`, a MAVSDK-family Docker image, or an explicit remote-source override is
  present
- **AND** helper evidence SHALL be written to ignored local paths rather than committed as portable demo evidence

#### Scenario: SAPIENT parser preflight is not conformance

- **WHEN** SemOps parses BSI Flex 335 v2 JSON fixtures derived from official SAPIENT sample shapes
- **THEN** the parser may be used as developer preflight evidence before graph projection
- **AND** SemOps may add descriptor-based binary protobuf preflight from official proto source without claiming
  full-message coverage, portable-suite authority, product support, or SAPIENT compliance
- **AND** SemOps may run SAPIENT raw ingress and decoder preflight as SemStreams input and processor components only
  when they publish raw/decoded streams and avoid graph mutation ports, owner claims, and product support wording
- **AND** the hosted app may compose that HTTP input -> decoder chain behind `SEMOPS_SAPIENT_ENABLED=true` only when
  URL, encoding, replay, and stale-source settings remain explicit preflight configuration
- **AND** SemOps may add `OwnerSAPIENT` and a source-partitioned `signal` track projection contract only for reviewed
  absolute-location detection reports
- **AND** that first projection rejects range/bearing, UTM, unsupported datum, tasking, association, and lifecycle
  semantics until those policies are reviewed
- **AND** SemOps may add a SAPIENT graph-producing processor component and graph writer only behind
  `SEMOPS_SAPIENT_GRAPH_ENABLED=true`, with `OwnerSAPIENT` registered only for that graph-enabled runtime path
- **AND** stack smoke coverage for SAPIENT graph projection remains opt-in, fixture-backed, and detection-source
  explicit so the default task-ack preflight smoke is not mistaken for graph or tasking support
- **AND** committed SAPIENT task-ack and absolute-location detection fixtures are listed in the fixture manifest and
  tested against the runtime fixture-service payloads
- **AND** SemOps does not claim compliance until scoped Dstl harness evidence exists
- **AND** SemOps does not claim product service support, tasking, association, UTM conversion, range/bearing
  conversion, or Apex middleware behavior until service mode, backpressure, command authority, and harness scope have
  been reviewed

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
- **AND** hosted SAPIENT graph production is opt-in and separate from preflight decode, product-service hosting,
  compliance, tasking, range/bearing, UTM, and association claims

#### Scenario: ADS-B starts with OpenSky fixture parsing and source-partitioned projection

- **WHEN** SemOps admits ADS-B into the feed ladder
- **THEN** the first executable gate parses bounded OpenSky-shaped state-vector fixtures before projection
- **AND** nullable callsign, position timestamp, longitude, latitude, altitude, velocity, track, vertical rate,
  receiver IDs, squawk, position source, and category fields remain explicit
- **AND** aircraft current state projects to source-partitioned ADS-B `track` entities with `signal` indexing,
  provenance, confidence, and source references
- **AND** deterministic OpenSky snapshot records can replay through the scenario runner without live network access
- **AND** committed OpenSky-shaped replay JSONL is listed in the fixture manifest and tested against the deterministic
  fixture generator
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

#### Scenario: SemSource proof closes only storage/governance evidence

- **WHEN** SemSource provides an opaque synthetic binary proof with by-reference storage, governed metadata,
  indexing-profile evidence, and no raw-binary graph triples
- **THEN** SemOps may count that as the KLV/SemSource storage/governance proof spike
- **AND** SemOps still SHALL NOT claim KLV parser support, live media ingress, streaming-binary product support,
  video-service support, or STANAG 4609 conformance from that proof

#### Scenario: KLV demux remains a SemOps product boundary for MVP

- **WHEN** SemOps consumes SemSource media references or native media ingress
- **THEN** the MVP demuxes KLV in a SemOps-owned component or sidecar
- **AND** future SemSource media-track extraction is treated as generic substrate support, not SemOps KLV/STANAG
  product support

### Requirement: DJI and weather remain first-class feed gates

SemOps SHALL track DJI and weather as explicit feed or layer gates rather than hiding them under MAVLink, CAP, or
KLV evidence.

#### Scenario: DJI media reinforces generic media references

- **WHEN** DJI video or recorded media enters SemOps
- **THEN** the media path uses generic media references and bounded metadata extraction where possible
- **AND** DJI telemetry, subtitles, or vendor metadata are not forced through the KLV/MISB decoder unless the source
  actually emits KLV
- **AND** deterministic synthetic DJI-shaped fixtures prove telemetry, media-reference, and command-authority posture
  shape before live DJI product support claims
- **AND** DJI fixture promotion uses SemStreams input and processor components with registered payloads, declared
  ports, config schema, health, and flow metrics before any graph projector or live bridge is added
- **AND** the existence of DJI video does not move KLV/STANAG product claims into SemSource

#### Scenario: Weather layers are tracked separately

- **WHEN** SemOps adds weather to the COP
- **THEN** visual weather tiles, CAP/public alerts, and tactical weather telemetry have separate evidence gates
- **AND** deterministic provider-shaped parser fixtures precede live weather-provider claims
- **AND** provider-shaped weather fixture promotion uses SemStreams input and processor components with registered
  payloads, declared ports, config schema, health, and flow metrics before any graph projector, HTTP poller, cache
  policy, or route-safety rule is added
- **AND** OGC EDR-shaped fixtures prove point first, then area, trajectory, corridor, and selected broader query-shape
  parsing before standards-facing tactical weather interop claims exceed point retrieval
- **AND** OGC EDR area, trajectory, and corridor parser fixtures remain parser evidence until spatial runtime
  payloads, graph projection, freshness/confidence policy, and route-weather UI semantics are accepted
- **AND** tactical weather graph projection uses source-partitioned `weather_observation` signal entities for bounded
  variable/time samples without claiming hazard, task, alert, route-decision, or advisory authority
- **AND** weather graph-projector promotion uses decoded forecast stream ports, declared SemStreams graph request
  ports, bounded observation caps, freshness configuration, and born-first reconciliation before hosted runtime
  enablement or live-provider claims
- **AND** hosted weather runtime enablement starts as explicit, default-off, fixture-backed point-forecast composition
  before live HTTP polling, cache/stale policy, spatial runtime payloads, tactical UI semantics, or route-safety rules
  are accepted
- **AND** tactical weather supports the relevant point, area, and route/trajectory query shapes before it influences
  routing or safety logic

### Requirement: KLV fixture and runtime evidence remains bounded

SemOps SHALL keep KLV public-sample, deterministic fixture, runtime, and UI evidence bounded to the proven MISB ST
0601 subset until stronger fixture, live media, and conformance gates are accepted.

#### Scenario: Public samples are smoke evidence only

- **WHEN** SemOps uses a public video-plus-KLV sample such as a widely circulated MPEG-TS KLV file
- **THEN** the sample has documented license and provenance before use
- **AND** the result is labeled as demux/parser smoke evidence, not deterministic correctness or conformance
- **AND** automated public-sample smoke is opt-in, local-path based, and does not download or vendor public media

#### Scenario: Deterministic KLV fixture proves engineering support

- **WHEN** SemOps claims MISB ST 0601 engineering support
- **THEN** a deterministic fixture traces truth JSON through encoded KLV and optional MPEG-TS wrapping to parsed output
- **AND** acceptance asserts the parsed field set and parsed numeric values against the original truth data within MISB
  integer quantization tolerances for the supported field subset
- **AND** footprint-polygon acceptance MAY use MISB ST 0601 offset-corner fields only when all four corner pairs and a
  frame center are present
- **AND** partial or unusable corner evidence SHALL produce warning evidence rather than a synthetic polygon
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
- **AND** storage-reference-only media refs are accepted only through an explicit bounded materializer
- **AND** it splits concatenated MISB ST 0601 local sets into distinct bounded packet payloads with packet refs and
  byte offsets
- **AND** it publishes registered `semops.klv_packet.v1` BaseMessages to the declared packet output
- **AND** it does not publish graph mutation requests

#### Scenario: KLV decoder worker remains flow based

- **WHEN** the first decoder worker path is enabled
- **THEN** it consumes registered `semops.klv_packet.v1` BaseMessages from the declared packet input
- **AND** storage-reference-only packet payloads are accepted only through an explicit bounded packet materializer
- **AND** it publishes registered `semops.klv_misb0601_frame.v1` BaseMessages to the declared frame output
- **AND** it does not publish graph mutation requests

#### Scenario: Engineering support is distinct from official conformance

- **WHEN** SemOps uses public examples and deterministic fixtures for KLV/MISB acceptance
- **THEN** demo and documentation language may claim engineering support for the tested subset
- **AND** engineering-support wording for KLV, MISB ST 0601, STANAG 4609, and streaming-binary claims must pass a
  dedicated adversarial claim-language review before it appears in demo copy
- **AND** official STANAG 4609 conformance or certification remains blocked until a funded validator or lab effort
  with proper access exists

#### Scenario: KLV worker uses SemStreams component flow

- **WHEN** SemOps promotes KLV/MISB beyond planning
- **THEN** it is composed as media-reference input, demux processor, MISB decode processor, projector processor, and
  optional interop processors
- **AND** every stage uses declared ports, registered payloads, health, flow metrics, and config schema
- **AND** graph writes occur only in the projector through declared SemStreams graph request ports
- **AND** the first projector contract writes only source-partitioned KLV `sensor_footprint` sensor/frame-center and
  decoded offset-corner footprint state with `indexing_profile=signal`, leaving broad footprint policy and media
  service behavior as later gates

#### Scenario: Hosted KLV runtime remains opt-in and local-media bounded

- **WHEN** the hosted SemOps app enables KLV/MISB through runtime config
- **THEN** the app registers the KLV `sensor_footprint` ownership contract before composing graph-writing components
- **AND** it wires media-ref input -> demux -> MISB decode -> projector through SemStreams subjects and component
  lifecycle hooks
- **AND** the media-ref input publishes only local file references discovered from the configured path and glob
- **AND** KLV stays disabled in the default stack until live media ingress, storage policy, and operator-facing claims
  receive separate adversarial review

#### Scenario: KLV UI proof uses graph readback

- **WHEN** SemOps makes KLV visible in the operator COP
- **THEN** the visible state comes from governed `sensor_footprint` graph readback through the SemOps COP API
- **AND** the first UI layer renders sensor position, frame center, and the sensor-to-frame-center ray only after the
  API exposes media reference, packet reference, decoded field inventory, warning evidence, source provenance, and
  claim posture
- **AND** the UI does not treat public-sample smoke, local media availability, or decoded packet bytes as footprint
  polygon, streaming-binary, video-service, or STANAG 4609 conformance evidence

#### Scenario: Streaming-binary claim is blocked by memory-bound evidence

- **WHEN** binary storage requires reading whole video files into memory
- **THEN** SemOps does not claim streaming-binary support until the implementation is replaced or bounded by tests
