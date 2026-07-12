# semlink-multi-companion-cop-demo Specification

## ADDED Requirements

### Requirement: SemOps Owns Fleet Aggregation Across SemLink Companions

SemOps SHALL treat the MVP SemLink deployment model as one vehicle-local companion boundary per vehicle while owning
the fleet-level aggregation, mesh/multiplex handling, and COP target derivation across companion outputs.

#### Scenario: Fleet proof uses one companion per vehicle

- **WHEN** SemOps consumes SemLink evidence for a multi-vehicle fleet proof
- **THEN** the default product claim is `n` SemLink companion boundaries for `n` vehicle-local companions
- **AND** each live vehicle claim requires distinct companion provenance plus distinct MAVLink target identity or a
  SemOps-reviewed source mapping
- **AND** SemOps does not require one SemLink runtime to multiplex multiple vehicles for the MVP proof

#### Scenario: Multi-vehicle companion evidence is labelled as aggregator evidence

- **WHEN** a SemLink report or artifact contains a companion node with `vehicle_count > 1`
- **THEN** SemOps preserves the reported vehicle count and source provenance
- **AND** SemOps labels the evidence as aggregator, stress, or non-default companion evidence unless the artifact
  carries an accepted multi-vehicle SemLink runtime profile
- **AND** deterministic multi-vehicle summaries alone do not satisfy a claim of multiple live ArduPilot vehicles

#### Scenario: SemOps multiplexes companion outputs without losing provenance

- **WHEN** SemOps receives multiple SemLink artifacts, streams, or gateway-demultiplexed companion outputs
- **THEN** it derives COP targets from companion provenance, MAVLink target system/component identifiers, configured
  source mapping, and born-target graph state
- **AND** equal MAVLink system IDs from different companion boundaries remain distinguishable until SemOps has an
  explicit deduplication or association decision
- **AND** raw MAVLink frames remain outside SemLink mesh evidence and outside CS API/SemConnect runtime dependency
  requirements for the hot path
