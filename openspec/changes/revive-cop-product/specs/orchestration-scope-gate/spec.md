## ADDED Requirements

### Requirement: COP value comes before orchestration UI

SemOps MUST NOT treat deployment orchestration, tier choreography, or topology panels as accepted product features
until they prove operator value for the COP.

#### Scenario: Structural COP does not require orchestration feature work

- **WHEN** Phase 1 is planned or implemented
- **THEN** map, source, provenance, alerts, feed health, and fusion correctness take priority over orchestration UI

#### Scenario: Orchestration proposal must name the operator job

- **WHEN** a feature proposes a topology, manifest, tier, or escalation control
- **THEN** the proposal identifies the operator decision it improves and the simpler alternative it replaces

### Requirement: Tier escalation starts as evidence, not a control surface

SemOps SHALL record statistical or semantic inference use as provenance and evidence before exposing it as an
operator-facing orchestration feature.

#### Scenario: Track association records inference evidence

- **WHEN** a statistical association service correlates ambiguous tracks
- **THEN** SemOps records source facts, association confidence, model or algorithm identity, and resulting evidence
  without requiring a tier-control panel
- **AND** association evidence SHALL be owned by the fusion owner, not by the source feed owners
- **AND** source tracks SHALL remain source-partitioned current state; association does not merge or mutate them
- **AND** the first implementation MAY be a pure scorer, graph projection plan, hosted processor, bounded candidate
  producer, and read-only API/UI evidence surface before automatic demo association is enabled by default
- **AND** any later merge, split, or identity-authority workflow requires an adversarial review before it becomes an
  operator control

#### Scenario: Semantic translation records trajectory

- **WHEN** a semantic service generates a civilian advisory or anomaly explanation
- **THEN** SemOps records the prompt/task, source evidence, output, and trajectory reference before deciding whether
  that transition deserves dedicated UI

### Requirement: Orchestration candidates are evaluated as risks

SemOps SHALL evaluate orchestration features for footguns before they enter the implementation backlog.

#### Scenario: Topology panel can be omitted

- **WHEN** service health, source provenance, and scenario state already answer the operator's question
- **THEN** SemOps omits or defers a topology panel instead of adding visual complexity

#### Scenario: Manifest does not become fake scheduling

- **WHEN** a manifest is only documenting local Compose services
- **THEN** SemOps treats it as deployment metadata, not as an orchestrator or scheduling framework

#### Scenario: Framework-shaped primitive needs upstream review

- **WHEN** a manifest, escalation event, or placement primitive looks reusable beyond SemOps
- **THEN** SemOps opens a SemStreams review before hardening product-local semantics
