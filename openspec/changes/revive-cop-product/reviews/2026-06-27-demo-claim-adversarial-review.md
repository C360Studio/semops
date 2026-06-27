# Demo Claim Adversarial Review

Date: 2026-06-27
Scope: SAPIENT, KLV, semantic, and standards-conformance language before demo promotion

## Decision

Accept the current SemOps demo claim set only as qualified engineering evidence. SemOps may demo source-partitioned
COP readback, opt-in component flows, deterministic semantic explanations, and the SemConnect read-side CS API bridge
when the claim names the exact evidence path and says what is not proven.

SemOps must not use unqualified SAPIENT support, KLV/STANAG support, semantic-service support, product e2e,
command/control, live-provider, or standards-conformance language from the current evidence.

This closes task `8.6` as an adversarial claim gate. It does not close future harness, conformance, service-mode,
operator-authority, or UI-promotion gates.

## Allowed Demo Language

- SAPIENT: "developer preflight for representative BSI Flex 335 v2 JSON/protobuf payloads" and "opt-in
  absolute-location detection projection through SemStreams components."
- KLV: "deterministic MISB ST 0601 subset evidence for sensor/frame-center/footprint geometry" and "opt-in local
  media fixture flow with COP readback and provenance."
- Semantic: "deterministic read-only COP explanation artifacts with task, source evidence, output, and trajectory
  references."
- CS API bridge: "read-side SemConnect fixture execution for Systems, Deployments, Datastreams, Observations, and
  System Events."
- Product e2e: "hosted feed/component-boundary evidence" only when the path under test enters through the supported
  input or SemStreams component boundary and exposes runtime/flow telemetry.

## Blocked Demo Language

- "SAPIENT compliant", "SAPIENT product support", "passed the Dstl harness", "hosted SAPIENT service", or implied
  SAPIENT tasking, lifecycle, UTM, range/bearing, association, or Apex middleware compatibility.
- "STANAG 4609 compliant", "STANAG 4609 certified", "full MISB ST 0601 support", "production KLV", "live video
  service", "streaming-binary support", or redistributable public KLV sample support.
- "Semantic service", "LLM agent", "civilian advisory translator", "semantic UI", "semantic graph projection", or
  autonomous recommendation authority from the deterministic explanation core.
- "OGC CS API conformance for SemOps", "CS API write-side interop", "tasking bridge", or command/status
  reconciliation from the read-side SemConnect fixture path.
- Any standards-conformance claim based on direct graph smokes, synthetic fixtures, product screenshots, or internal
  parser tests without an accepted external suite, harness, interoperability event, partner/lab result, or explicit
  non-conformance demo decision.

## Red-Team Findings

1. Fixture success can become fake standards credibility.

   Synthetic fixtures and deterministic generated media are excellent regression gates. They do not prove formal
   conformance, provider compatibility, operational reliability, or full protocol coverage.

2. Component promotion can be mistaken for product service support.

   SemStreams input/processor components prove lifecycle, flowgraph, payload, port, health, and telemetry shape. A
   component path still needs service-mode behavior, auth/session/federation, operator semantics, failure handling,
   and long-running evidence before product-support language is safe.

3. Semantic prose can look authoritative.

   Explanation text is helpful only when it is inseparable from task, evidence, trajectory, algorithm, and claim
   posture. Any UI or API that hides those fields would reintroduce unsupported authority.

4. Read-side standards bridges are not tasking bridges.

   SemConnect read-side fixture success proves egress shape and current-state publication. It does not prove CS API
   ingress, asynchronous command status, native feed execution, priority, TTL, local override, or conflict semantics.

5. Product e2e must enter through product boundaries.

   Direct graph writes, raw NATS shortcuts, and bespoke decoded payload publication remain contract evidence. Demo
   claims need hosted feed/component boundaries, owner-token discipline, runtime flow evidence, and Caddy/API/UI
   readback for the exact path under test.

## Stronger-Claim Unlocks

- SAPIENT compliance: Dstl harness pass or accepted authority result, role-scoped corpus, operating environment,
  limitations, and claim-language review.
- SAPIENT product service: hosted service design, sessions/auth, lifecycle/tasking policy, Apex/interoperability
  decision, telemetry, and failure-mode evidence.
- KLV/STANAG conformance: formal validator, accepted interoperability/lab evidence, full claim scope, public
  sample/license posture, and conformance review.
- Streaming media/KLV product support: sustained ingress, memory/backpressure bounds, storage materialization,
  media-service policy, and operator-visible failure evidence.
- Semantic service/UI: hosted execution design, prompt/trajectory storage, cost/failure controls, graph/API/UI
  readback, and authority-language review.
- CS API write-side/tasking: desired-state ingress, native command execution, async status reconciliation,
  TTL/priority/authority policy, and conflict semantics.
- Product e2e: feed/component-boundary ingress, registered payloads, graph request ports, owner-token checks, runtime
  metrics, and API/UI readback.
- Standards conformance: official conformance suite, accepted interoperability event, partner/lab result, or explicit
  non-conformance demo decision.

## Resolution

Allowed language may be used in demo notes and internal planning when the exact path is named. Blocked language stays
out of release notes, sponsor decks, PR descriptions, issue titles, and demo narration until the matching unlock gate
is complete.
