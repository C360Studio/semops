# Feed Validation And Indexing Ladder

## Status

Draft baseline for the SemOps COP revival, created on 2026-06-17.

Inputs:

- Current SemOps MAVLink parser, generator, raw-lane, and command codec code.
- Current SemLink TAK bridge and demo seeding script.
- Current SemSource media design and video handler.
- Current SemStreams indexing-profile contract in ADR-054.
- Public protocol, simulator, and API documentation linked below.

## Direction

Add feeds one at a time. Each feed earns its way into the demo with a reproducible mock, simulator, replay,
conformance suite, or public sample source. A feed is not a product claim until it has a repeatable acceptance gate.

Use `docs/feed-product-roadmap.md` alongside this ladder. The ladder answers "what evidence lets the feed enter the
demo?" The roadmap answers "how does the narrow demo path avoid blocking the full product path?"

Recommended order:

1. MAVLink.
2. TAK/CoT.
3. CAP/EDXL.
4. CS API bidirectional interop.
5. ADS-B.
6. SAPIENT.
7. KLV/STANAG 4609.

MAVLink and TAK should be first because they create the live operator picture quickly. MAVLink stresses high-rate
current-state projection. TAK stresses operator markers, chat, and tactical XML. CAP is the first loose civilian
warning feed. KLV should stay a proof spike until we have hard evidence that SemOps plus SemStreams can handle
binary-derived video metadata honestly.

## Evidence Ladder

Each feed must progress through these gates:

1. **Native parser gate:** Decode real or spec-shaped native messages without writing graph state.
2. **Mock or simulator gate:** Produce deterministic local input without requiring an external mission system.
3. **Projection gate:** Write governed SemStreams graph mutations with ownership, provenance, confidence, and an
   explicit `indexing_profile`.
4. **Replay gate:** Re-run the same scenario from captured native data and get stable COP state.
5. **Breaking-tag graph gate:** Prove generated or replay feed input against the live SemStreams graph path when a
   framework-breaking contract change, such as ADR-055/056 must-exist, is pending.
6. **Compliance gate:** Run a public conformance suite, official schema validation, or documented interoperability
   test where one exists.
7. **Demo gate:** Show the feed in the COP with health, source, provenance, and stale-data behavior.

No feed should skip the parser, projection, and demo gates. Compliance may be unavailable for some feeds, but the
absence must be explicit.

Before a feed enters the structural stack, run an adversarial review against the evidence, indexing profile,
cardinality risk, stale-data behavior, and claim language.

The first scenario-runner core lives in `internal/scenario`. It replays generated MAVLink, deterministic TAK/CoT seed
events, and CAP lifecycle XML records through the real adapter/projector seams and exposes a pollable run status.
`cmd/semops-scenario-runner` hosts that core in the local Compose stack and the stack smoke polls
`/scenario/status`; it also asserts the Caddy-routed COP snapshot contains the scenario's MAVLink track, TAK/CoT task
and advisory, and CAP hazard. This is replay infrastructure evidence, not a full shared-airspace vignette or operator
scenario control surface. The runner can also opt into deterministic ADS-B fixture replay with
`SEMOPS_SCENARIO_ADSB_FIXTURE=true`; that path exercises the hosted ADS-B adapter and born-first owner token without
making live ADS-B part of the default MVP stack.

## Indexing Pressure

This demo will stress-test SemStreams indexing profiles as much as feed decoding.

Current SemStreams profiles are:

- `signal`: measured values that are useful in aggregate, not as prose.
- `control`: durable, low-cardinality state such as missions, commands, and lifecycle facts.
- `content`: domain text, advisories, descriptions, and extracted facts meant for meaning-based retrieval.
- `trace`: high-cardinality append-heavy execution or replay detail.

SemOps should treat profile assignment as a contract per projected entity type:

- Current platform and sensor state from MAVLink, TAK, ADS-B, and SAPIENT should usually be `signal`.
- Alerts, tasks, commands, feed health, and scenario state should usually be `control`.
- CAP advisory text, translated warnings, operator notes, and semantic explanations should usually be `content`.
- Native packet references, replay steps, keyframe extraction logs, and raw decode events should usually be `trace`.

The risky cases are mixed-shape entities:

- A TAK event may be both a position signal and an operator message.
- A CAP alert may be both a hazard polygon and a text advisory.
- A KLV frame may be video metadata, sensor footprint, evidence reference, and binary artifact pointer.
- A fused track may aggregate high-rate source state while also becoming an operator-relevant entity.

SemOps should not solve this by inventing more profiles immediately. The first pressure test is whether entity
boundaries are right: split high-rate state, durable control state, textual content, and replay trace where the
storage shape differs. Only file an upstream SemStreams ask if the same entity truly needs a reusable multi-profile
or field-level indexing policy.

## Feed Findings

### MAVLink

Status: first feed.

Compliance and simulator evidence:

- MAVLink has a public developer guide and common message vocabulary.
- PX4 SITL has a documented simulator path and MAVLink simulator API.
- ArduPilot SITL runs autopilot code without hardware across vehicle types.

Local assets:

- SemOps has an active MAVLink v1/v2 parser and MAVLink v2 generator in `pkg/adapters/mavlink`.
- The active tests prove heartbeat, global position, attitude, battery status, COMMAND_LONG, COMMAND_ACK, split
  buffers, noisy resync, checksum rejection, and deterministic scenario frames.
- The active raw lane stores copied MAVLink frames under record and byte caps and annotates decoded packets with a
  `cop.provenance.source_ref` for current-state projections.
- The active replay store persists raw-lane records as JSON Lines fixtures and loads them back into parseable frames.
- The adapter harness composes parser, raw lane, projector, graph plan writer, health counters, and an opt-in UDP
  datagram listener before the dedicated adapter service boundary exists.
- The old ignored parser/generator and SITL controller/scenario references were deleted after extraction or rejection.

Mock or harness:

- Use `go test ./pkg/adapters/mavlink` as the no-container parser/generator gate.
- Use active COMMAND_LONG/COMMAND_ACK tests as the command codec gate.
- Use generated or replay MAVLink frames against a live SemStreams graph stack to clear SemOps issue #1 before
  making PX4/SITL a blocking dependency. The generated-frame graph smoke passed on 2026-06-17.
- The clean-stack owner-registration smoke passed on 2026-06-17 with typed owner tokens minted by SemStreams
  registry/bind results.
- The clean-stack metrics smoke now asserts before/after deltas for unclassified indexing-profile, owner-token
  mismatch, and dropped-foreign-edge counters when a SemStreams metrics URL is provided.
- The one-command hosted graph smoke now carries the same metrics URL and delta assertions through
  `scripts/cop-stack-smoke.sh`.
- Add an ArduPilot SITL, PX4 SITL, or MAVSDK smoke harness before claiming live command/control.

Indexing profile pressure:

- Raw frames stay off the graph on bounded lanes or object references.
- Vehicle current state is `signal`.
- Commands, mission state, and low-battery alerts are `control`.
- Replay decode details are `trace`.

First acceptance gate:

- Given real MAVLink frames for heartbeat, global position, attitude, and battery, the adapter writes one current
  vehicle entity with source provenance and a raw source reference, and does not create one graph entity per packet.

Breaking-tag gate:

- Given generated or replay MAVLink heartbeat and position frames, the live SemStreams graph path creates the source
  asset, creates the track with the strict `cop.track.source` edge, and updates the known track without
  `entity_not_found` failures or dropped foreign-edge evidence.
- This gate precedes PX4/SITL because it isolates SemStreams graph compliance from simulator fidelity.

Current codec gate:

- `go test ./pkg/adapters/mavlink` proves real binary decode before graph projection.
- Battery status now guards canonical MAVLink wire order because the ignored reference layout was self-consistent but
  not sufficient interoperability evidence.
- `pkg/adapters/mavlink/raw_lane_test.go` proves the bounded in-memory raw lane separately from durable fixture
  storage.
- `pkg/adapters/mavlink/commands_test.go` proves command frame encoding and ACK parsing before any live SITL harness.
- `pkg/adapters/mavlink/replay_test.go` proves durable fixture append/load and parser replay.
- `go test ./internal/adapters/mavlink` proves parse, raw capture, projection, graph-plan write, and health ordering.

### TAK/CoT

Status: second feed.

Compliance and simulator evidence:

- No public TAK/CoT compliance suite was verified in this pass.
- CoT is still useful because the native XML shape is simple enough for deterministic fixtures and replay.

Local assets:

- SemOps now has `pkg/adapters/cot`, a dependency-light XML CoT codec tested against SemLink-style seed shapes for
  operator dots, markers, GeoChat, air-track marshal/unmarshal, and malformed input rejection.
- SemOps also has a CoT bounded raw lane, JSON Lines replay store, deterministic seed fixture pack, graph-free adapter
  harness, and UDP/TCP fixture replay tests.
- SemOps now has `internal/projectors/cot`, a pure projection planner that births TAK source assets before strict
  track source edges, maps markers to `control` tasks, maps GeoChat to `content` advisories, and carries raw source
  references without embedding native XML in graph entities.
- SemLink has a TAK bridge and `scripts/demo-up.sh` seeds UDP CoT events for operators, markers, and chat.

Mock or harness:

- Use SemOps-owned UDP and TCP seed events through `internal/adapters/cot`; SemLink remains prior art, not runtime
  dependency.
- Expand fixture replay next for hazard marker, stale event, duplicate UID, and malformed transport cases.

Indexing profile pressure:

- Position events project to `signal`.
- Markers and task-like map control state project to `control`.
- GeoChat text projects to separate `content` advisory entities.
- Native event references and replay steps are `trace`.

First acceptance gate:

- Given seeded ALPHA/BRAVO operator dots, a checkpoint marker, and a chat event, the COP shows positions, source,
  event freshness, and provenance without treating the native XML as embedded prose.

Current parser gate:

- `go test ./pkg/adapters/cot` passes for SemLink-style ALPHA/BRAVO seed shapes, North Gate marker, GeoChat remarks
  fallback, air-track marshal/unmarshal, classifier checks, and malformed input rejection.

Current transport/replay gate:

- `go test ./internal/adapters/cot` passes for direct ingest, malformed capture, replay append error handling, UDP
  fixture replay, TCP fixture replay, and listener/replay config guardrails.
- `go test ./pkg/adapters/cot` also proves raw CoT JSON Lines replay append/load and parse-after-load stability.

Current projection gate:

- `go test ./internal/projectors/cot` passes for source-asset-before-track birth ordering, TAK owner tokens, strict
  track source edges, marker-to-task `control` projection, GeoChat-to-advisory `content` projection, source refs,
  unsupported alert no-ops, and restart born-state seeding.

Current graph-wiring gate:

- `go test ./internal/projectors/cot ./internal/adapters/cot ./internal/stack ./internal/app` passes for CoT
  create/update graph writer behavior, restart birth reconciliation, NATS-backed stack composition, opt-in hosted
  CoT adapter composition, and UDP/TCP listener lifecycle wiring.
- TAK/CoT graph wiring remains a local unit/integration gate until a skipped live graph smoke verifies source asset,
  track, task, advisory, and source-ref readback against a running SemStreams graph.

### CAP/EDXL

Status: third feed; parser, deterministic raw XML lifecycle fixture replay, append-evidence projection, graph writer,
COP readback, first lifecycle-status readback, and skipped-by-default live graph smoke exist.

Compliance and sample evidence:

- CAP 1.2 is an OASIS standard with normative message, producer, and consumer conformance sections.
- The NWS API provides open alerts and can return `application/cap+xml`.

Local assets:

- `pkg/adapters/cap` parses the CAP 1.2 subset needed for deterministic civilian-warning fixtures.
- `pkg/adapters/cap` stores replayable raw XML CAP alert records and includes a HA/DR flood lifecycle fixture with
  alert, update, cancel, and expired-alert records.
- `internal/projectors/cap` births source-partitioned `hazard_area` entities and appends CAP evidence through the
  CAP evidence contract.
- `internal/api/cop` maps CAP hazard evidence JSON into the COP hazard view model for the map overlay and derives
  operator-facing status from CAP `msgType`, `status`, `expires`, and freshness without writing authoritative hazard
  status predicates.

Mock or harness:

- Use local CAP fixtures for the parser gate.
- Use the local HA/DR flood lifecycle fixture for deterministic replay without requiring live NWS calls.
- Use NWS alert samples for realistic civilian-warning input.
- Validate XML schema and CAP consumer rules before claiming CAP consumer conformance.

Indexing profile pressure:

- The current CAP slice uses `content` because it contributes append-only advisory and geometry evidence.
- Future authoritative alert lifecycle state is `control`.
- Advisory text, instructions, and multilingual descriptions remain `content`.
- Poll history and raw alert fetch traces are `trace`.

First acceptance gate:

- Given a CAP alert with polygon or circle area data, the parser preserves area evidence, the projector writes a
  born-first hazard area with provenance and confidence, and `GET /api/cop/snapshot` renders the hazard and derived
  lifecycle status without CAP claiming authoritative hazard geometry, severity, or status predicates.

Current commands:

```bash
go test ./pkg/adapters/cap ./internal/projectors/cap ./internal/api/cop ./internal/smoke/cap
go test ./internal/scenario
```

Live graph gate:

```bash
SEMOPS_CAP_LIVE_GRAPH_NATS_URL=<nats-url> go test ./internal/smoke/cap -run TestLiveGraphCAPBornFirstSmoke -v
```

This test skips unless `SEMOPS_CAP_LIVE_GRAPH_NATS_URL` points at a live SemStreams graph stack. It creates a CAP
hazard entity before appending update evidence, polls `graph.query.prefix`, and asserts CAP did not write
authoritative hazard geometry, severity, or status predicates.

Remaining gates:

- NWS samples captured as deterministic fixtures.
- XML schema and CAP consumer-rule validation.
- NWS-backed update/cancel/expire fixture replay and stale-data behavior beyond the local synthetic lifecycle fixture.
- Hosted poller or webhook service boundary.

### CS API Bidirectional Interop

Status: interop after structural graph is stable.

Compliance evidence:

- SemConnect has an OGC Connected Systems API conformance harness. Treat `./conformance/run.sh` as the truth source
  for current status, not a substitute `go test`.
- OGC positions Connected Systems API as the bridge between feature resources and dynamic data. Part 1 covers
  feature resources such as systems, procedures, deployments, sampling features, subsystems/components, and property
  definitions. Part 2 covers dynamic data such as datastreams, observations, control streams, commands, command
  status, system events, streaming, and snapshots.

Local assets:

- SemConnect provides the CS API gateway and ETS harness.
- SemLink already demonstrates CS API bridge patterns.

Mock or harness:

- Use SemOps graph state as input and SemConnect as a standards-facing projection for egress.
- Use a later SemOps CS API ingress adapter only for systems that already speak CS API.
- Run the SemConnect harness when SemOps exposes enough system, deployment, datastream, and observation state.

Indexing profile pressure:

- CS API projection targets should not drive indexing directly. The SemOps graph owner decides profile at entity
  birth; the bridge is an interface.

First acceptance gate:

- Egress: a SemOps asset/platform, hosted sensor, datastream, observation, deployment, and system event can be
  projected through SemConnect and checked by the conformance harness without weakening SemOps ownership rules.
- Ingress: a CS API source fixture can be mapped into SemOps canonical COP state without bypassing born-first writes,
  provenance, freshness, or command authority.
- Tasking: CS API command/control input routes through SemOps command authority and native safety controls rather
  than directly mutating feed-owned state.
- Replay: the same fixture can be replayed deterministically so bridge drift is visible before conformance claims.

### ADS-B

Status: later air-picture feed with OpenSky-shaped parser, deterministic replay, hosted-adapter seam, optional
structural scenario replay, current-state projection, owner registration, and graph readback evidence.

Compliance and sample evidence:

- OpenSky exposes REST state vectors, flights, and tracks. Public state-vector calls are rate limited.
- Current OpenSky docs use OAuth2 client-credentials authentication for authenticated calls; fixture replay remains
  the deterministic gate.
- ASTERIX is a later binary/radar-like target, not the first ADS-B slice.

Local assets:

- `pkg/adapters/adsb` parses OpenSky `/states/all` snapshot fixtures and preserves nullable state-vector fields,
  position-source quality, receiver IDs, and category values.
- `go test ./pkg/adapters/adsb` is the first executable ADS-B boundary.
- `pkg/adapters/adsb` also provides deterministic OpenSky snapshot fixture records plus JSONL replay load/store
  support.
- `internal/projectors/adsb` projects aircraft current state to source-partitioned ADS-B tracks with `signal`
  indexing, provenance, confidence, and source references.
- `internal/scenario` can replay ADS-B snapshots through parse, projection, graph-plan writing, and born-state
  marking through the hosted ADS-B adapter when a scenario opts into ADS-B.
- `internal/adapters/adsb` provides a hosted snapshot ingest seam with bounded raw capture, replay append,
  projection writes, SemStreams mutation writer integration, born-first reconciliation, and health counters.
- `cmd/semops-scenario-runner` adds ADS-B snapshots only when `SEMOPS_SCENARIO_ADSB_FIXTURE=true`; the Compose service
  passes that flag through but defaults it to false.
- The scenario runner appends `semops.feed.adsb` ownership only for that opt-in path so structural ADS-B graph writes
  use SemStreams minted owner tokens.
- The COP API discovers ADS-B aircraft tracks from `c360.<platform>.cop.adsb.track.*` prefixes and exposes
  `feed.adsb` health when fresh tracks exist.

Mock or harness:

- Start with recorded OpenSky JSON fixtures and deterministic replay. This now exists for snapshot fixtures and
  opt-in scenario-runner input.
- Add live OpenSky calls only as an optional demo mode with rate-limit handling.
- Consider readsb or dump1090-style local fixtures if we want raw ADS-B later.

Indexing profile pressure:

- Aircraft current state is `signal`.
- Track association decisions are separate fusion evidence, likely `control`.
- Raw receiver messages and replay rows are `trace`.

First acceptance gate:

- Given a bounded OpenSky state-vector fixture, SemOps decodes typed aircraft current-state evidence with nullable
  fields preserved and malformed rows rejected before any graph writes.
- Given deterministic OpenSky replay records, SemOps loads raw snapshot refs and replays them through the scenario
  runner without live network access.
- Given a hosted ADS-B snapshot ingest, SemOps captures raw JSON, appends replay records, projects current-state
  tracks, writes graph plans, reconciles already-born tracks, and reports health counters.
- Given `SEMOPS_SCENARIO_ADSB_FIXTURE=true`, the hosted scenario runner replays two ADS-B snapshots through the
  adapter seam and token-backed graph writes without live network access.
- Given a projected ADS-B aircraft state, SemOps writes current-state track evidence without source-asset or
  cross-source association edges and reads it back through prefix discovery.
- Next gate: decide whether live mode should start with optional OpenSky, local receiver/readsb/dump1090 files, or a
  dedicated adapter service, without making live network access the default path.

### SAPIENT

Status: official artifacts found; local parser and harness qualification still required.

Compliance and sample evidence:

- GOV.UK identifies SAPIENT as a Dstl/MOD-owned architecture, names BSI Flex 335 as the freely available ICD, and
  links official GitHub assets for protobuf files, a SAPIENT Test Harness, and SAPIENT Middleware.
- BSI describes `SAPIENT Network of Autonomous Sensors and Effectors - BSI Flex 335 V2:2024` as the current message
  structure, format, and content reference.
- Dstl publishes official protobuf definitions in `dstl/SAPIENT-Proto-Files`, including `bsi_flex_335_v2_0`.
- Dstl publishes a BSI Flex 335 v2 Test Harness with true/false JSON message corpora and validators. Its README says
  it is Windows-focused, .NET 6-based, and requires PostgreSQL 12.
- Dstl publishes Apex SAPIENT Middleware for routing, optional protobuf validation, archiving, replay, and REST API
  access.
- The Windows-only harness posture is itself product pressure: a future SemOps/ecosystem effort should create a
  portable Linux/CI-friendly SAPIENT preflight suite, while keeping official compliance claims tied to the Dstl
  harness or another accepted authority.

Local assets:

- None implemented in SemOps yet.
- No local SAPIENT harness run, generated Go bindings, parser package, or graph projector exists.

Mock or harness:

- Start with parser-only fixtures from official protobuf/sample-message evidence.
- Run or qualify the Dstl BSI Flex 335 v2 Test Harness before using compliance language.
- Track a portable preflight harness as developer evidence, not official compliance, until accepted externally.
- Treat Apex middleware as an interop and service-shape reference, not as a substitute for SemOps ownership,
  provenance, freshness, and command-authority contracts.

Indexing profile pressure:

- Detections and tracks are usually `signal`.
- Sensor tasking, collection plans, and alert state are `control`.
- Native decode traces are `trace`.

First acceptance gate:

- Given BSI Flex 335 v2-aligned fixtures, malformed messages are rejected before graph writes and valid detections
  become governed tracks or observations with clear source ownership.
- Given SemOps-generated SAPIENT messages, the Dstl v2 Test Harness result is recorded before any SAPIENT
  compliance claim appears in demo materials.

### KLV/STANAG 4609

Status: stretch proof spike.

Compliance and parser evidence:

- jMISB implements several MISB standards, including ST 0601 UAS Datalink, ST 0805 KLV-to-CoT conversion, and
  ST 1402 MPEG-2 transport stream support.
- `klvdata` is a Python library for parsing MISB ST 0601 KLV metadata from STANAG 4609-compliant MPEG-TS streams.

Local assets:

- SemSource has draft media support and a video handler that extracts metadata and keyframes with ffprobe/ffmpeg.
- SemSource can store metadata-only video entities, and can store binary files when a media store is configured.
- The current video handler streams hashing, but reads the full video into memory when binary storage is enabled.

Mock or harness:

- Treat SemSource as a candidate media sidecar, not a proven KLV solution.
- First prove video metadata and keyframe ingestion on a small fixture.
- Then prove KLV metadata extraction from a small ST 0601 sample stream using jMISB or `klvdata`.
- Only after that decide whether the production adapter is Go-native, Java sidecar, Python sidecar, or SemSource
  extension.

Indexing profile pressure:

- Sensor footprints and extracted platform/sensor coordinates are `signal`.
- Clip, asset, and evidence package lifecycle is `control`.
- Frame/keyframe descriptions, OCR, and operator annotations are `content` when present.
- Packet/frame decode events are `trace`.
- Binary payloads stay in object storage or external media storage, not graph triples.

First acceptance gate:

- Given a small video-plus-KLV fixture, the demo extracts a sensor footprint and platform/sensor position, stores
  binary by reference, and proves memory-bounded handling before any "streaming binary" product claim.

## Upstream SemStreams Questions

Do not file all of these immediately. Use SemOps evidence first.

- Are the four profiles enough if entity boundaries are chosen correctly?
- Does COP need per-substrate policy examples for high-rate geospatial feeds?
- Should SemStreams provide first-class profile/cardinality test helpers for adapters?
- Should raw-lane plus current-state projection guidance become a reusable framework recipe?
- Do provenance and confidence need standard predicates for source confidence versus fusion confidence?
- Should object-store references and media-derived graph evidence have a canonical vocabulary?
- Do spatial-temporal query helpers belong in SemStreams once MAVLink, TAK, CAP, and ADS-B all need them?

## Source Links

- PX4 simulation: <https://docs.px4.io/main/en/simulation/>
- ArduPilot SITL: <https://ardupilot.org/dev/docs/sitl-simulator-software-in-the-loop.html>
- MAVLink developer guide: <https://mavlink.io/en/>
- OASIS CAP 1.2: <https://docs.oasis-open.org/emergency/cap/v1.2/CAP-v1.2-os.pdf>
- NWS API: <https://www.weather.gov/documentation/services-web-api>
- OpenSky REST API: <https://openskynetwork.github.io/opensky-api/rest.html>
- OGC API - Connected Systems overview: <https://ogcapi.ogc.org/connectedsystems/>
- OGC Connected Systems SWG repository: <https://github.com/opengeospatial/ogcapi-connected-systems>
- jMISB: <https://github.com/WestRidgeSystems/jmisb>
- klvdata: <https://github.com/paretech/klvdata>
