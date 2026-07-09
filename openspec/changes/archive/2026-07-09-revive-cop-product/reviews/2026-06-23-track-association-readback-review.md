# Track Association Readback Review

Date: 2026-06-23

Scope: fusion-owned association projection plans, COP API readback, fixture/UI evidence, and browser smoke

## Decision

Accept this slice as read-only fusion association evidence. SemOps now has the path needed to show ambiguous
cross-source association in the COP without letting feed adapters claim identity authority or mutate source tracks.

## Findings

1. The projection path stays born-first and owner-scoped.
   `internal/projectors/fusion` creates a fusion-owned association entity with `control` indexing and uses
   update-only mutation plans after birth. Strict primary/candidate track edges are create-only so updates do not
   repeatedly assert relationship topology.

2. COP readback is deliberately evidence-shaped.
   `GET /api/cop/snapshot` discovers fusion association entities by prefix, maps confidence, algorithm, metrics,
   source refs, and posture, and exposes `feed.fusion` health. It does not collapse the source tracks into one
   aircraft object.

3. The UI shows the operator the risk instead of hiding it.
   The association rail and inspector expose primary track, candidate track, algorithm, distance/time evidence,
   provenance, and the explicit no-merge/no-identity-authority posture. Association records have selection affordance
   but no map geometry or merge control.

## Boundaries

- This is not a hosted SemStreams fusion processor yet.
- This is not identity resolution, track merge, split, override, or operator arbitration.
- This is not evidence that ADS-B, MAVLink, TAK, or SAPIENT adapters may emit association edges.
- Runtime backpressure, replay-window behavior, bounded subscription shape, and Prometheus health for a hosted
  association component remain open.

## Verification

- `go test ./internal/api/cop ./internal/projectors/fusion ./internal/fusion/association ./pkg/cop ./internal/copownership`
- `npm run check`
- `npm run test`
- `npm run test:e2e`

## Follow-Up

Promote track association into a hosted SemStreams component flow only after the bounded input subscriptions,
born-state reconciliation, runtime telemetry, and adversarial operator-review requirements are explicit.
