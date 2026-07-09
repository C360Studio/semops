# KLV UI Evidence Plan Review

Date: 2026-06-22

## Decision

Accept KLV sensor-footprint API/UI readback as the next product-visible slice.

The slice should make the binary pipeline tangible by rendering a selectable sensor point, frame-center point, and
sensor-to-frame-center ray from governed `sensor_footprint` graph state. It must also expose media reference, packet
reference, decoded field inventory, warning evidence, source provenance, and claim posture in the selected-entity
inspector.

Do not include footprint polygons, video playback, thumbnails, 3D frustums, streaming-binary language, or STANAG 4609
conformance language in this slice.

## Findings

### High: Video-first UI would hide the real proof

Showing a public MPEG-TS sample in the browser would be visually satisfying but would not prove SemStreams-governed
binary-derived state. The proof has to read back graph state through SemOps API and tie every visible geometry to
media and packet provenance.

Resolution: API readback comes before any media player. The map renders sensor point, frame center, and ray from the
COP view model, not from raw KLV bytes or a local media file.

### High: Footprint polygons are a separate claim

The current projector has sensor/frame-center evidence, not a validated footprint polygon policy. A polygon layer would
look like a finished sensor-footprint capability even if the parser lacks the required MISB fields or the math policy
has not been reviewed.

Resolution: keep polygon extraction under task `8.3` and use the first UI layer to show a ray from sensor position to
frame center.

### Medium: Public sample smoke must not become compliance language

The local `Day Flight.mpg` smoke proves that SemOps can exercise demux/decode against a real-world-shaped public
sample when a developer supplies a local file and provenance. It does not prove redistribution rights, deterministic
correctness, or STANAG 4609 conformance.

Resolution: the selected-entity inspector should show public-sample posture as smoke evidence only, while deterministic
fixtures carry engineering-support language only for the tested MISB ST 0601 subset.

### Medium: The inspector is part of the product proof

A map mark without provenance would be eye candy. For this slice, the provenance inspector is the credibility layer:
media reference, packet reference, frame time, observed time, decoded fields, warnings, source hash/provenance, and
runtime component evidence should all be visible from selection.

Resolution: Playwright should assert the map selection path and the inspector contents, including narrow viewport and
keyboard selection behavior.

## Verification Expectations

- `openspec validate revive-cop-product --strict`
- `go test ./internal/api/cop` after the API readback implementation exists
- `npm --prefix ui run test` after the UI layer implementation exists
- `npm --prefix ui run test:e2e` after the Playwright KLV fixture path exists

## Follow-Ups

- Add the COP API `sensorFootprints` view model for KLV graph readback.
- Add deck.gl sensor point, frame-center point, and ray layers with picking.
- Add selected-entity provenance copy that distinguishes public-sample smoke from deterministic engineering evidence.
- Revisit video player, thumbnails, 3D frustum, and footprint polygon work only after separate media/cache, UX, math,
  and standards-evidence reviews.
