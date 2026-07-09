# SAPIENT Projection Contract Review

Date: 2026-06-23
Scope: `COP-007` SAPIENT absolute-location detection projection/readback gate

## Decision

Accept a narrow SAPIENT source-owner graph contract and pure projector for absolute-location detection reports only.
This approves `OwnerSAPIENT`, `semops.cop.track.sapient-detection-current-state`, source-partitioned `track` entities,
`signal` indexing, provenance, confidence, and COP API prefix-discovery readback.

This does not approve SAPIENT product support, SAPIENT conformance, local Dstl harness success, hosted SAPIENT graph
production, tasking, association, UTM conversion, range/bearing conversion, Apex service behavior, or a
SemOps-hosted SAPIENT service.

## Red-Team Findings

1. Public-looking examples can be coordinate traps.

   The known sample values can look like latitude/longitude while declaring `LOCATION_COORDINATE_SYSTEM_UTM_M`.
   Treating those as WKT points would create plausible but false map evidence. The first projector therefore accepts
   only `LOCATION_COORDINATE_SYSTEM_LAT_LNG_DEG_M` with WGS84 datum and rejects UTM until a deliberate conversion and
   datum policy exists.

2. Range/bearing is not a map point without sensor pose.

   A range/bearing detection needs source sensor position, reference frame, bearing unit, altitude behavior, and
   uncertainty before it can become a global coordinate. The first projector rejects range/bearing instead of silently
   projecting a guessed point.

3. `OwnerSAPIENT` is not a hosted feed claim.

   The owner token is acceptable for a reviewed source-partitioned graph contract, but the hosted runtime still runs
   SAPIENT as preflight HTTP input -> decoder only. Do not add SAPIENT to runtime owner registration, graph request
   ports, or product copy until the graph-producing component/writer boundary is reviewed.

4. Detection identity is not cross-source identity.

   SAPIENT object IDs are native detection identity, not proof that an ADS-B aircraft, MAVLink vehicle, TAK marker, or
   fused track is the same thing. The first SAPIENT contract declares no foreign edges. Association belongs to fusion
   or evidence contracts.

5. Harness evidence is still the compliance line.

   Parser, descriptor, projector, and COP readback tests are engineering evidence. They are not Dstl harness evidence
   and must not be presented as SAPIENT compliance.

## Evidence Accepted

- `pkg/cop` validates a source-partitioned SAPIENT track contract under `semops.feed.sapient`.
- `internal/projectors/sapient` plans create/update mutations for absolute-location detection reports and rejects UTM
  and range/bearing inputs.
- `internal/api/cop` maps prefix-discovered SAPIENT track state into COP snapshot tracks and `feed.sapient` health.

## Follow-Ups

- Promote runtime graph-producing SAPIENT components only through a separate runtime review; the accepted follow-on is
  recorded in `2026-06-23-sapient-runtime-graph-promotion-review.md`.
- Run or qualify the Dstl BSI Flex 335 v2 Test Harness before any compliance language.
- Decide whether generated Go bindings are needed after descriptor-based binary preflight proves insufficient.
- Design SAPIENT tasking, alert acknowledgements, and detection lifecycle as separate `control` contracts before
  command/control work begins.
