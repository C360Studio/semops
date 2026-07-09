# SAPIENT Graph Stack Smoke Review

Date: 2026-06-23
Scope: opt-in one-command SAPIENT absolute-location detection graph smoke

## Decision

Accept an opt-in SAPIENT graph stack smoke for the reviewed absolute-location detection contract.

`SEMOPS_COP_SMOKE_SAPIENT_GRAPH_ENABLED=true bash scripts/cop-stack-smoke.sh` may switch the local SAPIENT fixture URL
to `/sapient/detections`, enable `SEMOPS_SAPIENT_GRAPH_ENABLED=true`, assert Caddy-routed COP snapshot readback for a
SAPIENT track, and require SAPIENT HTTP input, decoder, and projector flow in Prometheus and `/api/cop/runtime`.

The default stack smoke remains decoded-stream preflight against `/sapient/messages`, with no SAPIENT owner
registration and no graph projector requirement.

## Red-Team Findings

1. Default smoke must stay preflight-only.

   The task-ack smoke is useful because it proves decoded SAPIENT flow without pretending to produce track state. Graph
   smoke must require an explicit graph flag and a detection-producing endpoint.

2. Graph readback is not SAPIENT service support.

   A track visible in the COP proves SemStreams component flow, ownership registration, born-first graph writes, and
   COP prefix discovery for one fixture. It does not prove sessions, tasking, alert acknowledgements, Apex middleware
   behavior, or a SemOps-hosted SAPIENT service.

3. The decoded-stream smoke must not be hard-coded to task acknowledgements.

   In graph mode the decoded stream should contain `detectionReport`, not `taskAck`. The smoke now takes an expected
   content-kind environment variable so both modes remain explicit.

4. Runtime telemetry must reflect graph mode.

   SAPIENT should have two components in preflight mode and three in graph mode. The smoke asserts the projector only
   when graph mode is enabled, preventing both under- and over-claiming.

## Evidence Accepted

- `scripts/cop-stack-smoke.sh` adds `SEMOPS_COP_SMOKE_SAPIENT_GRAPH_ENABLED` and switches the fixture URL and expected
  decoded content only under that flag.
- `internal/smoke/cop` asserts SAPIENT graph track readback, projector metrics, and runtime component count only in
  graph mode.
- `internal/smoke/sapient` validates either `taskAck` or `detectionReport` decoded content based on the explicit
  smoke expectation.
- `SEMOPS_COP_SMOKE_SAPIENT_GRAPH_ENABLED=true bash scripts/cop-stack-smoke.sh` passed on 2026-06-23 with MAVLink,
  CoT, CAP, ADS-B, SAPIENT graph snapshot, component Prometheus metrics, runtime flow, and SAPIENT decoded preflight
  enabled. KLV and weather remained skipped because their stack flags were not enabled.
- SAPIENT graph snapshot readback accepts `feed.sapient` as `live` or `stale` because the deterministic Dstl-shaped
  detection fixture carries an old source timestamp. The assertion remains strict on SAPIENT source, non-zero
  position, `semops.feed.sapient` provenance, and non-empty source reference.

## Follow-Ups

- Keep SAPIENT tasking, alert acknowledgements, detection lifecycle, association, range/bearing conversion, UTM
  conversion, and harness compliance as separate gates.
