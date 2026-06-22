# KLV Projector Contract Review

Date: 2026-06-22

## Decision

Accept a narrow KLV graph projection contract for the first product slice: decoded MISB ST 0601 frames may project
source-partitioned `sensor_footprint` current state with `indexing_profile=signal`.

The contract covers sensor position, frame center, azimuth/elevation, media reference, packet reference, platform
designation, provenance, owner-token fencing, and born-first create/update behavior. It does not claim full footprint
polygon extraction, clip/evidence package lifecycle, hosted runtime wiring, public-sample support, or STANAG 4609
conformance.

## Guardrails

- Use `semops.feed.klv` owner tokens minted through the SemStreams registry/bind path.
- Keep KLV state source-partitioned under `c360.*.cop.klv.sensor_footprint.*`.
- Do not create foreign edges or cross-feed associations in the first KLV projector slice.
- Do not place raw packet bytes in graph triples.
- Keep footprint polygons as a later explicit gate.

## Evidence

- `pkg/cop.KLVSensorFootprintContract` declares replace-owned signal predicates for KLV sensor/frame-center state.
- `internal/projectors/klv` plans born-first create/update mutations and graph writer requests.
- `internal/components/klv.ProjectorComponent` consumes registered frame BaseMessages and writes projector plans through
  an injected writer.
- `go test ./pkg/cop ./internal/projectors/klv ./internal/components/klv` passes.

## Follow-Up

- Add footprint polygon extraction after the parser has the required MISB tags or a derived-footprint policy.
- Wire KLV projector runtime only after storage-reference materialization and feed enablement policy are decided.
- Decide how packet/decode trace events are represented before exposing high-rate KLV diagnostics.
