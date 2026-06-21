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

Native adapters are tactical necessities for HADR-style operations: agencies bring MAVLink, CoT, CAP, ADS-B, video,
and vendor-specific feeds as they are. SemOps should ingest those formats directly, keep native semantics where they
matter, and project governed COP state into SemStreams without waiting for every source capability to be represented
first as a standards driver.

CS API is still important, but as an interface rather than the internal architecture. It buys decoupling for
standards-aware clients, a possible vendor plug-and-play path when systems already expose CS API, and a unified
standards-facing vocabulary for tasking/actuation. The bridge should be bidirectional: SemOps can consume CS API from
systems that speak it and publish CS API for consumers that require it. The bridge must preserve SemOps ownership,
provenance, freshness, command authority, and indexing decisions.

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
  cadence-driven.

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

- Demo/MVP boundary: parser-only BSI Flex 335 v2 protobuf fixtures and optional Dstl harness qualification before
  graph projection, with bounded raw replay for exact JSON/protobuf payload evidence.
- Full product path: SAPIENT-facing service with versioned protobuf, sensor identity, detection lifecycle, tasking,
  fusion, deployment profiles, Apex/middleware interop, and eventual SemOps-owned SAPIENT service capability if
  product demand requires it.
- Promotion trigger: parser fixture success, projection ownership/indexing review, service-mode decision, plus a
  documented Dstl BSI Flex 335 v2 Test Harness run or explicit decision that the current phase is non-compliance demo
  evidence only.
- Guardrail: no guessed schema support and no SAPIENT compliance language without harness scope and result. Treat a
  future portable Linux/CI preflight suite as developer evidence until an accepted authority recognizes it. Do not add
  `OwnerSAPIENT`, projection writes, or hosted components before entity semantics and service mode are reviewed.

KLV/STANAG 4609:

- Demo/MVP boundary: tiny video-plus-KLV proof spike with binary-by-reference and memory bounds.
- Full product path: media/KLV pipeline with demux, parser sidecar or native parser, object storage, keyframes,
  footprints, security, replay, and retention.
- Promotion trigger: real video metadata exploitation or streaming-binary proof need.
- Guardrail: binary bytes stay in object/media storage, not graph triples.

## Feed Roadmaps

### MAVLink

Demo/MVP lane:
Generated/replay and UDP current-state ingest for heartbeat, position, attitude, battery, bounded raw lane,
born-first source asset and track graph writes.

Full product lane:
PX4/ArduPilot SITL and hardware profiles, MAVSDK smoke, UDP/TCP/serial transports, signed or authenticated links
where applicable, multi-vehicle lifecycle, command authority, mission state, reconnect, and staleness behavior.

Boundary to preserve now:
Keep codec, raw lane, transport listener, projector, and command authority separate so simulator and hardware support
can grow without changing graph ownership.

Not claimed yet:
Full GCS/autopilot management, hardware certification, or complete mission-command product.

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

### CS API Bidirectional Interop

Demo/MVP lane:
Curated SemOps graph state projected through SemConnect once structural graph state is stable and conformance harness
inputs exist. CS API ingress remains a later adapter boundary for systems that already publish Systems, Datastreams,
Observations, Deployments, or Events through CS API.

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
Full OGC Connected Systems API product inside SemOps, replacing SemConnect, or automatic support for every new sensor
because a CS API schema exists.

### ADS-B

Demo/MVP lane:
Recorded OpenSky-shaped JSON fixtures for aircraft current state, freshness, source, provenance, and bounded replay.
The first implemented slice is `pkg/adapters/adsb`, which parses `/states/all` snapshot fixtures and preserves
nullable position fields plus position-source quality before projection. Current slices now include deterministic
OpenSky snapshot replay, hosted snapshot ingest, source-partitioned ADS-B aircraft projection with `signal` indexing,
COP graph prefix readback, opt-in structural scenario replay with `SEMOPS_SCENARIO_ADSB_FIXTURE=true`, and an
OpenSky-compatible HTTP input -> decoder -> graph-projector component package proved with local provider fixtures.
The hosted app can compose that chain with `SEMOPS_ADSB_ENABLED=true`, provider URL/stale/replay settings, raw-lane
caps, and `semops.feed.adsb` ownership registration only for the enabled flow.

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
middleware. The first SemOps lane is parser-only: JSON preflight and descriptor-based binary protobuf preflight now
validate representative Dstl-harness-shaped messages before graph writes. Bounded raw capture and JSON Lines replay now
preserve exact JSON/protobuf payload bytes for repeatable preflight and future harness comparison. The first
SemStreams component lane is preflight-only: HTTP raw input plus decoder processor produce raw/decoded streams without
graph ports, owner claims, or product support wording.

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
owner. `internal/components/sapient` is a preflight flow boundary, not a SemOps-hosted SAPIENT product service.

Not claimed yet:
SAPIENT conformance, product support, local test-harness success, portable-suite authority, full-message coverage, or
inferred schema compatibility.

### KLV/STANAG 4609

Demo/MVP lane:
Proof spike with tiny video-plus-KLV fixture, extracted platform/sensor position or footprint,
binary-by-reference storage, and memory-bounded handling.

Full product lane:
Production media/KLV pipeline with demux, parser sidecar or native parser, object storage, frame/keyframe evidence,
sensor footprints, security review for binary handling, replay, and retention.

Boundary to preserve now:
Treat SemSource as a candidate media sidecar, not a proven answer; keep binary bytes out of graph triples.

Not claimed yet:
Streaming-binary support, STANAG 4609 conformance, or production video exploitation.

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
