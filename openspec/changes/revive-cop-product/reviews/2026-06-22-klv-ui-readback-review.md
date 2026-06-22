# KLV UI Readback Review

Date: 2026-06-22

## Decision

Accept the first KLV product-visible readback slice.

`GET /api/cop/snapshot` now maps governed KLV `sensor_footprint` graph entities into a bounded COP view model, and the
Svelte COP renders selectable sensor points, frame-center points, and sensor-to-frame-center rays. The selected-entity
inspector shows KLV evidence, media reference, packet reference, decoded-field inventory, warnings, claim posture, and
provenance.

This is still sensor/frame-center evidence only. It is not footprint polygon extraction, video service support,
streaming-binary support, or STANAG 4609 conformance.

## Findings

### High: The layer must remain graph-readback evidence

The UI does not read raw KLV bytes, a local MPEG-TS file, or the public sample directly. It renders from the COP API
view model, which is populated from SemStreams prefix-discovered graph state.

Resolution: keep KLV UI data in `sensor_footprints[]` on the snapshot contract and keep raw media handling behind
SemStreams component ports and graph projection.

### High: The ray is not a footprint polygon

The implemented geometry proves sensor position and frame center. It intentionally avoids polygon styling and copy so
operators do not confuse a ray with a sensor footprint area.

Resolution: task `8.3` remains open for footprint polygon extraction after parser tags and derived-footprint policy are
reviewed.

### Medium: Decoded-field inventory is currently derived

The API derives decoded-field inventory from present governed predicates. That is good enough for the current proof,
but future public-sample and deterministic-fixture posture would be cleaner with explicit graph predicates for source
hash, fixture class, and parser warnings.

Resolution: keep the current derived inventory, but treat richer evidence vocabulary as a follow-up before stronger KLV
support language.

### Medium: KLV runtime is not hosted by default

The UI readback can display KLV graph state and fixture-backed API data, but the KLV media-ref -> demux -> decode ->
project flow is not wired into the default hosted stack.

Resolution: do not present this as live media ingress. Runtime composition remains a separate opt-in stack gate.

Update: the opt-in hosted runtime gate is now wired for configured local media references. KLV remains disabled by
default, and live media ingress, storage policy, and stronger video/STANAG claims remain separate gates.

## Verification

- `go test ./internal/api/cop`
- `npm --prefix ui run check`
- `npm --prefix ui run test`
- `npm --prefix ui run test:e2e`

## Follow-Ups

- Add explicit KLV evidence predicates for source hash/provenance class and parser warnings if the product needs
  stronger fixture/posture wording.
- Keep hosted KLV runtime disabled by default until live media ingress, materialization policy, and product claims are
  reviewed.
- Keep video player, thumbnails, 3D frustum, and footprint polygon work under separate reviews.
