# semlink-semops-companion-contract Specification

## ADDED Requirements

### Requirement: Companion Identity Is Vehicle-Local For MVP

SemOps SHALL interpret `companion_node_id` in the v0 SemLink readback contract as the authenticated vehicle-local
companion boundary for one ArduPilot vehicle unless a later reviewed profile explicitly declares an aggregator.

#### Scenario: Readback target is derived by SemOps

- **WHEN** SemOps accepts a SemLink readback request
- **THEN** it treats `companion_node_id` as companion provenance, not distributed mesh causality metadata
- **AND** it derives the canonical COP target from companion provenance, MAVLink target system/component IDs,
  configured source mapping, and born-target graph state
- **AND** SemLink is not required to send `target_asset_id` or prove the canonical COP target

#### Scenario: Gateway multiplexing remains a SemOps boundary concern

- **WHEN** a deployment gateway, sidecar, or future SemOps ingress path multiplexes multiple SemLink companion outputs
- **THEN** the gateway or SemOps ingress must preserve the authenticated companion boundary for each request
- **AND** SemOps rejects or quarantines requests whose companion provenance cannot be separated before target
  derivation and graph writes
- **AND** this multiplexing does not require SemLink to expose CS API or to host multiple vehicle sources in one
  runtime for the MVP hot path
