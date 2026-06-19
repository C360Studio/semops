# UI/API/Caddy Spine Adversarial Review

Date: 2026-06-19

## Decision

Accept the first UI/API/Caddy spine as a Phase 1 implementation slice, not as completion of the graph-backed COP UI.

## Evidence Checked

- `internal/api/cop` exposes `GET /api/cop/snapshot` and `/healthz` from a fixture provider.
- `ui` renders a Svelte 5 first screen with track, asset, hazard, alert, feed, freshness, confidence, and provenance
  state.
- `compose.cop.yml` adds `semops-ui` and `caddy`; Caddy routes `/api/*` and `/healthz` to SemOps API on the same
  origin as the UI.
- `scripts/cop-stack-smoke.sh` now polls direct SemOps API and Caddy-routed UI/API paths before the MAVLink graph
  smoke.

## Objections

- The snapshot provider is fixture-backed. It proves browser/API shape, not live graph query correctness.
- The map surface is a tactical placeholder. It does not yet prove MapLibre/deck.gl behavior, high-rate overlays,
  picking, trails, footprints, or temporal scrubbing.
- The UI shows provenance owner/source fields, but not graph revision, ownership claim status, or readback evidence.
- The Compose stack still lacks scenario runner, TAK/CoT, CAP, and second-feed graph-state smoke coverage.
- Caddy hides CORS drift by design; direct API smoke must remain so API failures are diagnosable outside the browser
  ingress.

## Accepted Risks

- Keep fixture fallback in the UI so frontend work can continue while graph-backed snapshot queries are built.
- Keep the direct API port exposed in local Compose for diagnostics.
- Defer topology, orchestration, tier controls, and dynamic ontology-generated UI until a later UX review proves
  operator value.

## Follow-Up Tasks

- Replace fixture snapshots with live graph-backed COP snapshot reads.
- Add bounded delta stream contract after the snapshot shape stabilizes.
- Replace the placeholder map surface with MapLibre/deck.gl layers.
- Add accessibility and e2e checks for entity selection, source state, and provenance inspection.
- Expand stack smoke to at least two feed types before Phase 1 signoff.
