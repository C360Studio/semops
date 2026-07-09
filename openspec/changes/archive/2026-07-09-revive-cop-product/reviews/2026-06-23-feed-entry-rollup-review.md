# Feed Entry Rollup Adversarial Review

Date: 2026-06-23

## Decision

Accept the current feed-entry review set as sufficient for the Phase 1 structural stack.

This is a retrospective corrective review. Several feeds already entered the structural stack before this top-level
task was closed, but each accepted entry now has a feed-specific evidence record, claim boundary, and executable or
documented gate. The acceptance is scoped to bounded MVP behavior, not product support, standards conformance, live
service reliability, or command/control readiness.

## Accepted Structural Entries

- MAVLink: accept codec, bounded raw lane, generated/replay graph smoke, hosted UDP path, and skipped external SITL
  harness. Do not accept PX4/MAVSDK fidelity proof, live command/control, or durable checkpointing.
- TAK/CoT: accept native parser, UDP/TCP input components, projection, graph smoke, and hosted listener readback. Do
  not accept TAK Server, federation, full CoT coverage, or verified conformance.
- CAP: accept parser, consumer-rule preflight, opt-in XSD/sample smoke, derived lifecycle replay, and opt-in HTTP
  component flow. Do not accept CAP consumer conformance, default live NWS/IPAWS service, or captured lifecycle corpus.
- ADS-B: accept OpenSky-shaped parser/replay, opt-in HTTP component graph path, and portable derived fixture. Do not
  accept live OpenSky reliability, credentials, receiver support, ASTERIX, or statistical association.
- SAPIENT: accept JSON/binary descriptor preflight, raw replay, and opt-in absolute-location detection graph path. Do
  not accept compliance, hosted SAPIENT product service, tasking, association, UTM, or range/bearing support.
- KLV: accept deterministic MISB subset, synthetic packet fixture, opt-in local media chain, and sensor/frame-center
  projection. Do not accept STANAG conformance, public media redistribution, live media service, or footprint polygons.
- Weather: accept Open-Meteo-shaped point, OGC EDR-shaped fixtures, point graph projection, and opt-in hosted point
  runtime. Do not accept live provider, visual weather tiles, cache/stale policy, route safety, or OGC conformance.
- DJI: accept synthetic telemetry/media-reference parser and component preflight. Do not accept live DJI bridge, media
  relay, command authority path, or DJI compatibility/certification.

## Red-Team Findings

1. Structural entry can be mistaken for product support.

   Every feed must keep its default runtime posture and claim language visible. Opt-in component paths are evidence,
   not default service promises.

2. Synthetic fixtures can look like captured provider evidence.

   The fixture manifest and per-feed reviews now distinguish ignored live captures, cleared committed fixtures, and
   derived story fixtures. Demo copy must preserve that distinction.

3. High-rate feeds can stress SemStreams indexing and runtime telemetry.

   The accepted structural entries rely on `indexing_profile` boundaries, source-partitioned ownership, component
   health, flow metrics, and explicit opt-in runtime flags. Mixed-feed cardinality and query helper asks remain
   upstream candidates after more evidence.

4. Command/control is not implied by readback.

   MAVLink commands, TAK tasking, SAPIENT tasks, DJI command authority, and CS API tasking must stay behind authority,
   TTL, priority, local override, and asynchronous status reviews.

5. Standards-facing claims still need separate evidence.

   CAP, SAPIENT, KLV/STANAG, OGC EDR, CS API, ADS-B/ASTERIX, TAK/CoT, and DJI compatibility claims require official
   schemas, conformance suites, accepted interoperability tests, or documented partner/lab evidence.

## Follow-Up Gates

- Close `8.6` only through the later demo-claim adversarial review before SAPIENT, KLV, semantic, or
  standards-conformance claims are demoed. [done: `2026-06-27-demo-claim-adversarial-review.md`]
- Keep `9.7` open for adversarial ownership review before filing broad upstream SemStreams asks.
- Do not close MAVLink PX4/SITL, SAPIENT Dstl harness, KLV/SemSource streaming-binary, CS API bridge, or association
  tasks from this rollup.
