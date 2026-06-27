## ADDED Requirements

### Requirement: SemOps owns the complete COP product

SemOps SHALL be the product repo for the complete data-fusion common operating picture.

#### Scenario: Product boundary is explicit

- **WHEN** architecture, tickets, specs, or implementation decide where complete COP behavior belongs
- **THEN** SemOps owns the operator COP, canonical COP model, feed set, UI, fusion behavior, scenario runner, and
  product vocabulary

#### Scenario: SemLink remains a basic demo

- **WHEN** SemOps uses code, patterns, or UI ideas from SemLink
- **THEN** the work treats SemLink as prior art or a reusable pattern source, not as the owning product surface

### Requirement: SemStreams remains substrate-owned

SemOps MUST consume SemStreams substrate contracts rather than reimplementing framework-owned behavior locally.

#### Scenario: Graph writes use SemStreams contracts

- **WHEN** a SemOps component writes current state, events, or relationship evidence into the graph
- **THEN** it uses current SemStreams projection, ownership, graph mutation, and indexing-profile contracts

#### Scenario: Product runtime uses framework lifecycle patterns

- **WHEN** SemOps promotes parser, projector, replay, transport, or scenario behavior into a hosted runtime component
- **THEN** SemOps uses SemStreams component lifecycle, flowgraph, port, payload-registry, config-schema, health, and
  flow-metric interfaces where they apply
- **AND** transport listeners are input components whose stream output ports can be tapped by processor components
- **AND** parser, decoder, projector, and fusion behavior are processor components wired through declared ports rather
  than hidden NATS subject dependencies
- **AND** SemOps does not recreate a parallel component registry, lifecycle manager, flowgraph, port model, payload
  registry, or config schema locally

#### Scenario: Reusable framework pressure is routed upstream

- **WHEN** a SemOps feature needs reusable deployment metadata, inference evidence, provenance, or query helpers
- **THEN** SemOps records the product need and routes the reusable primitive to SemStreams for review

#### Scenario: Runtime utilities come from SemStreams first

- **WHEN** SemOps needs NATS access, request/reply, KV access, retry behavior, error classification, in-memory caches,
  or bounded concurrent buffers
- **THEN** SemOps evaluates SemStreams utility packages such as `natsclient`, `pkg/errs`, `pkg/cache`, and
  `pkg/buffer` before adding SemOps-local helper packages
- **AND** SemOps records concrete upstream issues when those utilities are missing feed-runtime capabilities needed by
  the COP product

### Requirement: SemConnect owns standards bridge evidence

SemOps SHALL use SemConnect for OGC Connected Systems API bridge behavior and conformance claims unless SemOps is
explicitly rechartered to own that gateway product.

#### Scenario: Governed COP state is published as CS API

- **WHEN** the COP publishes Systems, Datastreams, Observations, SystemEvents, or Commands to a standards consumer
- **THEN** SemOps sends a curated projection to SemConnect rather than making raw native feed data pass through CS API
  first

#### Scenario: CS API input enters through the same governed path

- **WHEN** a source system already publishes OGC Connected Systems API resources
- **THEN** SemOps may consume them through a CS API ingress adapter or bridge
- **AND** the adapter maps that input into SemOps governed COP state with the same ownership, provenance, freshness,
  indexing, and command-authority rules as native feed adapters

#### Scenario: Standards tasking does not bypass command authority

- **WHEN** CS API ControlStream, Command, or command-status input reaches SemOps
- **THEN** SemOps validates the request, returns a bounded standards-facing accept/reject response, and records
  governed command intent or desired state before any native actuation is attempted
- **AND** SemOps routes the intent through product command authority, native safety checks, and audit/replay evidence
  before any feed-owned state changes
- **AND** command intent records carry source authority, priority, TTL/deadline, idempotency key, correlation ID,
  target entity, local override policy, and cancellation/supersession semantics
- **AND** the first CS API write-side ingress layer may normalize Command and ControlStream input into command intent
  only, without granting native execution or upstream command-status publication authority
- **AND** native drivers reconcile actual execution asynchronously and publish command acknowledgements, progress,
  timeout, rejection, partial execution, or failure as graph-backed status evidence
- **AND** stale, duplicate, conflicting, or superseded upstream commands cannot become native actions after reconnect
  or replay unless the command authority policy explicitly renews them
- **AND** SemConnect or any CS API bridge remains a standards interface, not the authority that decides whether a
  vehicle, sensor, alerting, or collaboration command is safe to issue

#### Scenario: Conformance is not overclaimed

- **WHEN** a demo uses SAPIENT, OGC, STANAG 4609, ASTERIX, or another standard
- **THEN** docs and UI distinguish demo-grade subset support from tested conformance
