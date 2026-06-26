# Feed-Boundary Orchestration Checkpoint Review

Date: 2026-06-26

Scope: scenario-runner expansion after the product e2e anti-cheat review.

## Decision

Extend scenario orchestration as claim-scoped checkpoints before adding operator controls or new high-risk claims.

The scenario runner may coordinate deterministic feed playback, fixture-provider timing, and expected-state checks,
but each checkpoint must declare the claim it supports, the external feed boundary or SemStreams input component used,
the expected COP readback, expected component/runtime evidence, and graph owners allowed to write state.

## Objections

1. A generic scenario shell would look useful but could silently become a second product framework inside SemOps.
2. Start/reset/pause/resume controls can imply command authority or operator workflow maturity that the current stack
   does not yet prove.
3. A single successful HADR smoke could be overused as evidence for command-control, CS API, simulator-fidelity,
   provider, standards, or operator-control claims.
4. Direct graph or NATS payload injection remains valuable for contract tests, but it is too easy to confuse with
   product e2e when wrapped in scenario language.

## Evidence Checked

- Default product scenario status reports `ingress_mode=feed-boundary`, feed-boundary deliveries, zero graph
  mutations, and zero contract graph mutation attempts.
- The one-command stack smoke consumes Caddy-routed `/scenario/status`, validates feed-boundary scenario state, and
  checks owner-token mismatch metrics before direct contract smokes run.
- MAVLink and TAK/CoT product state enters through hosted UDP feed boundaries.
- CAP and ADS-B product visibility enters through hosted HTTP component paths.
- SAPIENT default stack evidence is decoded-stream preflight; graph projection remains opt-in and fixture-backed.
- The UI scenario panel is read-only evidence, not a scenario control surface.

## Accepted Risks

- The first checkpoint contract is documentation/spec evidence; manifest-backed checkpoint execution remains future
  implementation work.
- Stage/demo operators still need manual stack commands for start/reset while operator controls are gated.
- The base HADR smoke proves feed-boundary product e2e only for its current feed set and local fixture scope.

## Follow-Ups

- Add a manifest-backed scenario checkpoint format when replay start/reset/pause/resume or expected-state checkpoints
  become implementation work.
- Keep operator scenario controls behind adversarial UX review.
- Keep command-control checkpoints behind native transmitter or driver evidence, command ACK/status correlation, and
  post-command COP readback.
- Keep CS API checkpoints behind a real CS API ingress/egress boundary and governed desired/actual-state
  reconciliation.
- Keep provider, simulator-family, standards, and conformance claims tied to explicit provider endpoints, simulator
  families, or conformance harnesses.
