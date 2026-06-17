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

Recommended order:

1. MAVLink.
2. TAK/CoT.
3. CAP/EDXL.
4. SemConnect CS API egress.
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
- The in-process adapter harness composes parser, raw lane, projector, graph plan writer, and health counters before
  the container service boundary exists.
- The old ignored parser/generator and SITL controller/scenario references were deleted after extraction or rejection.

Mock or harness:

- Use `go test ./pkg/adapters/mavlink` as the no-container parser/generator gate.
- Use active COMMAND_LONG/COMMAND_ACK tests as the command codec gate.
- Use generated or replay MAVLink frames against a live SemStreams graph stack to clear SemOps issue #1 before
  making PX4/SITL a blocking dependency. The generated-frame graph smoke passed on 2026-06-17.
- The clean-stack owner-registration smoke passed on 2026-06-17 with registry-derived `<owner>#<incarnation>` tokens.
- Before broad expansion, add before/after assertions for unclassified indexing-profile, owner-token mismatch, and
  dropped-foreign-edge counters.
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

- SemLink has a TAK bridge and `scripts/demo-up.sh` seeds UDP CoT events for operators, markers, and chat.

Mock or harness:

- Start with SemLink-style UDP and TCP seed events.
- Add fixture replay for common event kinds: operator location, marker, GeoChat, hazard marker, and stale event.

Indexing profile pressure:

- Position events project to `signal`.
- Markers, tasks, and chat-visible operator state project to `control` unless the payload is primarily text.
- Remarks and chat text may need separate `content` entities or evidence fields.
- Native event references and replay steps are `trace`.

First acceptance gate:

- Given seeded ALPHA/BRAVO operator dots, a checkpoint marker, and a chat event, the COP shows positions, source,
  event freshness, and provenance without treating the native XML as embedded prose.

### CAP/EDXL

Status: third feed.

Compliance and sample evidence:

- CAP 1.2 is an OASIS standard with normative message, producer, and consumer conformance sections.
- The NWS API provides open alerts and can return `application/cap+xml`.

Local assets:

- None identified in SemOps yet.

Mock or harness:

- Use OASIS CAP examples and local fixtures for the parser gate.
- Use NWS alert samples for realistic civilian-warning input.
- Validate XML schema and CAP consumer rules before graph writes.

Indexing profile pressure:

- Hazard areas and alert lifecycle state are `control`.
- Advisory text, instructions, and multilingual descriptions are `content`.
- Poll history and raw alert fetch traces are `trace`.

First acceptance gate:

- Given a CAP alert with polygon or circle area data, the adapter writes a hazard area and advisory with provenance,
  confidence, expiry/staleness behavior, and no overwrite of stricter source facts.

### SemConnect CS API Egress

Status: egress after structural graph is stable.

Compliance evidence:

- SemConnect has an OGC Connected Systems API conformance harness. Treat `./conformance/run.sh` as the truth source
  for current status, not a substitute `go test`.

Local assets:

- SemConnect provides the CS API gateway and ETS harness.
- SemLink already demonstrates CS API bridge patterns.

Mock or harness:

- Use SemOps graph state as input and SemConnect as a standards-facing projection.
- Run the SemConnect harness when SemOps exposes enough system, deployment, datastream, and observation state.

Indexing profile pressure:

- CS API projection targets should not drive indexing directly. The SemOps graph owner decides profile at entity
  birth; egress is a view.

First acceptance gate:

- A SemOps asset, sensor, datastream, and observation can be projected through SemConnect and checked by the
  conformance harness without weakening SemOps ownership rules.

### ADS-B

Status: later air-picture feed.

Compliance and sample evidence:

- OpenSky exposes REST state vectors, flights, and tracks. Public state-vector calls are rate limited.
- ASTERIX is a later binary/radar-like target, not the first ADS-B slice.

Local assets:

- None identified in SemOps yet.

Mock or harness:

- Start with recorded OpenSky JSON fixtures and deterministic replay.
- Add live OpenSky calls only as an optional demo mode with rate-limit handling.
- Consider readsb or dump1090-style local fixtures if we want raw ADS-B later.

Indexing profile pressure:

- Aircraft current state is `signal`.
- Track association decisions are separate fusion evidence, likely `control`.
- Raw receiver messages and replay rows are `trace`.

First acceptance gate:

- Given a bounded OpenSky state-vector fixture, the adapter writes aircraft current state, freshness, source, and
  position-source evidence without creating high-cardinality graph noise.

### SAPIENT

Status: evidence gap before phase commitment.

Compliance and sample evidence:

- A public SAPIENT compliance suite was not verified in this pass.
- The first task is to find the authoritative SAPIENT ICD, protobuf definitions, sample messages, validator, or
  conformance tooling.

Local assets:

- None identified in SemOps yet.

Mock or harness:

- Do not build around guessed schema.
- Once authoritative artifacts are found, create a parser-only gate and a strict fixture suite before graph writes.

Indexing profile pressure:

- Detections and tracks are usually `signal`.
- Sensor tasking, collection plans, and alert state are `control`.
- Native decode traces are `trace`.

First acceptance gate:

- Given authoritative SAPIENT fixtures, malformed messages are rejected before graph writes and valid detections
  become governed tracks or observations with clear source ownership.

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
- jMISB: <https://github.com/WestRidgeSystems/jmisb>
- klvdata: <https://github.com/paretech/klvdata>
