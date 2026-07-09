# semlink-multi-companion-cop-demo Specification

## Purpose
Define SemOps' consumer proof for SemLink companion demo evidence: normalize SemLink single-node and simple-mesh
reports into curated COP/GCS glass, show at least three companion nodes, and keep SemLink mesh mechanics, BlueOS
hardware claims, raw MAVLink, and CS API runtime dependencies outside the MVP hot path.

## Requirements
### Requirement: SemOps Consumes SemLink Demo Evidence

SemOps SHALL consume SemLink single-node and simple-mesh demo evidence as downstream product input without requiring
SemLink, BlueOS, Navigator hardware, ArduPilot SITL, SemConnect, CS API, or raw MAVLink telemetry to be live in the
first consumer proof.

#### Scenario: Single-node report is accepted

- **WHEN** SemOps receives a SemLink `single-node-companion-demo` report or equivalent fixture
- **THEN** it preserves `kind`, `generated_at`, `vehicle_profile`, companion node identity, vehicle count, endpoint
  probe status, `/api/evidence` contract status, command posture, simulator-command posture, SemOps readback adapter
  status, and assertion results
- **AND** it treats the evidence as SemLink-produced demo evidence, not graph-backed live COP evidence
- **AND** it does not require a running SemLink process for the fixture consumer test to pass

#### Scenario: Simple mesh report is accepted

- **WHEN** SemOps receives a SemLink `simple-mesh-companion-demo` report with `n` companion nodes
- **THEN** it preserves generated time, vehicle profile, expected summary count, companion node IDs, vehicle counts,
  peer counts, watermark counts, selected-state summary counts, bounded diff counts, TTL merge posture, raw MAVLink
  exclusion posture, and assertion results
- **AND** the accepted evidence can represent boats, drones, or rovers through vehicle profile and MAVLink-native
  vehicle evidence rather than through BlueOS-only assumptions
- **AND** SemOps does not consume raw MAVLink frames as mesh evidence

### Requirement: Companion Fleet View Is Curated COP Glass

SemOps SHALL expose SemLink demo evidence through a curated companion-fleet view model for COP/GCS glass instead of
passing raw SemLink reports directly to the browser.

#### Scenario: Companion fleet view is produced

- **WHEN** SemLink report evidence has been normalized
- **THEN** the SemOps view model includes companion node identity, vehicle profile, vehicle count, peer count,
  watermark posture, summary count, diff posture, readback status, assertion state, and no-transmit posture
- **AND** fields absent from a report kind are represented as unavailable evidence rather than guessed values
- **AND** raw report details, peer URLs, native packets, and raw MAVLink frames remain behind the API unless a later
  diagnostic workflow explicitly exposes them

#### Scenario: Readback evidence keeps status and result separate

- **WHEN** a SemLink report contains SemOps readback adapter evidence
- **THEN** SemOps preserves the native readback request/response status separately from any `COMMAND_ACK` status or
  decoded `AUTOPILOT_VERSION` result evidence
- **AND** the view model keeps readback intent/status visible without implying native command execution authority

### Requirement: Browser Smoke Proves N Companion Nodes

SemOps SHALL provide browser or component smoke evidence that the COP can show and inspect `n` SemLink companion nodes
from deterministic SemLink demo evidence.

#### Scenario: Operator sees the SemLink fleet

- **WHEN** the demo fixture contains at least three SemLink companion nodes
- **THEN** the COP surface shows a SemLink companion fleet source or equivalent source evidence
- **AND** it exposes the node count, vehicle profile, node identities, vehicle counts, peer/watermark posture, and
  assertion state
- **AND** selecting or inspecting the SemLink fleet shows no native transmit authority, no companion hardware transmit
  authority, and raw MAVLink exclusion posture
- **AND** fixture/report evidence is labelled so it cannot be mistaken for live BlueOS, Navigator, hardware, or
  radio/mesh reliability evidence

### Requirement: SemOps Does Not Own SemLink Mesh Mechanics

SemOps SHALL keep SemLink companion runtime and mesh mechanics out of the SemOps implementation boundary.

#### Scenario: Mesh evidence is displayed without control

- **WHEN** SemOps renders SemLink simple-mesh evidence
- **THEN** peer counts, watermarks, selected-state catch-up, bounded diffs, and TTL merge posture are displayed as
  evidence
- **AND** SemOps does not expose peer management, mesh topology editing, service registration, BlueOS package
  lifecycle, Navigator readiness controls, raw MAVLink forwarding, or companion command-rule controls

#### Scenario: Standards projection remains an edge concern

- **WHEN** SemOps projects accepted SemLink readback or companion fleet evidence to SemConnect/CS API
- **THEN** the projection preserves SemLink source provenance, MAVLink target/system concepts, no-transmit posture,
  readback status, and result evidence where those concepts fit
- **AND** CS API/SemConnect remains an interoperability edge rather than a required dependency for the SemLink
  companion runtime or the SemLink-to-SemOps hot path
