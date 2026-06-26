# Product E2E Anti-Cheat Review

Date: 2026-06-26

Scope: stack smoke, scenario-runner, command-control, and standards-bridge evidence boundaries after SemStreams
`v1.0.0-beta.115` ownership-token diagnostics.

Decision: pause new product e2e, command-control, and CS API claims until the evidence path enters SemOps through
hosted feed/component boundaries. Direct graph writes remain useful contract tests, but they are not product e2e.

## Findings

1. Direct graph writes are contract evidence, not product evidence.

   They prove born-first projection contracts, owner-token handling, indexing profiles, and graph readback. They do
   not prove transport ingress, SemStreams component lifecycle, payload registry decoding, flowgraph wiring,
   backpressure, runtime health, Prometheus telemetry, or browser-visible source flow.

2. Scenario-runner direct graph writes create owner-incarnation ambiguity in a full stack.

   A hosted SemOps runtime and a hosted scenario runner must not bind the same feed owner in the same product e2e
   stack. If two processes claim `semops.feed.mavlink` or another feed owner, owner-token mismatch warnings are
   evidence that the test is bypassing the intended component path.

3. Raw NATS subjects are not an escape hatch.

   NATS is the SemStreams substrate. Product e2e can use NATS only through declared SemStreams ports, registered
   payloads, component lifecycle, and graph request ports. A bespoke test helper that publishes graph mutations or
   synthetic component outputs directly to subjects is a contract shortcut, not an end-to-end test.

4. Direct live graph smokes still matter.

   They stay in the ladder as focused contract tests for SemStreams compatibility, ADR-055/056 born-first behavior,
   ADR-060 classified errors, foreign-edge ownership, and indexing-profile pressure. They must be named as
   contract-only when cited.

## Rules

- Product e2e evidence MUST enter through a supported external feed boundary or through a SemStreams input component
  that emits registered native/raw payloads.
- Product e2e evidence MUST exercise hosted input/processor components, flowgraph ports, payload registry decoding,
  graph request ports, owner tokens, component health, `DataFlow()`, and Prometheus telemetry when those surfaces
  exist for the feed.
- Full-stack smokes MUST NOT seed target COP state by calling graph mutation subjects directly, by using a bespoke
  graph writer, or by publishing decoded/projected payloads around the hosted component graph.
- A direct graph smoke MAY satisfy a projection-contract, SemStreams compatibility, or restart-reconciliation gate.
  It MUST NOT satisfy product e2e, operator UI, live command-control, CS API, provider integration, simulator-fidelity,
  or standards-conformance gates.
- A product e2e stack MUST have one live writer incarnation per governed feed owner for the path under test.
  Owner-token mismatch warnings, stale owner leases, or observe-only owner mismatch deltas fail product e2e evidence.
- Scenario-runner product mode SHOULD become a feed producer/orchestrator. Any retained direct graph projection mode
  is contract/replay infrastructure only.

## Follow-Up

- Refactor the full-stack scenario path so MAVLink, TAK/CoT, CAP, ADS-B, SAPIENT, weather, and future feeds enter
  through their hosted input/component boundaries before their state is used for product e2e claims.
- Keep CAP direct lifecycle projection as contract evidence until the hosted CAP poller/replay input owns the product
  e2e path for CAP.
- Resume the MAVLink `command-live-sim` gate only after PX4 telemetry and COMMAND_ACK evidence reach the COP snapshot
  through the hosted MAVLink input/decoder/projector chain without scenario-runner direct graph seeding.
- Keep CS API bidirectional e2e blocked until desired-state ingress, native feed execution, async status readback, and
  command-priority policy are tested through product boundaries rather than direct graph fixtures.

## Resolution Update

The first follow-up slice is complete for default stack evidence:

- `cmd/semops-scenario-runner` now defaults to product mode, emits MAVLink and TAK/CoT over hosted UDP feed
  boundaries, and does not create a SemStreams client, bind owners, or write graph mutations.
- The previous direct graph path remains available as `SEMOPS_SCENARIO_MODE=contract` and is contract/replay evidence
  only.
- Default CAP product visibility now comes from the hosted CAP HTTP poller reading the local fixture provider
  `/cap/alert` endpoint.
- The one-command stack smoke checks SemStreams owner-token mismatch metrics before direct graph contract smokes run.

Remaining blocked claims are command-control, CS API bidirectional interop, simulator-fidelity, live provider
integration, standards conformance, and operator scenario-control behavior.
