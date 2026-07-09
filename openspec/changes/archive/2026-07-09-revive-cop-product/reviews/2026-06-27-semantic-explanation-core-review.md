# Semantic Explanation Core Review

Date: 2026-06-27
Scope: deterministic semantic translation and anomaly explanation artifacts

## Decision

Accept the first semantic explanation slice as read-only derived evidence. It is enough to close the deterministic
semantic core gate, but not enough to claim a hosted semantic service, LLM workflow, graph projection, COP UI surface,
or standards-facing semantic conformance.

## Red-Team Findings

1. Explanation is output, not authority.

   The builder emits operator-readable text with algorithm identity and read-only claim posture. It does not mutate
   source tracks, merge identities, publish graph state, or drive command/control behavior.

2. No orphan prose.

   Every explanation carries a task prompt, task input reference, source evidence, output text, and semantic
   trajectory reference. If a later service cannot preserve those fields, it should not be promoted into the COP.

3. Association anomaly language must preserve ambiguity.

   Review-worthy fusion evidence is explained as candidate/anomalous evidence and keeps primary and candidate tracks
   as separate evidence refs. The explanation may help an operator inspect the situation, but it is not an identity
   decision.

4. Hosted semantic service remains a later gate.

   This package is graph-free, service-free, UI-free, and LLM-free. A later hosted service needs explicit component
   telemetry, cost/failure behavior, prompt/trajectory storage, operator semantics, and adversarial review before it
   becomes demo language.

## Accepted Evidence

- `internal/semantic` builds deterministic explanation sets from `internal/api/cop.Snapshot`.
- Track translations preserve source owner, source ref, observed time, confidence, alert evidence, task input, and
  semantic trajectory reference.
- Association anomaly explanations preserve fusion owner, source-track evidence refs, confidence, distance/time
  metrics, algorithm identity, claim posture, and source-track separation.
- Tests assert prompt/task, output, evidence, trajectory reference, algorithm, and read-only claim posture invariants.

## Follow-Ups

- Add hosted semantic-service execution only after prompt/trajectory storage and component telemetry are designed.
- Add COP API/UI readback only after the object contract is stable enough to avoid free-text authority ambiguity.
- Re-run adversarial review before any semantic, SAPIENT, KLV, or standards-conformance claim appears in demo copy.
