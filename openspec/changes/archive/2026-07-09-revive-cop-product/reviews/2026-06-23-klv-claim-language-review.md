# KLV/STANAG Claim Language Review

Date: 2026-06-23
Scope: Demo and sponsor-facing language for KLV, MISB ST 0601, STANAG 4609, and streaming-binary claims

## Decision

SemOps may use narrow engineering-support language for the tested MISB ST 0601 subset only when it is tied to the
deterministic fixture, bounded component flow, governed graph projection, and COP API/UI readback evidence.

SemOps must not claim STANAG 4609 conformance, STANAG 4609 certification, full MISB ST 0601 coverage, production
video exploitation, live media service support, footprint polygon extraction, or general streaming-binary product
support from the current proof spike.

## Allowed Language

- "KLV-derived sensor and frame-center evidence for the tested MISB ST 0601 subset."
- "Deterministic MISB ST 0601 fixture support with truth-data assertions inside integer quantization tolerances."
- "Opt-in public-sample smoke for real-world MPEG-TS demux/parser plumbing when a local sample and provenance notes are
  supplied."
- "Governed SemStreams projection and COP readback for sensor point, frame-center point, ray, media reference, packet
  reference, decoded-field inventory, warnings, and claim posture."

## Blocked Language

- "STANAG 4609 compliant", "STANAG 4609 certified", or equivalent certification language.
- "Full MISB ST 0601 support" or "complete KLV parser".
- "Streaming-binary support" without separate sustained-ingress, memory, backpressure, and media-service evidence.
- "Video service", "video player", "thumbnail/keyframe exploitation", "3D frustum", or "footprint polygon" support
  from the current slice.
- "Redistributable public sample fixture" for FFmpeg-hosted MPEG-TS samples until a license review clears that path.

## Findings

1. Public-sample smoke is valuable but legally and mathematically weak.

   The local `Day Flight.mpg` smoke exercises a real-world-shaped MPEG-TS/KLV path, but the current evidence does not
   clear redistribution rights and does not provide SemOps-owned truth data for correctness assertions.

2. The deterministic fixture is the engineering-support core.

   The SemOps-owned truth JSON to generated packet to decoded frame path is the evidence that can support limited
   MISB ST 0601 subset claims. MPEG-TS wrapping proves container plumbing, not broad live-media support.

3. The UI proof must expose provenance, not just geometry.

   A sensor point, frame-center point, and ray are useful only when the inspector shows the media reference, packet
   reference, decoded field inventory, warnings, source/provenance posture, and component-flow evidence.

4. SemSource storage proof is not SemOps KLV proof.

   SemSource may prove binary-by-reference storage and governed metadata, but KLV/MISB/STANAG interpretation and
   product claims remain SemOps-owned.

## Required Before Stronger Claims

- Public media license/provenance review before vendoring or default CI use.
- A formal STANAG 4609 validator, accepted interoperability event, or lab path before conformance/certification
  language.
- Footprint polygon math and MISB tag policy review before footprint extraction language.
- Live media ingress, memory, backpressure, storage materialization, and operator alert evidence before
  streaming-binary or video-service product language.
