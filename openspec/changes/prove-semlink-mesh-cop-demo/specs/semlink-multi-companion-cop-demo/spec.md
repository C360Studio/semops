# semlink-multi-companion-cop-demo Specification

## ADDED Requirements

### Requirement: SemOps Mesh Demo Uses One-To-One SemLink Companion Evidence

SemOps SHALL provide a mesh demo proof that consumes SemLink-generated
one-vehicle-per-companion mesh evidence and renders it as curated COP/GCS glass
without expanding SemLink runtime or command-authority claims.

#### Scenario: One-to-one SemLink mesh artifact is accepted

- **WHEN** SemOps imports a SemLink-generated
  `semlink-companion-demo-artifact-v0` artifact whose embedded report kind is
  `simple-mesh-companion-demo`
- **THEN** the report contains at least three companion nodes
- **AND** each companion node reports `vehicle_count=1`
- **AND** `expected_summaries` equals the companion node count
- **AND** raw MAVLink exclusion posture is preserved from the artifact
- **AND** SemOps labels the evidence as SemLink-generated deterministic mesh
  evidence unless the artifact carries accepted live-source metadata

#### Scenario: Operator sees SemLink mesh posture in COP glass

- **WHEN** the SemOps COP displays the imported SemLink mesh artifact
- **THEN** the companion fleet view exposes node count, vehicle profile, node
  identities, one vehicle per companion, selected-summary catch-up,
  watermark/diff posture, assertion state, raw MAVLink exclusion, source
  fidelity, and no-transmit posture
- **AND** selecting or inspecting the fleet does not expose peer management,
  mesh topology editing, raw MAVLink forwarding, SemLink runtime controls, or
  companion command-rule controls

#### Scenario: Demo claim language is constrained

- **WHEN** SemOps prepares mesh demo evidence from deterministic SemLink
  artifacts
- **THEN** SemOps may claim GCS/COP display of a SemLink-produced
  one-companion-per-vehicle mesh artifact
- **AND** SemOps must not claim live BlueOS, Navigator, radio mesh, hardware,
  SITL, native MAVLink transmit, companion hardware transmit, mission upload,
  mode change, arm/disarm, offboard control, or CS API hot-path evidence
  without later evidence-gated specs
