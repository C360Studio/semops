# DJI Component Preflight Review

Date: 2026-06-23

Scope: fixture-backed DJI input and decoder components for synthetic telemetry/media-reference evidence.

## Decision

Accept the first DJI component-flow slice as preflight evidence only.

The components are valuable because they force DJI work through SemStreams lifecycle, payload registry, ports, config
schema, health, and flow metrics before runtime wiring or graph projection. They do not prove live DJI integration,
SDK compatibility, Cloud API support, media relay behavior, replay service durability, or command/actuation safety.

## Adversarial Notes

### High: Component shape is not product support

`internal/components/dji` publishes raw and decoded synthetic fixture payloads. That is framework-compliance evidence,
not DJI support evidence.

Mitigation:

- Keep docs explicit that the current path is fixture-backed and synthetic.
- Block live-bridge wording until an ingress surface, legal fixture, replay strategy, and SDK/cloud boundary exist.

### High: No graph ownership exists yet

The decoder emits stream payloads only. There is no DJI graph projector, no owner token, and no ownership contract.

Mitigation:

- Do not add runtime owner claims for DJI until a graph projection contract defines source-partitioned asset/sensor
  state, media references, command authority, provenance, and indexing profile.
- Keep command authority represented as data only until a safety review approves an actuation boundary.

### Medium: File input is intentionally narrow

The file input proves fixture flow through SemStreams ports. It does not model cloud polling, controller file export,
SDK callbacks, live streams, or dock/session APIs.

Mitigation:

- Treat fixture file input as a replaceable preflight seam.
- Use the same raw/decoded payload boundary when live ingress is chosen.

## Verification

- `go test ./internal/components/dji`
- `go test ./internal/contracts`
- `go test ./...`

## Follow-Ups

- Choose DJI live/captured ingress surface only after representative fixture and licensing review.
- Add a replay store if captured data proves the fixture shape should persist beyond tests.
- Add a graph projector only after ownership, indexing, command-authority, and UI claim posture are reviewed.
