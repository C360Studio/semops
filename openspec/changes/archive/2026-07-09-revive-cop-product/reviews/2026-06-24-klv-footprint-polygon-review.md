# KLV Footprint Polygon Review

Date: 2026-06-24
Scope: deterministic MISB ST 0601 offset-corner footprint extraction, graph projection, COP API readback, and UI map
layer rendering.

## Verdict

Accept KLV footprint polygons as a narrow deterministic MISB ST 0601 offset-corner subset.

This closes task 8.3 for the current demo slice, but it does not create STANAG 4609 conformance, live media service,
streaming-binary, 3D frustum, or broad footprint-policy claims.

## Findings

1. Polygon support must require complete corner evidence.
   SemOps should compute a footprint polygon only when all four MISB ST 0601 offset-corner latitude/longitude pairs
   and a frame center are present. Partial or invalid corner evidence should produce warning evidence, not a synthetic
   polygon.

2. The graph contract should own SemOps KLV geometry, not hazard authority.
   `cop.sensor_footprint.geometry` belongs to the KLV sensor-footprint contract under `semops.feed.klv`. It must not
   be confused with `cop.hazard.geometry`, fusion associations, or authoritative incident boundaries.

3. UI polygons need provenance posture.
   Rendering a polygon on the COP is useful eye candy with real evidence, but the inspector must keep media reference,
   packet reference, decoded field inventory, claim posture, and provenance visible so the shape is not mistaken for a
   certified sensor footprint or tasking boundary.

4. Deterministic fixture expansion is acceptable.
   The truth JSON and hex packet are SemOps-authored synthetic artifacts. They can support engineering language for the
   tested subset, but not public-sample redistribution, lab conformance, or production media reliability language.

## Boundaries

- No video player, keyframe strip, thumbnail generation, or MediaMTX/media relay support is implied.
- No STANAG 4609 conformance or official MISB conformance result is implied.
- No broad derived-footprint policy is implied for packets that lack all four offset-corner pairs.
- No hazard geometry, association, or command authority is implied.

## Follow-Ups

- Add public-sample polygon smoke only after redistribution/provenance review clears the sample for the intended use.
- Keep 3D frustum and video playback behind separate UI and media-stack reviews.
- Consider a geometry helper/vocabulary ask to SemStreams only after KLV, weather, and CS API workflows prove shared
  spatial helper pressure.
