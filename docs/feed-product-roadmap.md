# Feed Product Roadmap

## Status

Created on 2026-06-19 to keep feed work honest across two horizons:

- **Demo/MVP lane:** the narrow capability SemOps needs to prove governed feed fusion in the Phase 1 COP.
- **Full product lane:** the integration, operations, compliance, security, and scale work a production COP must
  eventually support.

This document is not permission to broaden the MVP. It is the guardrail that keeps narrow demo adapters from becoming
dead-end architecture.

For feeds that normally arrive through specialized servers, gateways, or collaboration infrastructure, the full product
lane may include SemOps creating its own SemStreams-backed service. The MVP lane should still start with bounded
adapters and fixtures, but the design must preserve a path from "consume this feed" to "host the product-grade service
for this feed" when SemOps needs to own that capability.

## Roadmap Rule

Every feed entry MUST answer four questions before it enters implementation:

1. What is the minimum demo path?
2. What does the full product eventually need?
3. What abstraction or boundary prevents the demo path from blocking the full product?
4. What are we explicitly not claiming yet?

Every feed also needs a service-promotion answer. A feed starts as a bounded adapter or fixture unless product forces
justify a SemOps-owned service or gateway. Promotion requires at least one concrete force: external protocol exposure,
auth/session/federation state, bidirectional command or tasking, scaling or placement isolation, durable collaboration
or replay state, secrets, cost, or failure-domain isolation. The MVP seam must keep parser, transport, session/service
state, command authority, and graph projection separate so promotion does not rewrite governed COP state contracts.

## Standards Positioning

SemOps uses a native core plus standards bridge strategy.

Native adapters are tactical necessities for HADR-style operations: agencies bring MAVLink, CoT, CAP, weather,
ADS-B, DJI/video, and vendor-specific feeds as they are. SemOps should ingest those formats directly, keep native
semantics where they matter, and project governed COP state into SemStreams without waiting for every source
capability to be represented first as a standards driver.

CS API is still important, but as an interface rather than the internal architecture. It buys decoupling for
standards-aware clients, a possible vendor plug-and-play path when systems already expose CS API, and a unified
standards-facing vocabulary for tasking/actuation. The MVP priority is read-side egress: SemOps publishes governed COP
state for consumers that require CS API through SemConnect. Full bidirectional support remains a product goal, but
write-side ingestion and tasking stay stretch scope until they can preserve SemOps ownership, provenance, freshness,
command authority, and indexing decisions.

The positioning statement is: SemOps moves fast natively, interoperates through standards, and isolates standards
change at the bridge instead of coupling the COP core to an external schema lifecycle.

OGC's Connected Systems API material reinforces this split. Part 1 covers feature resources such as systems,
procedures, deployments, sampling features, subsystems/components, and property definitions. Part 2 covers dynamic
data such as datastreams, observations, control streams, commands, command status, system events, streaming, and
snapshot mechanisms. Those are excellent edge contracts for standards-aware systems, but SemOps still chooses entity
ownership, provenance, freshness, command authority, and indexing profile inside the COP graph.

## Service Promotion Matrix

MAVLink:

- Demo/MVP boundary: local codec, bounded raw lane, UDP/replay adapter, and governed graph projector.
- Full product path: vehicle-link service with SITL/hardware profiles, multi-vehicle lifecycle, mission/command
  authority, reconnect, and stale-data policy.
- Promotion trigger: multi-vehicle operations, command workflows, hardware links, or auth/signing.
- Guardrail: keep autopilot protocol and command authority out of the graph projector.

DJI:

- Demo/MVP boundary: recorded or fixture-backed DJI telemetry/media-reference ingest that can show a common HADR
  drone source without claiming full DJI command/control or video exploitation.
- Full product path: DJI bridge service with SDK/cloud integration, vehicle identity, gimbal/camera state, media
  sessions, replay, command authority, and security review for vendor credentials.
- Promotion trigger: live DJI aircraft, remote-control/cloud session state, camera/gimbal control, or live media
  relay requirement.
- Guardrail: do not force DJI video metadata through the KLV/MISB decoder unless the source actually emits KLV.
  Shared media infrastructure may provide generic media references or track extraction, but SemOps owns DJI product
  semantics.

TAK/CoT:

- Demo/MVP boundary: native CoT parser, UDP/TCP fixture replay, and source-aware track/task/advisory projection.
- Full product path: SemStreams-backed SemOps TAK service with auth/certs, identity/session, subscriptions, GeoChat,
  markers, packages, and federation posture.
- Promotion trigger: collaboration state, subscriptions, federation, or TAK-compatible gateway ownership.
- Guardrail: do not hide TAK Server behavior inside the first CoT adapter.

CAP/EDXL:

- Demo/MVP boundary: parser, lifecycle fixtures, scenario replay, deterministic HTTP poller/decoder/projector
  component gate, NWS samples, and append-evidence hazard/advisory projection.
- Full product path: SemStreams input/processor component service with polling/webhook ingestion,
  update/cancel/expire lifecycle, geocodes, multilingual info, resources, retention, and audit.
- Promotion trigger: continuous public-alert ingestion, webhook exposure, vendor feed integration, or alert audit
  obligations.
- Guardrail: CAP evidence does not become authoritative hazard truth. The first component package proves hosted HTTP
  polling, decoding, and born-first graph projection contracts with local tests, but it is not a default live NWS
  service or CAP conformance claim. Hosted CAP pollers use SemStreams `HTTPClientPort` plus a sibling `TimerPort` when
  cadence-driven. Broader EDXL remains a separate roadmap lane until a product need selects a concrete EDXL family
  member and fixture set.

Weather:

- Demo/MVP boundary: keep CAP/public weather alerts on the existing CAP lane, use browser-side weather tiles for
  visual context when needed, and use a provider-shaped tactical weather fixture plus component flow before weather
  influences routing.
- Full product path: weather gateway with OGC API EDR, Open-Meteo or provider-specific polling, visual tile
  configuration, route/trajectory weather sampling, stale-data policy, cache, and source confidence.
- Promotion trigger: drone safety/routing logic, route planning, incident-area weather overlays, or multiple weather
  providers.
- Guardrail: visual raster/tiles do not need graph ingestion. Only localized tactical observations, forecasts,
  alerts, and decisions become governed graph evidence. The current UI can render fixture-backed localized
  `weather_observation` points as source/provenance evidence, but it still does not claim visual weather tiles,
  route-safety decisions, live provider reliability, or provider conformance.

CS API:

- Demo/MVP boundary: standards bridge over SemOps graph state through SemConnect once structural state exists.
- Full product path: standards gateway with ingress/egress, auth, pagination, subscriptions/delta export, tasking
  governance, and conformance evidence.
- Promotion trigger: federated consumers, CS API-native sources, or proposal/compliance obligations.
- Guardrail: do not route native feeds through CS API just to make the core look standard-shaped.

ADS-B:

- Demo/MVP boundary: recorded OpenSky-shaped JSON fixture parsing for aircraft current state, replay/projection, and a
  local-fixture-proved OpenSky-compatible HTTP component chain that the hosted app can run behind
  `SEMOPS_ADSB_ENABLED=true`.
- Full product path: opt-in live OpenSky, receiver/readsb/dump1090 service, rate limits, ASTERIX later, association,
  and airspace filters.
- Promotion trigger: shared-airspace vignette or live receiver requirement.
- Guardrail: raw receiver rows stay off the graph; association is a separate fusion claim.

SAPIENT:

- Demo/MVP boundary: BSI Flex 335 v2 JSON/protobuf preflight fixtures, bounded raw replay for exact payload evidence,
  preflight input/decoder components, and a narrow absolute-location detection projection/readback/runtime graph gate.
- Full product path: SAPIENT-facing service with versioned protobuf, sensor identity, detection lifecycle, tasking,
  fusion, deployment profiles, Apex/middleware interop, and eventual SemOps-owned SAPIENT service capability if
  product demand requires it.
- Promotion trigger: parser fixture success, projection ownership/indexing review, service-mode decision, plus a
  documented Dstl BSI Flex 335 v2 Test Harness run or explicit decision that the current phase is non-compliance demo
  evidence only.
- Guardrail: no guessed schema support and no SAPIENT compliance language without harness scope and result. Treat a
  future portable Linux/CI preflight suite as developer evidence until an accepted authority recognizes it. Keep the
  graph-producing runtime behind `SEMOPS_SAPIENT_GRAPH_ENABLED=true`, and keep tasking, association, UTM conversion,
  range/bearing projection, and service semantics behind separate gates.

KLV/STANAG 4609:

- Demo/MVP boundary: SemSource opaque-binary storage/governance proof plus SemOps-owned deterministic KLV packet and
  MPEG-TS fixture gates with binary-by-reference posture and bounded component handling.
- Full product path: media/KLV pipeline with demux, parser sidecar or native parser, object storage, keyframes,
  footprints, security, replay, and retention.
- Promotion trigger: real video metadata exploitation or streaming-binary proof need.
- Guardrail: SemSource proof is substrate evidence only; SemOps owns KLV/MISB/STANAG semantics and claims. Binary
  bytes stay in object/media storage, not graph triples.

## Feed Roadmaps

### MAVLink

Demo/MVP lane:
Generated/replay and UDP current-state ingest for heartbeat, position, attitude, battery, bounded raw lane,
born-first source asset and track graph writes. COMMAND_ACK readback now projects to control task state with a
born-first target edge, but this is command lifecycle evidence only. SemOps also has a product-owned command-intent
contract for desired tasking state with authority, priority, expiry, idempotency, and correlation fields. PX4/Gazebo
telemetry has hosted COP snapshot smoke pass evidence through a managed Compose-network route, and the hosted MAVLink
runtime can listen on both `14550` and `14540` for PX4 return paths. ArduPilot and MAVSDK/offboard have fail-closed
family-specific modes. Simulator command-control now has a `command-live-sim` gate that can run a reviewed external
transmitter and requires graph-visible `COMMAND_ACK` task evidence plus post-command track refresh before passing. For
MVP, that transmitter should stay to a single read-side command: `MAV_CMD_REQUEST_MESSAGE` for `AUTOPILOT_VERSION`.

Full product lane:
PX4/ArduPilot SITL and hardware profiles, MAVSDK smoke, UDP/TCP/serial transports, signed or authenticated links
where applicable, multi-vehicle lifecycle, command authority, command priority, TTL windows, mission state, reconnect,
and staleness behavior.

Boundary to preserve now:
Keep codec, raw lane, transport listener, projector, and command authority separate so simulator and hardware support
can grow without changing graph ownership. Simulator evidence must also name its family (`px4`, `ardupilot`,
`mavsdk`, `hardware`, or `other`) so one telemetry pass cannot be reused as another simulator or command-control
claim. Dedicated ArduPilot and MAVSDK/offboard lanes now fail closed until real family-specific sources are present.
Command-control preflight can record intended target, action, safety profile, local override, ACK, and post-state
polling posture. The live simulator gate exists, but command authority remains unclaimed until a reviewed transmitter
command passes the ACK/post-state snapshot smoke against a named simulator source. A 2026-06-27 PX4/Gazebo attempt
kept telemetry live but produced no decoded `COMMAND_ACK`, `COMMAND_LONG`, or `AUTOPILOT_VERSION` reply frames through
the hosted component chain. Do not expand the MVP into mission upload, mode control, arm/disarm, or offboard command
authority; read-side feed validation remains the priority.

Not claimed yet:
Full GCS/autopilot management, hardware certification, or complete mission-command product.

### DJI

Demo/MVP lane:
Recorded or fixture-backed DJI telemetry and media-reference ingest for a common HADR drone source. The first demo
shape should prove a DJI source card, vehicle/sensor state, media reference, and freshness/provenance without
claiming live DJI cloud or SDK control. SemOps now has a synthetic DJI-shaped parser fixture that preserves
telemetry, media references, and command-authority posture without graph writes, plus a SemStreams fixture input ->
decoder component flow that publishes raw and decoded BaseMessages without owner claims.

Full product lane:
DJI bridge service with SDK/cloud integration, aircraft identity, RC/dock/session state, gimbal/camera state,
recorded media, live media relay, replay capture, command authority, credential handling, and safety/security review.

Boundary to preserve now:
Keep DJI telemetry, command authority, media sessions, and graph projection separate. DJI video should enter as
generic media references or vendor metadata first; it must not be treated as KLV/MISB unless the source actually
emits KLV. If media relay becomes shared infrastructure, SemSource or a media sidecar may own generic track
extraction, while SemOps owns DJI semantics and claims. The current parser fixture is synthetic SemOps contract
evidence only; it is not captured SDK, Cloud API, flight-log, subtitle, media metadata, or DJI product compatibility
evidence. The current components are preflight file/decoder components, not a live bridge, replay service, graph
projector, or command path.

Not claimed yet:
Live DJI Cloud/API integration, DJI command/control, dock/RC session management, payload SDK compatibility, or
production video exploitation.

### TAK/CoT

Demo/MVP lane:
SemOps-local CoT parser, deterministic UDP/TCP fixture replay, born-first governed projection for operator dots,
markers, and GeoChat, freshness, provenance, and source-aware COP display.

Full product lane:
A SemStreams-backed SemOps TAK service when product need justifies it: CoT ingest/egress, certificate/auth
configuration, user/team context, subscriptions, GeoChat, markers, data packages or mission packages if required,
federation-aware deployment posture, and interoperability with deployed TAK Server or TAK-compatible gateways.

Boundary to preserve now:
Keep CoT codec, transport, identity/session, collaboration state, and graph projection separate so the MVP bridge can
evolve into a SemOps-owned TAK service instead of trapping server concerns inside an adapter.

Not claimed yet:
TAK Server-equivalent service in the MVP, federation services, full TAK mission package support, or public TAK
conformance.

### CAP/EDXL

Demo/MVP lane:
CAP XML parser with local lifecycle fixtures, deterministic HTTP poller, decoder, and projector components, NWS
samples, schema/consumer-rule validation, hazard/advisory evidence, expiry/staleness, and append-evidence ownership.
This lane is CAP-specific; broader EDXL variants stay out of Phase 1 unless a separate feed gate promotes them.
The later gate is tracked in `openspec/changes/revive-cop-product/feed-evidence/edxl-beyond-cap.md` so CAP evidence
does not accidentally imply broader EDXL support.

Full product lane:
Polling and webhook adapters for NWS/IPAWS/vendor feeds, alert update/cancel/expire lifecycle, multilingual
info/resources, geocode/circle/polygon handling, EDXL variants, retention, and audit policy.

Boundary to preserve now:
Keep tolerant ingest separate from strict hazard/advisory projection and never let CAP overwrite stricter tactical
source facts. CAP now has an HTTP poller input component, raw decoder processor, and born-first graph-projector
processor for hosted-polling shape. The app runtime can opt into that flow with `SEMOPS_CAP_ENABLED=true`, but it is
not wired as the default live service. `SEMOPS_CAP_REPLAY_PATH` can capture provider-shaped raw CAP XML replay records
from that opt-in path. The poller declares an `HTTPClientPort` for method, endpoint pattern, auth reference, contact
policy, and interface, plus a sibling `TimerPort` for cadence. Alert lifecycle and provider health remain separate
gates, though component health now degrades to `stale` when no fresh provider payload arrives within the configured
`stale_after` threshold.

Not claimed yet:
Full EDXL suite, default live NWS/IPAWS service, CAP consumer conformance, authoritative hazard truth, or
emergency-alerting authority.

### Weather

Demo/MVP lane:
Three separate layers: visual weather context in the browser, public alerts through the CAP lane, and localized
tactical weather telemetry for points, incident areas, or routes. The first backend slice should use deterministic
fixtures or a provider-shaped HTTP response before live provider claims. SemOps now has the first Open-Meteo-shaped
point forecast parser fixture plus a synthetic OGC EDR-shaped CoverageJSON point fixture for tactical weather
variables, plus synthetic OGC EDR-shaped area, trajectory, and corridor parser fixtures for spatial weather query
preflight. The SemStreams fixture input -> decoder component flow publishes raw and decoded BaseMessages for
provider-shaped point fixtures before graph writes, live provider support, spatial payload promotion, or conformance
language. SemOps now also has a governed `weather_observation` graph contract, graph writer, and point-forecast graph
projector component evidence for localized variable/time samples. The hosted app can now opt into a fixture-backed
file input -> decoder -> graph-projector runtime flow with owner-token registration and component metrics, but this
is still local fixture evidence rather than a live weather-provider claim. The COP API can read graph-backed
`weather_observation` entities, the one-command stack smoke can opt into a Caddy-routed weather fixture readback
check, and the UI can render a selectable point-observation evidence marker with provider/value/query/provenance
details. SemOps still does not render visual weather tiles, route-safety decisions, incident-area weather products, or
live provider reliability.

Full product lane:
OGC API EDR and provider-specific weather gateway with point, area, trajectory, corridor, and selected broader EDR
query support; cache and rate-limit behavior; stale-data policy; source confidence;
pressure/wind/visibility/precipitation profiles relevant to drone safety and routing; and optional visual tile
configuration for the UI.

Boundary to preserve now:
Visual raster or tile layers can stay browser-only unless they produce operator decisions or evidence. Tactical
weather that affects routing, safety, alerts, or fusion must become governed graph evidence with freshness and
provenance. CAP-style alerts remain append-evidence and must not overwrite stricter hazard truth. The current
Open-Meteo-shaped and OGC EDR-shaped parser fixtures plus the `weather_observation` graph contract are tactical
telemetry evidence only; they are not a weather gateway, OGC ETS/conformance result, live EDR capture, runtime spatial
component payload, or route-safety rule. The current hosted runtime path plus selectable point-observation UI marker is
an opt-in local point-forecast fixture chain, not an HTTP poller, live provider integration, cache/stale policy, visual
weather tile layer, spatial runtime payload, route-weather model, or provider reliability claim.

Not claimed yet:
Default live weather service reliability, weather-routing authority, provider conformance, broader EDR query support,
spatial weather runtime payloads, cache/stale policy, default-stack tactical weather runtime, or radar product hosting.

### CS API Read-Side Interop

Demo/MVP lane:
Curated SemOps graph state projected through SemConnect once structural graph state is stable and conformance harness
inputs exist. The first executable SemOps artifact is a read-side COP snapshot to CS API-shaped read model for Systems,
Datastreams, Observations, Deployments, and System Events. CS API ingress remains a later adapter boundary for systems
that already publish Systems, Datastreams, Observations, Deployments, or Events through CS API. A first narrow
write-side artifact now maps CS API Command and ControlStream command input into SemOps command intent only.

Full product lane:
Production standards gateway with auth, pagination, feature-resource coverage for systems, procedures, deployments,
sampling features, subsystems/components, and property definitions; dynamic-data coverage for datastreams,
observations, control streams, commands, command status, system events, snapshots, subscriptions or delta export,
schema evolution, bidirectional mapping, command/tasking governance, and conformance evidence per release.

Boundary to preserve now:
Keep CS API as a bridge over SemOps-owned graph state; do not route native feed ingestion through CS API just to make
the demo look standards-shaped. CS API ingress is acceptable when a source already speaks CS API, but it must enter the
same governed projection path as native adapters.

Not claimed yet:
Full OGC Connected Systems API product inside SemOps, replacing SemConnect, hosted write-side CS API service behavior,
non-command writes, native tasking, upstream command-status publication, or automatic support for every new sensor
because a CS API schema exists.

### ADS-B

Demo/MVP lane:
Recorded OpenSky-shaped JSON fixtures for aircraft current state, freshness, source, provenance, and bounded replay.
The first implemented slice is `pkg/adapters/adsb`, which parses `/states/all` snapshot fixtures and preserves
nullable position fields plus position-source quality before projection. Current slices now include deterministic
OpenSky snapshot replay, hosted snapshot ingest, source-partitioned ADS-B aircraft projection with `signal` indexing,
COP graph prefix readback, contract-mode structural scenario replay, and an OpenSky-compatible HTTP input -> decoder
-> graph-projector component package proved with local provider fixtures. The hosted app can compose that chain with
`SEMOPS_ADSB_ENABLED=true`, provider URL/stale/replay settings, raw-lane caps, and `semops.feed.adsb` ownership
registration only for the enabled flow. Product ADS-B evidence uses that hosted component path, not scenario-runner
graph replay.

Full product lane:
Optional runtime wiring for live OpenSky with rate-limit handling, local receiver/readsb/dump1090 paths, raw ADS-B or
ASTERIX later, association with MAVLink/SAPIENT/fusion tracks, and airspace filters.

Boundary to preserve now:
Keep raw receiver rows off the graph and project current aircraft state plus association evidence separately. ADS-B
owner registration is valid for token-backed structural replay and opt-in runtime polling; it is not a default live
feed or receiver-service claim. Do not treat `internal/components/adsb` as a default live service claim; it proves the
SemStreams component shape for OpenSky-compatible HTTP polling while readsb/dump1090 file tailing, receiver TCP/UDP,
and ASTERIX still need separate input components when chosen.

Not claimed yet:
Default live air-traffic feed reliability, default-enabled ADS-B service support, ASTERIX support, cross-source
aircraft association, or complete surveillance/radar processing.

### SAPIENT

Demo/MVP lane:
Official artifacts now exist: GOV.UK points to BSI Flex 335, Dstl protobufs, a BSI Flex 335 v2 Test Harness, and Apex
middleware. The first SemOps lane is parser-first: JSON preflight and descriptor-based binary protobuf preflight now
validate representative Dstl-harness-shaped messages before graph writes. Bounded raw capture and JSON Lines replay now
preserve exact JSON/protobuf payload bytes for repeatable preflight and future harness comparison. The first
SemStreams component lane has two gates: HTTP raw input plus decoder processor produce raw/decoded streams without
owner claims when only `SEMOPS_SAPIENT_ENABLED=true`, while the decoded-message graph projector is composed only when
`SEMOPS_SAPIENT_GRAPH_ENABLED=true`. The hosted app can run preflight with an explicit URL, encoding, stale-source
settings, raw-lane caps, and optional replay capture; graph mode additionally requires a detection-producing source.
The first governed projection lane is narrower still: absolute-location detection reports in
`LOCATION_COORDINATE_SYSTEM_LAT_LNG_DEG_M` with WGS84 datum can write and read back source-owned `signal` track state.
SemOps has qualified the Dstl BSI Flex 335 v2 Test Harness as out of MVP runtime scope because it is a
Windows/.NET/PostgreSQL 12/manual-configuration validation activity, not a normal Linux CI gate. This is not SAPIENT
product support, compliance, local harness success, or a SemOps-hosted SAPIENT service.

Full product lane:
SemOps-hosted SAPIENT-facing service if needed, with sensor/detection/tasking integration, versioned protobuf
compatibility, validator or compliance harness, sensor identity, detection lifecycle, multi-sensor fusion, deployment
profiles, interop with existing SAPIENT systems, optional Apex/middleware bridge behavior where useful, and a possible
portable SAPIENT preflight suite for Linux CI.

Boundary to preserve now:
Do not guess schemas; put official BSI Flex 335 v2 artifacts behind parser, session, and service boundaries before
graph projection. Treat Apex as an interop reference, while SemOps owns product semantics, graph ownership,
provenance, freshness, replay, and command authority. First graph projection should start with absolute-location
reports only unless source sensor pose, reference frame, and uncertainty make range/bearing conversion honest.
Associated detections and cross-source links belong to fusion or evidence contracts rather than the SAPIENT source
owner. UTM and other coordinate systems require a deliberate conversion/datum policy before projection.
`internal/components/sapient` and the opt-in app-runtime path are flow boundaries, not a SemOps-hosted SAPIENT product
service. The graph projector may write only the reviewed absolute-location detection contract and must stay separate
from future service/session/tasking behavior.

Not claimed yet:
SAPIENT conformance, product support, local test-harness success, portable-suite authority, full-message coverage,
hosted SAPIENT service support, tasking, association, UTM conversion, range/bearing conversion, or inferred schema
compatibility. Compliance-facing promotion needs a scoped Dstl harness or accepted authority result with environment,
corpus, and limitations recorded.

### KLV/STANAG 4609

Demo/MVP lane:
Proof spike with tiny video-plus-KLV fixture, extracted platform/sensor position or footprint,
binary-by-reference storage, and memory-bounded handling. The first implemented media path is fixture-grade:
registered media-ref BaseMessage to FFmpeg/ffprobe demux worker to bounded KLV packet BaseMessage to Go-native
MISB ST 0601 local-set decoder, with a SemOps-owned truth JSON fixture that generates deterministic KLV packet bytes
and an optional local FFmpeg smoke that wraps those bytes in MPEG-TS before demuxing and decoding them back to truth.
The demux worker can split concatenated MISB ST 0601 local sets into distinct packet messages and can use an explicit
bounded materializer for storage-reference-only media refs. The decoder component can also materialize
storage-reference-only packet payloads, but only through an explicit bounded packet materializer. Decoded fields are
asserted within MISB integer quantization tolerance. The graph projection contract covers KLV-owned sensor/frame-center
and decoded offset-corner footprint state. Opt-in hosted runtime wiring now exists for local media-ref input -> demux
-> decode -> projector flow, but default-stack live media ingress, broad footprint policy, media packages, and stronger
KLV/STANAG claims remain later gates.

Full product lane:
Production media/KLV pipeline with demux, parser sidecar or native parser, object storage, frame/keyframe evidence,
sensor footprints, streaming/media relay where needed, storage materialization policies, security review for binary
handling, replay, and retention.

Boundary to preserve now:
Treat SemSource as a candidate media sidecar, not a proven answer; keep binary bytes out of graph triples. DJI video
reinforces the need for generic media references and optional shared media-track extraction, but it does not make
KLV/STANAG demux a SemSource responsibility or turn DJI metadata into MISB/KLV evidence.

Not claimed yet:
Streaming-binary support, live media support, STANAG 4609 conformance, or production video exploitation.

## TAK Server Roadmap

TAK Server belongs on the product roadmap as a future SemOps/SemStreams-backed service capability, not as MVP scope.

Phase 1 should prove that SemOps can ingest CoT-shaped tactical events through deterministic local fixtures and project
governed COP state. The full product path should then add a TAK service boundary with configurable endpoints,
certificate/auth material, subscription/filter behavior, durable collaboration state, and replayable interoperability
tests against deployed TAK Server instances or compatible gateways.

SemOps should avoid putting TAK-server concerns inside the first CoT adapter. The safer default is: build the MVP as a
small governed feed boundary, then graduate shared collaboration, identity, session, subscription, and federation
behavior into a SemStreams-backed SemOps TAK service when the product needs to own that layer. Existing TAK Server
integration remains valuable as an interoperability and migration path, not the only long-term destination.

The same pattern applies to any feed whose "real" product shape is bigger than an adapter: start with bounded ingest,
preserve service seams, and only promote to a SemOps-owned server/gateway after fixtures, operators, and deployment
needs prove the value.

## Review Gates

- Before each feed enters the structural stack, review both lanes and confirm the MVP boundary still preserves the full
  product path.
- Before claiming compliance, link the exact harness, schema, official test, or documented interoperability run.
- Before broadening a feed, verify whether the new capability belongs in SemOps product space, SemStreams framework
  space, SemConnect CS API interop, SemSource media handling, or an external system integration.

## Source Links

- OGC API - Connected Systems overview: <https://ogcapi.ogc.org/connectedsystems/>
- OGC Connected Systems SWG repository: <https://github.com/opengeospatial/ogcapi-connected-systems>
- OGC API - Environmental Data Retrieval: <https://ogcapi.ogc.org/edr/>
- NWS API documentation: <https://www.weather.gov/documentation/services-web-api>
- MSC GeoMet OGC API: <https://api.weather.gc.ca/>
- Open-Meteo API docs: <https://open-meteo.com/en/docs>
- DJI Onboard SDK overview: <https://developer.dji.com/onboard-sdk/documentation/introduction/homepage.html>
