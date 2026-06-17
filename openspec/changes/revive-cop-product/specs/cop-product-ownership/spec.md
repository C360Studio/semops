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

#### Scenario: Reusable framework pressure is routed upstream

- **WHEN** a SemOps feature needs reusable deployment metadata, inference evidence, provenance, or query helpers
- **THEN** SemOps records the product need and routes the reusable primitive to SemStreams for review

### Requirement: SemConnect owns standards egress

SemOps SHALL use SemConnect for OGC Connected Systems API egress and conformance claims.

#### Scenario: Governed COP state is published as CS API

- **WHEN** the COP publishes Systems, Datastreams, Observations, SystemEvents, or Commands to a standards consumer
- **THEN** SemOps sends a curated projection to SemConnect rather than making raw feed data pass through CS API

#### Scenario: Conformance is not overclaimed

- **WHEN** a demo uses SAPIENT, OGC, STANAG 4609, ASTERIX, or another standard
- **THEN** docs and UI distinguish demo-grade subset support from tested conformance
