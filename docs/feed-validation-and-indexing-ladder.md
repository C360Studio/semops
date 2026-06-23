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
3. CAP, with broader EDXL held for later feed-validation gates.
4. Weather as a layered feed: visual map context, CAP/public alerts, and tactical point/area/route weather telemetry.
5. DJI sensor/telemetry and media references.
6. CS API bidirectional interop.
7. ADS-B.
8. SAPIENT.
9. KLV/STANAG 4609.

MAVLink and TAK should be first because they create the live operator picture quickly. MAVLink stresses high-rate
current-state projection. TAK stresses operator markers, chat, and tactical XML. CAP is the first loose civilian
warning feed. Weather and DJI are critical COP layers, but they should enter through explicit layer/feed gates rather
than destabilizing the current Phase 1 spine. KLV should stay a proof spike until we have hard evidence that SemOps
plus SemStreams can handle binary-derived video metadata honestly.

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
making live ADS-B part of the default MVP stack. The Compose stack also includes `cmd/semops-feed-fixtures`, a local
HTTP provider simulator for ADS-B and SAPIENT smoke tests. That service is mock infrastructure only; it is not a
SemOps-owned TAK, SAPIENT, OpenSky, or CS API product service.

## Indexing Pressure

This demo will stress-test SemStreams indexing profiles as much as feed decoding.

Current SemStreams profiles are:

- `signal`: measured values that are useful in aggregate, not as prose.
- `control`: durable, low-cardinality state such as missions, commands, and lifecycle facts.
- `content`: domain text, advisories, descriptions, and extracted facts meant for meaning-based retrieval.
- `trace`: high-cardinality append-heavy execution or replay detail.

SemOps should treat profile assignment as a contract per projected entity type:

- Current platform and sensor state from MAVLink, TAK, DJI, ADS-B, and SAPIENT should usually be `signal`.
- Tactical weather observations near assets or routes should usually be `signal`.
- Alerts, tasks, commands, feed health, and scenario state should usually be `control`.
- CAP/weather advisory text, translated warnings, operator notes, and semantic explanations should usually be
  `content`.
- Native packet references, replay steps, keyframe extraction logs, and raw decode events should usually be `trace`.

The risky cases are mixed-shape entities:

- A TAK event may be both a position signal and an operator message.
- A CAP alert may be both a hazard polygon and a text advisory.
- A weather layer may be browser-only raster context, a tactical point/route observation, or an alert/advisory.
- DJI may be a vehicle track, gimbal/sensor state, command authority, media reference, and vendor-specific evidence.
- A KLV frame may be video metadata, sensor/frame-center evidence, sensor footprint, evidence reference, and binary
  artifact pointer.
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
- The one-command hosted stack smoke also scrapes the SemOps `/metrics` endpoint through Caddy and asserts
  `semops_component_health_status`, `semops_component_flow_messages_per_second`, and
  `semops_component_flow_last_activity_timestamp_seconds` for the hosted MAVLink input/decoder/projector chain.
- SemOps now has a skipped-by-default external PX4/MAVSDK/SITL telemetry smoke:
  `SEMOPS_MAVLINK_SITL_SMOKE_SNAPSHOT_URL=<cop-snapshot-url> go test ./internal/smoke/mavlink -run
  TestExternalSITLTelemetryCOPSnapshot -count=1 -v`. The smoke observes a real simulator track through
  `GET /api/cop/snapshot` without injecting generated frames, requires live MAVLink feed health, provenance, source
  refs, velocity evidence, and repeated simulator updates, and can require motion with
  `SEMOPS_MAVLINK_SITL_SMOKE_REQUIRE_MOTION=true`.
- The one-command stack smoke can include that external telemetry gate with
  `SEMOPS_COP_SMOKE_MAVLINK_SITL_ENABLED=true`; in that mode `compose.cop.yml` honors
  `SEMOPS_COP_MAVLINK_SYSTEM_IDS`, defaulting to systems `1,42` for PX4-style system ID 1 plus the deterministic
  generated-frame system 42.
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
- `go test ./internal/smoke/mavlink` proves the external SITL smoke skips unless an explicit COP snapshot URL is set.
  A real PX4/MAVSDK run is still required before MAVLink simulator-fidelity claims.

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
- SemOps now has `internal/components/cot`, a SemStreams flow component package with UDP/TCP input components,
  raw-event decoder processor, graph projector processor, registered raw/decoded `message.BaseMessage` payloads,
  config schemas, health, and flow metrics.
- The one-command hosted stack smoke now asserts Prometheus component health and flow samples for the hosted CoT UDP
  input, decoder, and projector chain through SemOps `/metrics`.
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

Current component and graph-wiring gate:

- `go test ./internal/components/cot ./internal/projectors/cot ./internal/adapters/cot ./internal/stack ./internal/app`
  passes for CoT component payload/flowgraph contracts, create/update graph writer behavior, restart birth
  reconciliation, NATS-backed stack composition, opt-in hosted component-flow composition, and UDP/TCP input lifecycle
  wiring.
- The Docker stack smoke and direct live graph smoke verify source asset, track, task, advisory, and source-ref
  readback against a running SemStreams graph.

### CAP/EDXL

Status: third feed. CAP has parser, deterministic raw XML lifecycle fixture replay, append-evidence projection, graph
writer, COP readback, first lifecycle-status readback, skipped-by-default live graph smoke, and initial HTTP poller,
decoder, and graph-projector component package. Broader EDXL is split into
`openspec/changes/revive-cop-product/feed-evidence/edxl-beyond-cap.md` as a later feed-validation gate and is not
Phase 1 scope.

Compliance and sample evidence:

- CAP 1.2 is an OASIS standard with normative message, producer, and consumer conformance sections.
- The NWS API provides open alerts and can return `application/cap+xml`.

Local assets:

- `pkg/adapters/cap` parses the CAP 1.2 subset needed for deterministic civilian-warning fixtures.
- `pkg/adapters/cap` rejects wrong or missing CAP 1.2 namespaces and validates enum-shaped consumer fields before
  graph writes. This is namespace/consumer-rule preflight evidence, not formal XSD conformance.
- `pkg/adapters/cap` has an opt-in schema/sample smoke that uses `SEMOPS_CAP_XSD_PATH`,
  `SEMOPS_CAP_SCHEMA_SAMPLE_PATHS`, `SEMOPS_CAP_SCHEMA_REPLAY_PATH`, and local `xmllint` to validate supplied CAP XML
  files or replay records before parsing them through SemOps. This is not default CI and not conformance.
- `pkg/adapters/cap` stores replayable raw XML CAP alert records and includes a HA/DR flood lifecycle fixture with
  alert, update, cancel, and expired-alert records.
- `internal/projectors/cap` births source-partitioned `hazard_area` entities and appends CAP evidence through the
  CAP evidence contract.
- `internal/api/cop` maps CAP hazard evidence JSON into the COP hazard view model for the map overlay and derives
  operator-facing status from CAP `msgType`, `status`, `expires`, and freshness without writing authoritative hazard
  status predicates.
- `internal/components/cap` provides a SemStreams lifecycle HTTP poller input component, a raw-alert decoder
  processor, a born-first graph-projector processor, registered raw/decoded `message.BaseMessage` payloads,
  `HTTPClientPort`, `TimerPort`, stream ports, graph request ports, config schema, health, and flow metrics. It is
  deterministic local component evidence, not default live NWS service proof.

Mock or harness:

- Use local CAP fixtures for the parser gate.
- Use the local HA/DR flood lifecycle fixture for deterministic replay without requiring live NWS calls.
- Use NWS alert samples for realistic civilian-warning input.
- Validate formal XML schema and CAP consumer profile rules before claiming CAP consumer conformance.

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
go test ./pkg/adapters/cap ./internal/projectors/cap ./internal/api/cop ./internal/smoke/cap ./internal/components/cap
go test ./internal/scenario
```

The CAP adapter command now proves wrong/missing CAP 1.2 namespaces, invalid `status`/`msgType`/`scope` values,
invalid `info` category/severity-style values, invalid `expires` ordering, and missing `areaDesc` values fail before
graph writes.

Optional CAP schema/sample smoke:

```bash
SEMOPS_CAP_XSD_PATH="fixtures/cap/schema/CAP-v1.2.xsd" \
SEMOPS_CAP_SCHEMA_SAMPLE_PATHS="fixtures/cap/nws-samples" \
go test ./pkg/adapters/cap -run TestCAPSchemaSmokeWithLocalSamples -count=1 -v
```

`docs/cap-schema-smoke.md` records the local XSD/sample and replay JSONL workflow. Local schema copies and captured
NWS/IPAWS/vendor samples remain ignored until a fixture review clears what may be committed.

Current component gate:

```bash
go test ./internal/components/cap ./internal/contracts
```

This gate proves the CAP HTTP poller, decoder, and projector are real SemStreams components with registered payloads,
declared ports, config schema, health, and flow metrics. The poller uses `HTTPClientPort` for the external feed
dependency and a sibling `TimerPort` referenced by `TriggerPort` for cadence. The projector declares graph mutation
request ports and reconciles restart-time create conflicts. The poller now exposes `stale_after`, treats
`304 Not Modified` as provider contact without duplicate publish, and reports stale health when fresh provider payloads
age out. The decoder can append provider-shaped raw CAP XML to replay JSONL through `SEMOPS_CAP_REPLAY_PATH`. Tests
use a local HTTP server; live NWS remains a separate optional gate. SemStreams issue #312 tracks first-class flowgraph
cadence semantics for `TimerPort`.

Live graph gate:

```bash
SEMOPS_CAP_LIVE_GRAPH_NATS_URL=<nats-url> go test ./internal/smoke/cap -run TestLiveGraphCAPBornFirstSmoke -v
```

This test skips unless `SEMOPS_CAP_LIVE_GRAPH_NATS_URL` points at a live SemStreams graph stack. It creates a CAP
hazard entity before appending update evidence, polls `graph.query.prefix`, and asserts CAP did not write
authoritative hazard geometry, severity, or status predicates.

Remaining gates:

- NWS samples captured as deterministic fixtures.
- Recorded formal XML schema validation and captured NWS sample replay.
- NWS-backed update/cancel/expire fixture replay and stale-data behavior beyond the local synthetic lifecycle fixture.
- Real NWS/IPAWS/vendor sample capture for the opt-in `SEMOPS_CAP_ENABLED=true` runtime chain.
- Default live-provider enablement; Compose exposes CAP knobs but keeps hosted public-alert polling disabled by default.
- Webhook, watched-file, or vendor feed service boundaries.
- Telemetry-driven backpressure decisions once hosted CAP polling has real provider cadence, retry, retention, and
  audit pressure.

### Weather

Status: critical COP layer with first Open-Meteo-shaped point, OGC EDR-shaped point, OGC EDR-shaped spatial parser
fixtures, SemStreams component-flow evidence for point payloads, a governed tactical-weather graph writer and
projector component gate, and opt-in hosted runtime composition for the local point-forecast fixture, split into
visual context, alert evidence, and tactical telemetry.

Compliance and source evidence:

- OGC API - Environmental Data Retrieval is the standards-facing target for tactical weather because it defines
  discovery plus query operations and supports position, area, trajectory, and corridor query shapes.
- The OGC API EDR 1.1 standard also defines radius, cube, items, locations, and instances query resources. The current
  SemOps gate covers point, area, trajectory, and corridor-shaped synthetic CoverageJSON only.
- Environment and Climate Change Canada's MSC GeoMet is a practical public OGC API/WMS/WCS source for testing open
  weather interoperability.
- Open-Meteo is a useful developer-friendly JSON source for deterministic provider-shaped fixtures and early
  tactical variables such as wind, precipitation, visibility, pressure, and temperature.
- SemOps now carries `fixtures/weather/open-meteo-point.json` as the first deterministic provider-shaped tactical
  weather fixture.
- SemOps also carries `fixtures/weather/ogc-edr-position.json` as a synthetic OGC EDR-shaped CoverageJSON point
  fixture. It is parser/storage/governance evidence only, not an official OGC ETS run, live EDR server capture, or
  conformance sample.
- SemOps also carries synthetic OGC EDR-shaped area, trajectory, and corridor fixtures. They prove simple WKT
  `POLYGON`, `LINESTRING`, corridor width/height, unit, and time-series variable parsing before route-weather
  projection; they are not route-safety, provider-dimensionality, Z/M coordinate-time, or conformance evidence.
- `pkg/adapters/weather` parses Open-Meteo-shaped and OGC EDR-shaped point forecasts and preserves provider, query
  shape, position, elevation, units, sample time, temperature, precipitation, visibility, surface pressure, wind speed,
  gusts, wind direction, and weather code without graph writes.
- `pkg/adapters/weather` also parses OGC EDR-shaped area, trajectory, and corridor spatial forecasts without graph
  writes or point-payload promotion.
- `pkg/cop` defines `weather_observation` as source-partitioned `signal` evidence for localized weather variable
  samples under `semops.feed.weather`.
- `internal/projectors/weather` plans and writes weather observation graph mutations for point and spatial forecasts
  without UI semantics or route-safety decisions.
- `internal/components/weather` wraps provider-shaped weather fixtures as SemStreams file input components, decoder
  processors, and point-forecast graph projector processors with registered payloads, file/NATS/NATS-request ports,
  config schema, health, flow metrics, per-payload observation caps, freshness windows, and born-first reconciliation.
- The hosted app can opt into the local point-forecast fixture flow with `SEMOPS_WEATHER_ENABLED=true`; it registers
  `semops.feed.weather` before composing the SemStreams file input -> decoder -> graph-projector chain.
- `GET /api/cop/snapshot` can read graph-backed `weather_observation` entities by prefix discovery and expose provider,
  variable/value/unit, query shape, query geometry, valid/model/freshness time, provenance, and claim posture without
  adding route-safety or visual weather-tile semantics.
- The COP UI can render a selectable point-observation evidence marker and inspector details for graph-backed weather
  observations. This proves browser readback of localized weather evidence only; it does not prove a tactical weather
  product, incident-area layer, route-weather model, or live provider reliability.
- NWS API already fits the CAP lane for alerts and can return CAP content via content negotiation. NWS API explicitly
  points radar display users to separate radar/OGC services rather than treating `/api.weather.gov` as a radar tile
  source.

Mock or harness:

- Visual layer: browser-side raster/vector weather tiles or WMS are allowed without graph ingestion when they are
  human context only.
- Alert layer: continue using CAP-style append-evidence for public alerts and warnings.
- Tactical layer: backend component queries point, area, trajectory, or corridor weather near active assets,
  incident zones, or planned routes and publishes bounded forecast/observation payloads.
- The current backend component gate is fixture-backed preflight evidence only; live Open-Meteo, OGC EDR, MSC GeoMet,
  NWS forecast/observation, cache/stale behavior, and provider reliability remain separate gates.

Indexing profile pressure:

- Localized weather observations and forecasts that affect routing or asset safety are `signal`.
- Weather alert lifecycle and route-safety decisions are `control`.
- Weather advisory text and forecast discussions are `content`.
- Provider request/response diagnostics and replay records are `trace`.

First acceptance gate:

- Given the deterministic Open-Meteo-shaped point fixture, SemOps parses wind, gusts, precipitation, visibility,
  pressure, temperature, weather code, timestamp, point location, provider, query shape, and units without graph
  writes.
- Given the synthetic OGC EDR-shaped point CoverageJSON fixture, SemOps parses the equivalent tactical weather
  variables without graph writes, live provider claims, or OGC conformance claims.
- Given the synthetic OGC EDR-shaped area, trajectory, and corridor fixtures, SemOps parses simple spatial query
  geometry, corridor dimensions, units, and tactical weather variables without graph writes, live provider claims,
  route-safety claims, or OGC conformance claims.
- Given weather component promotion, SemOps publishes raw and decoded provider-shaped weather forecasts through
  SemStreams registered BaseMessage payloads and NATS stream ports before tactical graph projection, live-provider
  claims, or route-safety decisions. [done for Open-Meteo-shaped and OGC EDR-shaped point fixtures]
- Given tactical weather graph-contract promotion, SemOps represents each bounded weather variable/time sample as a
  source-partitioned `weather_observation` signal entity with provider, query shape, geometry, valid time, model time,
  freshness, unit, value, confidence, and source reference. [done for pure planner, graph writer, and point-payload
  projector component evidence]
- Given weather graph-projector component promotion, SemOps consumes decoded point forecasts from a declared stream
  port and writes through declared SemStreams graph request ports with max-observation and freshness config gates.
  [done for point-payload component and opt-in hosted runtime composition]
- Given opt-in stack readback, the Caddy-routed COP snapshot exposes graph-backed local point-forecast weather
  observations and component/runtime flow evidence without claiming live provider support or tactical UI semantics.
  [done behind `SEMOPS_COP_SMOKE_WEATHER_ENABLED=true`]
- Given UI rendering, the COP browser shows a selectable weather-observation point and inspector evidence from the
  snapshot contract without claiming live provider support, visual weather tiles, or route-safety decisions. [done]
- Given future OGC EDR-shaped fixtures, SemOps should parse selected broader query shapes before claiming
  standards-facing tactical weather interop beyond point, area, trajectory, and corridor retrieval.
- Given projection, localized tactical weather writes source-partitioned governed evidence and does not overwrite CAP
  hazard or operator task state.
- Given UI rendering, visual weather tiles may be configured in the browser without implying backend weather
  ingestion or radar hosting.
- Given routing or drone-safety use, the weather source, model time, query shape, stale policy, and confidence are
  visible to the operator before any action recommendation is made.

Known gaps:

- No OGC EDR radius, cube, item, location, or instance fixture exists yet.
- No OGC EDR spatial runtime component payload, route-weather model, or UI tactical-weather layer exists yet.
- No OGC EDR conformance/ETS run, live EDR server capture, or standards-facing bridge test exists yet.
- No live weather provider integration exists yet.
- No live weather HTTP poller, default-stack weather hosting, cache/stale policy, or UI tactical layer exists yet.
- The current hosted weather runtime path is fixture-backed point-forecast evidence only.
- The current weather COP API readback path is source/provenance evidence only; it is not a tactical weather map layer.
- No weather routing/safety rule is accepted yet.
- No visual tile source has passed license/cache/reliability review.

### DJI

Status: critical HADR drone/vendor layer with first synthetic parser fixture and SemStreams component-flow evidence.

Compliance and source evidence:

- DJI should be treated as a proprietary product integration, not a standards feed. It may arrive through SDK/cloud
  surfaces, controller/dock/session APIs, recorded files, subtitles, or media streams depending on deployment.
- DJI Onboard SDK documentation positions DJI platform integration around drone information, control, payload/camera,
  video image analysis, and onboard applications.
- DJI Mobile SDK documentation exposes application surfaces around flight controller, camera, gimbal, smart battery,
  missions, media management, and video stream decoding samples.
- DJI Payload SDK documentation confirms payload-device development is a separate integration lane from mobile,
  onboard, cloud, Windows, and payload SDK surfaces.
- DJI video is not automatically KLV/STANAG evidence. Some paths may expose vendor telemetry, subtitles, sidecar
  metadata, or live streams rather than MISB ST 0601 KLV packets.
- SemOps now carries a synthetic DJI-shaped parser fixture at `fixtures/dji/telemetry-media.json`.
- `pkg/adapters/dji` parses aircraft pose, altitude, heading, speed, battery, gimbal/camera state, media references,
  source identity, and command-authority posture without graph writes.
- `internal/components/dji` wraps the fixture as a SemStreams file input component and decoder processor with
  registered payloads, file/NATS ports, config schema, health, and flow metrics.

Mock or harness:

- Start with recorded or synthetic DJI-shaped telemetry and media-reference fixtures.
- The current fixture is synthetic SemOps contract evidence only, not captured SDK, Cloud API, flight-log, subtitle,
  media metadata, or product-compatibility evidence.
- Keep DJI telemetry, media references, command authority, and graph projection as separate seams.
- Keep the current component flow preflight-only until live ingress surface, replay store, ownership, and safety
  posture are reviewed.
- Use SemSource or a media sidecar only for generic storage/reference/track extraction; SemOps owns DJI semantics.

Indexing profile pressure:

- Vehicle pose, gimbal orientation, camera/sensor state, and selected telemetry are `signal`.
- Mission/session/control state is `control`.
- Operator notes, detected objects, and media annotations are `content`.
- Raw vendor payload references, media extraction logs, and replay records are `trace`.

First acceptance gate:

- Given a DJI-shaped telemetry fixture, SemOps can parse vehicle position, heading, altitude, gimbal/camera state,
  timestamps, source identity, and freshness without graph writes. [done for synthetic fixture]
- Given a DJI media fixture or reference, SemOps records media refs and bounded metadata without embedding video in
  triples. [done for synthetic fixture]
- Given DJI command-authority posture, SemOps records the authority mode, holder, local override requirement, and
  remote-command disabled posture as data only. [done for synthetic fixture]
- Given DJI component promotion, SemOps publishes raw and decoded DJI-shaped telemetry through SemStreams registered
  BaseMessage payloads and NATS stream ports without graph writes, owner claims, or runtime live-bridge claims. [done
  for synthetic fixture]
- Given DJI video metadata, SemOps routes it to a DJI or generic media decoder path unless the source actually emits
  KLV/MISB packets.
- Given any DJI command/control work, command authority, local override, and safety policy are reviewed before graph
  projection or driver actuation.

Known gaps:

- No legal representative DJI telemetry/media fixture has been selected.
- No DJI replay store exists beyond the committed synthetic JSON fixture.
- No DJI SDK/cloud integration strategy has been chosen.
- No live DJI bridge, media relay, or command authority path exists.
- No DJI graph projector, runtime wiring, ownership claim, or UI layer exists yet.
- No DJI product support or compatibility claim is allowed yet.

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
- CS API tasking is an impedance-mismatch boundary. The bridge should accept or reject quickly, persist governed
  command intent or desired state, and let native drivers reconcile tactical execution asynchronously.

First acceptance gate:

- Egress: a SemOps asset/platform, hosted sensor, datastream, observation, deployment, and system event can be
  projected through SemConnect and checked by the conformance harness without weakening SemOps ownership rules.
- Ingress: a CS API source fixture can be mapped into SemOps canonical COP state without bypassing born-first writes,
  provenance, freshness, or command authority.
- Tasking: CS API command/control input routes through SemOps command authority and native safety controls rather
  than directly mutating feed-owned state.
- Tasking edge cases: desired-state records include source authority, priority, TTL/deadline, idempotency key,
  correlation ID, local-operator override policy, cancellation/supersession semantics, and status mapping before any
  live native driver is connected.
- Tasking status: native acknowledgements, partial execution, timeout, stale-command rejection, duplicate-command
  rejection, link loss, and upstream polling/subscription status are modeled as graph-backed state or evidence rather
  than an open HTTP request.
- Replay: the same fixture can be replayed deterministically so bridge drift is visible before conformance claims.

### ADS-B

Status: later air-picture feed with OpenSky-shaped parser, deterministic replay, hosted-adapter seam, HTTP component
package, opt-in app-runtime wiring, optional structural scenario replay, local HTTP provider fixture smoke,
current-state projection, owner registration, and graph readback evidence.

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
- `internal/components/adsb` provides the first SemStreams component promotion for OpenSky-compatible HTTP polling:
  `HTTPClientPort`, `TimerPort`, raw/decoded `message.BaseMessage` payload registry entries, stream ports, graph
  request ports, replay capture, stale-source health, and local provider-shaped HTTP fixture tests.
- `cmd/semops` can wire ADS-B as an opt-in SemStreams component flow with `SEMOPS_ADSB_ENABLED=true`,
  `SEMOPS_ADSB_HTTP_URL`, replay capture, raw-lane caps, stale-source config, and token-backed graph writes.
- `cmd/semops-feed-fixtures` serves deterministic OpenSky-compatible `/adsb/states` JSON for local Compose smoke
  tests, and `scripts/cop-stack-smoke.sh` enables ADS-B against that fixture by default.
- The one-command hosted stack smoke now asserts Prometheus component health and flow samples for the opt-in ADS-B
  HTTP poller, decoder, and projector chain through SemOps `/metrics`.
- `cmd/semops-scenario-runner` adds ADS-B snapshots only when `SEMOPS_SCENARIO_ADSB_FIXTURE=true`; the Compose service
  passes that flag through but defaults it to false.
- The scenario runner appends `semops.feed.adsb` ownership only for that opt-in path so structural ADS-B graph writes
  use SemStreams minted owner tokens.
- The ADS-B app-runtime path appends `semops.feed.adsb` ownership only when enabled; the default runtime still keeps
  ADS-B off and does not claim live OpenSky reliability, readsb/dump1090 file tailing, receiver TCP/UDP, or ASTERIX
  support.
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
- Given an OpenSky-compatible HTTP endpoint, SemOps can run a SemStreams input -> decoder processor -> graph projector
  processor chain against a local provider fixture, with payload registry and flowgraph edges visible before any live
  service claim.
- Given `SEMOPS_ADSB_ENABLED=true`, the hosted app composes the ADS-B HTTP poller -> decoder -> graph-projector flow,
  captures optional replay, and mints `semops.feed.adsb` ownership through the runtime ownership registration path.
- Given the local Compose fixture provider, the one-command stack smoke enables ADS-B, polls the Caddy-routed COP
  snapshot, and verifies an ADS-B track written through the HTTP component path is visible by prefix discovery.
- Given `SEMOPS_SCENARIO_ADSB_FIXTURE=true`, the hosted scenario runner replays two ADS-B snapshots through the
  adapter seam and token-backed graph writes without live network access.
- Given a projected ADS-B aircraft state, SemOps writes current-state track evidence without source-asset or
  cross-source association edges and reads it back through prefix discovery.
- Next gate: prioritize local receiver/readsb/dump1090 input components, authenticated OpenSky option handling, or
  ASTERIX only after rate, replay, and backpressure expectations are explicit.
- Component gate: `internal/components/adsb` exists for OpenSky-compatible HTTP polling; receiver and ASTERIX ingress
  still require separate input components when chosen.

### SAPIENT

Status: JSON and binary descriptor preflight, raw replay, preflight input/decoder components, opt-in app-runtime
preflight wiring, local decoded-stream smoke, a narrow absolute-location detection projection/readback gate, and an
explicitly graph-gated SAPIENT projector component exist. Harness qualification, service-mode review, and broader
message semantics are still required before product support or conformance claims.

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
  harness or another accepted authority. The portable suite is evaluated as a useful ecosystem contribution and
  developer-preflight gate, not an MVP blocker or a compliance substitute.

Local assets:

- `pkg/adapters/sapient` parses representative Dstl-harness-shaped JSON preflight fixtures for registration, status
  report, detection report, and task acknowledgement messages.
- `pkg/adapters/sapient` embeds official Dstl BSI Flex 335 v2 proto sources and compiles them with
  `github.com/bufbuild/protocompile` for dynamic descriptor-based binary `SapientMessage` decode.
- Generated SAPIENT Go bindings are deliberately deferred. Dynamic descriptors are sufficient for current preflight,
  raw replay, component-flow, and absolute-location graph projection work; generated bindings should re-enter only for
  service mode, outbound tasking, exact typed protobuf round trips, broad message coverage, or measured performance
  need. If that happens, use a reproducible buf-based workflow.
- `pkg/adapters/sapient` now has a bounded raw lane and JSON Lines replay store for JSON and protobuf payloads; replay
  decodes through the same preflight boundary rather than treating captured bytes as normalized graph state.
- `go test ./pkg/adapters/sapient` rejects malformed JSON and binary required-field cases before graph writes.
- `internal/components/sapient` now provides SemStreams SAPIENT HTTP input, decoder processor, and graph-projector
  processor components. The input/decoder path has `HTTPClientPort`, `TimerPort`, registered raw/decoded payloads,
  stream ports, replay capture, and stale-source health. The projector path has SemStreams graph request ports and is
  limited to the reviewed absolute-location detection contract.
- `cmd/semops` can run that HTTP input -> decoder preflight chain behind `SEMOPS_SAPIENT_ENABLED=true` with explicit
  URL, encoding, stale-source settings, raw-lane caps, and optional replay capture.
- `cmd/semops` can additionally compose decoded SAPIENT messages into the graph projector only when
  `SEMOPS_SAPIENT_GRAPH_ENABLED=true`. Runtime ownership registration for `OwnerSAPIENT` occurs only under that second
  gate, and the default fixture URL remains the task-ack decoded-stream smoke rather than a track-producing detection
  feed.
- `cmd/semops-feed-fixtures` serves deterministic SAPIENT task-ack JSON for local Compose smoke tests, and
  deterministic absolute-location detection JSON for graph-projection development. `scripts/cop-stack-smoke.sh`
  verifies the hosted preflight chain publishes a typed decoded payload on the declared SAPIENT output stream.
- `pkg/cop` now defines `OwnerSAPIENT` and a source-partitioned `signal` track contract for absolute-location
  detection reports only.
- `internal/projectors/sapient` plans create/update graph mutations for
  `LOCATION_COORDINATE_SYSTEM_LAT_LNG_DEG_M` WGS84 detection reports and rejects range/bearing, UTM, unsupported
  datum, or invalid latitude/longitude inputs.
- `internal/api/cop` can read prefix-discovered SAPIENT tracks back into COP snapshots and source-health state.
- No local SAPIENT harness run, product service adapter, tasking surface, association model, range/bearing conversion,
  or UTM conversion exists. Generated Go bindings are deferred by decision rather than missing as an MVP blocker.

Mock or harness:

- Start with parser-only fixtures from official protobuf/sample-message evidence.
- Treat the current JSON and binary descriptor preflight as developer evidence only; generated Go bindings remain
  optional and deferred, while Dstl harness execution remains a separate gate.
- Run or qualify the Dstl BSI Flex 335 v2 Test Harness before using compliance language.
- Track a portable preflight harness as developer evidence, not official compliance, until accepted externally.
- Treat Apex middleware as an interop and service-shape reference, not as a substitute for SemOps ownership,
  provenance, freshness, and command-authority contracts.

Indexing profile pressure:

- Detections and tracks are usually `signal`.
- Sensor tasking, collection plans, and alert state are `control`.
- Native decode traces are `trace`.
- Absolute-location reports are the first safe projection candidate. Range/bearing detections require source sensor
  pose, reference frame, and uncertainty before they can become global coordinates.
- Associated detections and derived links are fusion or evidence outputs, not source-owned SAPIENT adapter truth.

First acceptance gate:

- Given BSI Flex 335 v2-aligned fixtures, malformed messages are rejected before graph writes and valid detections
  remain parser evidence until the projection ownership review approves a narrow graph entity model.
- Given binary BSI Flex 335 v2 payloads, the embedded descriptor toolchain decodes `SapientMessage` before SemOps
  claims protobuf preflight support.
- Given captured JSON or protobuf SAPIENT payloads, SemOps can replay exact native bytes through the parser boundary
  without writing graph state or claiming hosted SAPIENT support.
- Given an HTTP source of SAPIENT JSON or protobuf bytes, SemOps can run a SemStreams input -> decoder processor chain
  against local fixtures, producing raw and decoded preflight streams without graph writes or owner claims.
- Given `SEMOPS_SAPIENT_ENABLED=true` and `SEMOPS_SAPIENT_GRAPH_ENABLED=false`, the hosted app composes only the
  SAPIENT preflight HTTP input and decoder, captures optional replay, and avoids SAPIENT owner registration or decoded
  graph projector subscriptions.
- Given `SEMOPS_SAPIENT_ENABLED=true`, `SEMOPS_SAPIENT_GRAPH_ENABLED=true`, and a detection-producing SAPIENT source,
  the hosted app registers `OwnerSAPIENT`, composes the decoded-message projector component, writes born-first
  absolute-location detection track state, and still rejects tasking, association, UTM, and range/bearing semantics.
- Given the local Compose fixture provider, the one-command stack smoke enables SAPIENT and observes a typed decoded
  task-ack payload on `semops.feed.sapient.decoded` without adding SAPIENT graph ownership or projector subscriptions.
- The same smoke asserts Prometheus component health and flow samples for the SAPIENT HTTP input and decoder through
  SemOps `/metrics`; this remains preflight telemetry, not product support or conformance evidence.
- Given `SEMOPS_COP_SMOKE_SAPIENT_GRAPH_ENABLED=true`, the one-command stack smoke points SAPIENT at
  `/sapient/detections`, enables the graph projector, asserts Caddy-routed SAPIENT track readback through
  `GET /api/cop/snapshot`, and expects SAPIENT HTTP input, decoder, and projector component flow in Prometheus and
  `GET /api/cop/runtime`. This is fixture-backed absolute-location projection evidence only.
- Given an absolute-location SAPIENT detection report using `LOCATION_COORDINATE_SYSTEM_LAT_LNG_DEG_M` and WGS84
  datum, SemOps can plan signal-profiled source-owned track mutations and read equivalent graph state through
  `GET /api/cop/snapshot` prefix discovery. [done for projector/API tests only]
- Given UTM or range/bearing SAPIENT detections, SemOps rejects projection until coordinate conversion, source sensor
  pose, reference frame, and uncertainty policy are accepted. [done]
- Given SemOps-generated SAPIENT messages, the Dstl v2 Test Harness result is recorded before any SAPIENT
  compliance claim appears in demo materials.
- Given a future product-hosted SAPIENT feed, SemOps promotes beyond the current opt-in component flow only after
  service mode, command authority, association, coordinate-conversion policy, and harness scope are reviewed. The
  current graph-enabled runtime path is absolute-location detection projection only and must not be read as product
  hosted-service support.

### KLV/STANAG 4609

Status: stretch proof spike.

Compliance and parser evidence:

- jMISB implements several MISB standards, including ST 0601 UAS Datalink, ST 0805 KLV-to-CoT conversion, and
  ST 1402 MPEG-2 transport stream support.
- `klvdata` is a Python library for parsing MISB ST 0601 KLV metadata from STANAG 4609-compliant MPEG-TS streams.
- `klvdata` documents a small binary packet sample and an FFmpeg workflow for extracting KLV from the public
  `Day Flight.mpg` MPEG-TS sample. Treat that as smoke evidence only until license/provenance and cache policy are
  reviewed.
- `docs/klv-public-sample-smoke.md` records the opt-in local public-sample smoke path. It requires a local sample
  path plus source/provenance environment variables and never downloads or vendors public media.
- FFmpeg can map data streams explicitly; KLV extraction commands must not assume data streams are selected
  automatically.

Local assets:

- SemSource has draft media support and a video handler that extracts metadata and keyframes with ffprobe/ffmpeg.
- SemSource can store metadata-only video entities, and can store binary files when a media store is configured.
- The current video handler streams hashing, but reads the full video into memory when binary storage is enabled.
- SemOps does not currently have a real legal KLV, STANAG 4609, or SKG binary fixture to hand to SemSource.
- `internal/components/klv` declares the first registered KLV payload schemas: `semops.klv_media_ref.v1`,
  `semops.klv_packet.v1`, and `semops.klv_misb0601_frame.v1`.
- `internal/components/klv` also declares the first component skeleton: media-reference input, KLV demux processor,
  MISB ST 0601 decode processor, and governed projector processor with file, stream, and graph request ports.
- `internal/components/klv` includes the first Go-native deterministic MISB ST 0601 local-set decoder for bounded
  packet bytes, and the decoder component can publish decoded-frame BaseMessages when configured with a SemStreams
  bus.
- `internal/components/klv` includes a fixture-grade FFmpeg/ffprobe demux worker path. The demux component can consume
  registered media-ref BaseMessages, select an explicit data stream with ffprobe, extract bounded KLV bytes with
  FFmpeg `-map`, split concatenated MISB ST 0601 local sets into bounded packet BaseMessages, and publish each
  packet on the declared stream. Storage-reference-only media refs are accepted only when an explicit bounded
  materializer is supplied. This is not a production live-media or STANAG conformance claim.
- `internal/projectors/klv` can plan born-first, owner-token-fenced graph writes for source-partitioned
  `sensor_footprint` sensor/frame-center state. The contract includes sensor position, frame center,
  azimuth/elevation, media reference, packet reference, platform designation, and provenance, but not footprint
  polygons, video service support, or conformance claims.
- `internal/api/cop` and the Svelte COP UI now read back that governed KLV `sensor_footprint` state as selectable
  sensor points, frame-center points, and rays with a provenance inspector. This is the product-visible proof for
  sensor/frame-center evidence only, not a video player, footprint polygon, streaming-binary, or STANAG conformance
  claim.
- `cmd/semops-klv-fixture` can generate a tiny deterministic MPEG-TS fixture from SemOps-owned MISB ST 0601 truth
  data, and the generated media output is ignored rather than vendored.
- The one-command hosted stack smoke can opt into KLV with `SEMOPS_COP_SMOKE_KLV_ENABLED=true`, building the
  `media-tools` image target with FFmpeg, running the local-media KLV component flow, and verifying COP snapshot plus
  Prometheus/runtime readback. The default stack keeps KLV disabled and uses the production image target.
- `openspec/changes/revive-cop-product/reviews/2026-06-23-klv-claim-language-review.md` records the allowed
  engineering-support wording for the tested MISB ST 0601 subset and blocks STANAG conformance, certification,
  full-parser, live-media, footprint-polygon, and general streaming-binary claims.

Mock or harness:

- Treat SemSource as a candidate media sidecar, not a proven KLV solution.
- SemSource's governed SemStreams migration leaves KLV/MISB/STANAG/SAPIENT/SKG interpretation to SemOps or a
  SemOps-owned worker. That is the product boundary SemOps should preserve.
- Demux does not need to happen in SemSource for the MVP. SemOps owns the `media_ref -> demux -> decode -> project`
  flow so MPEG-TS/KLV behavior, MISB decode, and support claims stay in the COP product.
- A future SemSource or media-stack component may provide generic media-track extraction or byte-range materialization,
  especially for shared FFmpeg/GStreamer/MediaMTX needs, but it should emit generic media/data-track evidence rather
  than SemOps product claims.
- If SemSource needs an immediate fixture for its governed SemStreams migration, use a deliberately synthetic binary
  fixture and label it as storage/governance proof only.
- The synthetic fixture may prove raw bytes by reference, storage hashes, governed metadata entities, owner-token
  writes, indexing profiles, and memory-bounded handling.
- The synthetic fixture must not be described as KLV, STANAG 4609, SAPIENT, SKG, streaming-binary, or protocol
  conformance evidence.
- SemOps owns the product fixture ladder: public KLV sample smoke, deterministic MISB ST 0601 fixture, and any formal
  STANAG 4609 conformance path.
- First prove video metadata and keyframe ingestion on a small synthetic or public fixture.
- Then use a public KLV sample only as demux/parser smoke after license and provenance review.
- Use the deterministic MISB ST 0601 fixture as the first engineering-support acceptance gate: truth JSON to generated
  KLV packet bytes to decoded output. MPEG-TS wrapping is now the local container-proof step, not a broader live-media
  or conformance claim.
- Public examples commonly used by open-source FMV/KLV tooling plus deterministic fixtures are acceptable for
  demo-grade engineering support. Official STANAG 4609 conformance or certification stays blocked until someone funds
  a validator or lab effort with proper access. Demo copy must follow the KLV claim-language review rather than
  upgrading smoke evidence into conformance language.
- First parser strategy: keep the first spike Go-native and deterministic for the supported MISB ST 0601 local-set
  subset. Keep MPEG-TS demux behind the demux component boundary, use FFmpeg/ffprobe for the first fixture-grade
  extraction path, and defer GStreamer, `klvdata`, jMISB, or a Rust worker until public-sample smoke or production
  throughput proves the need.

Indexing profile pressure:

- Sensor footprints and extracted platform/sensor coordinates are `signal`.
- Clip, asset, and evidence package lifecycle is `control`.
- Frame/keyframe descriptions, OCR, and operator annotations are `content` when present.
- Packet/frame decode events are `trace`.
- Binary payloads stay in object storage or external media storage, not graph triples.

First acceptance gate:

- Given a SemOps-owned KLV/MISB worker design, the input is a SemSource storage reference or native media reference,
  the output is governed derived facts, and component telemetry/backpressure/memory bounds are declared.
- Given hosted KLV/MISB work, SemOps uses a SemStreams media-reference input component, demux processor, MISB decode
  processor, projector processor, and optional interop processors rather than a monolithic COP server feature.
- Given component promotion, KLV messages use registered `semops.klv.*` payloads and declared ports; graph writes
  happen only in the projector through SemStreams request ports.
- Given `go test ./internal/components/klv ./internal/contracts`, the first KLV payload schemas round-trip through
  SemStreams `message.BaseMessage`, enforce by-reference posture for packet payloads that do not carry bounded bytes,
  and connect media-ref -> demux -> decode -> projector through SemStreams flowgraph ports.
- Given `go test ./internal/components/klv`, the first deterministic MISB ST 0601 local-set decoder extracts frame
  time, platform designation, sensor position, frame center, azimuth, and elevation from bounded packet bytes without
  graph writes.
- Given the opt-in decoder worker path, a registered `semops.klv_packet.v1` BaseMessage produces a registered
  `semops.klv_misb0601_frame.v1` BaseMessage on the declared frame subject, not a graph mutation subject.
- Given a storage-reference-only packet payload, decode proceeds only when a configured packet materializer provides
  packet bytes under `max_packet_bytes`; the parser core remains bounded-byte only.
- Given the opt-in demux worker path, a registered `semops.klv_media_ref.v1` BaseMessage for a local file URI invokes
  ffprobe data-stream discovery, extracts bounded bytes with FFmpeg explicit `-map`, and publishes a registered
  `semops.klv_packet.v1` BaseMessage on the declared packet subject for each split MISB ST 0601 local set.
- Given `go test ./internal/app ./internal/stack`, hosted KLV runtime composition stays opt-in, registers the KLV
  `sensor_footprint` ownership contract only when enabled, exposes component health/flow through the common runtime
  metric source path, and writes graph plans through the SemStreams request writer with typed owner tokens.
- Given a storage-reference-only media ref, demux proceeds only when a configured materializer stages the media under
  `max_materialized_bytes`; the demux worker still enforces `max_extract_bytes`, `max_packet_bytes`, and
  `max_packets` before publishing packet payloads.
- Given local FFmpeg tooling is available, the deterministic KLV truth packet can be wrapped into MPEG-TS, demuxed
  back through the SemOps demux worker, and decoded to the original truth without network downloads or vendored media.
- Given `SEMOPS_COP_SMOKE_KLV_ENABLED=true bash scripts/cop-stack-smoke.sh`, the hosted stack generates a local
  deterministic MPEG-TS fixture, runs the opt-in KLV component flow through media-ref input, demux, decode, and graph
  projector stages, and asserts Caddy-routed COP snapshot readback plus Prometheus/runtime flow samples.
- Given a public video-plus-KLV smoke sample with documented license and provenance, the demo extracts plausible
  KLV metadata without calling the result deterministic correctness or conformance evidence.
- Given `SEMOPS_KLV_PUBLIC_SAMPLE_PATH`, `SEMOPS_KLV_PUBLIC_SAMPLE_SOURCE_URL`, and
  `SEMOPS_KLV_PUBLIC_SAMPLE_PROVENANCE` are set, the opt-in public sample smoke exercises the SemOps demux and
  decoder components against a local MPEG-TS KLV file without downloading or vendoring media.
- Given a deterministic MISB ST 0601 fixture, parsed sensor position, frame time, frame center, azimuth, elevation,
  and supported field presence match the source truth data within MISB integer quantization tolerances.
- Given `go test ./internal/projectors/klv ./internal/components/klv`, decoded KLV frames project born-first
  source-partitioned `sensor_footprint` state with `indexing_profile=signal`, owner token fencing, sensor position,
  frame center, media reference, packet reference, and no footprint polygon claim yet.
- Given KLV is made visible in the COP, the API/UI first reads back governed `sensor_footprint` graph state and renders
  only the sensor point, frame-center point, and sensor-to-frame-center ray with provenance for media reference,
  packet reference, decoded field inventory, warnings, provenance, and claim posture.
- Given the KLV layer is selected, the inspector labels public samples as smoke evidence and deterministic fixtures as
  engineering-support evidence for the tested MISB ST 0601 subset; it does not imply footprint polygon extraction,
  video service support, streaming-binary support, or STANAG 4609 conformance.
- Given any video-plus-KLV path, binary is stored by reference and memory-bounded behavior is proven before any
  "streaming binary" product claim.

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
- OGC API - Environmental Data Retrieval: <https://ogcapi.ogc.org/edr/>
- OGC API - Environmental Data Retrieval Standard 1.1: <https://docs.ogc.org/is/19-086r6/19-086r6.html>
- MSC GeoMet OGC API: <https://api.weather.gc.ca/>
- Open-Meteo API docs: <https://open-meteo.com/en/docs>
- DJI Onboard SDK overview: <https://developer.dji.com/onboard-sdk/documentation/introduction/homepage.html>
- OpenSky REST API: <https://openskynetwork.github.io/opensky-api/rest.html>
- OGC API - Connected Systems overview: <https://ogcapi.ogc.org/connectedsystems/>
- OGC Connected Systems SWG repository: <https://github.com/opengeospatial/ogcapi-connected-systems>
- jMISB: <https://github.com/WestRidgeSystems/jmisb>
- klvdata: <https://github.com/paretech/klvdata>
