# Fusion Default Posture Review

Date: 2026-06-23

Scope: automatic demo association posture after hosted candidate production, association projection, runtime telemetry,
and stack-level COP readback passed.

## Decision

Hold default automatic fusion association. Keep `SEMOPS_FUSION_ENABLED` and
`SEMOPS_FUSION_CANDIDATES_ENABLED` opt-in for now.

The stack smoke proves the flow. It does not yet prove that operators will read the result as candidate evidence rather
than identity resolution.

## Findings

1. The graph/runtime contract is acceptable for gated experiments.
   The hosted flow uses SemStreams lifecycle components, registered candidate payloads, bounded candidate production,
   graph prefix discovery, fusion owner tokens, born-first graph writes, Prometheus metrics, and COP readback. That is
   enough to keep testing association with real feeds.

2. The UI is evidence-shaped but not yet default-safe.
   The inspector shows distance, time delta, algorithm, provenance, and the no-merge/no-identity-authority posture.
   However, the row-level status can still read as `associated`, which is too easy to interpret as a resolved identity
   when scanning a busy COP.

3. The confidence policy is mission-dependent.
   Default thresholds are useful for a synthetic demo but not for HADR, ADS-B proximity, SAPIENT detections, or later
   DJI/KLV imagery without mission profile context. Turning it on by default would make a product promise before
   SemOps has threshold, ambiguity, stale-window, and source-priority controls.

4. There are no operator merge/split/acknowledge controls yet.
   That is correct for MVP, but it means default association must remain read-only and clearly tentative. A future
   default-on posture needs an operator path for hiding, acknowledging, or challenging bad evidence without mutating
   source-owned tracks.

## Boundaries

- This is not a rejection of the fusion component flow.
- This does not require identity merge controls for MVP.
- This does not block opt-in stack smoke, fixture UI, or controlled demo runs.
- This does not authorize adapters to emit cross-source association edges.

## Follow-Up

- Rename or normalize operator-facing non-final statuses toward candidate/possible association language.
- Add mission-profile configuration for distance, time-window, confidence, ambiguity margin, source priority, and stale
  windows before default enablement.
- Add UI affordances for evidence inspection, filtering, and operator challenge/acknowledgement before identity-fusion
  claims.
- Re-run the stack smoke and browser e2e once the row-level language and default profile are ready.
