# Fusion Stack Smoke Review

Date: 2026-06-23

Scope: opt-in hosted fusion candidate production, association projection, runtime telemetry, and COP readback

## Findings

1. The stack smoke is a valid flow proof, not an identity-fusion proof.
   It seeds one generated MAVLink track and one close TAK/CoT operator track through their transport components,
   lets the fusion candidate producer discover source-owned graph state, and verifies a fusion-owned association
   entity through COP readback. It does not prove persistent identity, operator merge authority, or automatic
   association policy.

2. The smoke should remain opt-in until operator semantics are reviewed.
   Association evidence can be useful eye candy with teeth, but default automatic association risks teaching users
   that the COP has merged tracks or resolved identity. Keep `SEMOPS_COP_SMOKE_FUSION_ENABLED=true` as the proof gate
   and leave default stack fusion disabled until identity policy, confidence display, and operator affordances are
   reviewed.

3. The implementation stays inside the SemStreams component contract.
   The smoke uses hosted MAVLink and TAK/CoT inputs instead of publishing `semops.fusion.track_candidates` directly.
   Candidate production uses graph prefix-query request ports, registered payloads, bounded batching, and component
   telemetry; projection uses the fusion owner and born-first graph mutation request contracts.

## Accepted Posture

- Accept the stack-level flow proof for `mavlink` plus `tak` source-track association.
- Keep ADS-B/SAPIENT association as later scenario coverage once receiver timing and SAPIENT detection semantics are
  less fixture-specific.
- Keep default automatic association, identity fusion controls, and merge language closed until the next adversarial
  operator review.
