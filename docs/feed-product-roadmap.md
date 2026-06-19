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
CAP XML parser with OASIS examples and NWS samples, schema/consumer-rule validation, hazard/advisory evidence,
expiry/staleness, and append-evidence ownership.

Full product lane:
Polling and webhook adapters for NWS/IPAWS/vendor feeds, alert update/cancel/expire lifecycle, multilingual
info/resources, geocode/circle/polygon handling, EDXL variants, retention, and audit policy.

Boundary to preserve now:
Keep tolerant ingest separate from strict hazard/advisory projection and never let CAP overwrite stricter tactical
source facts.

Not claimed yet:
Full EDXL suite, authoritative hazard truth, or emergency-alerting authority.

### SemConnect CS API Egress

Demo/MVP lane:
Curated SemOps graph state projected through SemConnect once structural graph state is stable and conformance harness
inputs exist.

Full product lane:
Production standards gateway with auth, pagination, deployment/system/datastream/observation coverage, subscriptions
or delta export, schema evolution, and conformance evidence per release.

Boundary to preserve now:
Keep CS API as egress/view over SemOps-owned graph state; do not route raw feed ingestion through CS API to make the
demo look standards-shaped.

Not claimed yet:
Full OGC Connected Systems API product inside SemOps or replacing SemConnect.

### ADS-B

Demo/MVP lane:
Recorded OpenSky-shaped JSON fixtures for aircraft current state, freshness, source, provenance, and bounded replay.

Full product lane:
Optional live OpenSky with rate-limit handling, local receiver/readsb/dump1090 paths, raw ADS-B or ASTERIX later,
association with MAVLink/SAPIENT/fusion tracks, and airspace filters.

Boundary to preserve now:
Keep raw receiver rows off the graph and project current aircraft state plus association evidence separately.

Not claimed yet:
Live air-traffic feed reliability, ASTERIX support, or complete surveillance/radar processing.

### SAPIENT

Demo/MVP lane:
No implementation until authoritative ICD/protobuf/schema/sample/validator evidence exists. The first lane is
parser-only fixtures once artifacts are found.

Full product lane:
SemOps-hosted SAPIENT-facing service if needed, with sensor/detection/tasking integration, versioned protobuf
compatibility, validator or compliance harness, sensor identity, detection lifecycle, multi-sensor fusion, deployment
profiles, and interop with existing SAPIENT systems.

Boundary to preserve now:
Do not guess schemas; put authoritative artifacts behind parser, session, and service boundaries before graph
projection.

Not claimed yet:
SAPIENT conformance, product support, or inferred schema compatibility.

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
  space, SemConnect egress, SemSource media handling, or an external system integration.
