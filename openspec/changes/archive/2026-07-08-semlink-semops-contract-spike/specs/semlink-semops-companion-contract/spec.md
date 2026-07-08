## ADDED Requirements

### Requirement: Native Companion Contract Is Standards-Mapped

SemOps SHALL define the MVP SemLink-to-SemOps companion readback contract as a native low-friction contract with a
required standards mapping table.

#### Scenario: Contract field is accepted

- **WHEN** a field appears in the native SemLink companion readback request, response, error, duplicate, or status
  fixture
- **THEN** the contract maps the field to MAVLink, CS API, SensorThings, SWE, or another named standard where a
  natural mapping exists
- **AND** fields without a natural standards mapping are marked as C360 governance exceptions
- **AND** each governance exception states why it is required, which boundary owns it, and why it is not domain
  vocabulary

#### Scenario: Private vocabulary is rejected

- **WHEN** a proposed field uses C360 vocabulary for a vehicle-native or standards-owned concept
- **THEN** the field is renamed or reshaped to use MAVLink, CS API, SensorThings, or SWE vocabulary before the
  contract can be accepted

### Requirement: SemLink Hot Path Does Not Require CS API By Default

The MVP SemLink-to-SemOps companion path SHALL NOT require SemLink to expose or consume CS API unless a friction spike
proves CS API reduces operational friction and preserves command-governance semantics.

#### Scenario: SemLink submits MVP readback intent

- **WHEN** SemLink acts as a boat-local BlueOS/Navigator/Pi companion node for an ArduPilot vehicle
- **THEN** it may submit the MVP `AUTOPILOT_VERSION` readback request through the native SemOps companion contract
- **AND** the request uses MAVLink-native command/message and target system/component identifiers
- **AND** the request carries only the SemOps governance fields required for trusted caller posture, companion-node
  provenance, idempotency, target proof, correlation, TTL, and audit
- **AND** SemConnect is not a required runtime dependency for this hot-path request

#### Scenario: Raw telemetry remains native

- **WHEN** SemLink emits high-rate or autopilot-facing MAVLink telemetry
- **THEN** that telemetry remains on MAVLink/native telemetry ports or bounded raw lanes
- **AND** CS API/SemConnect is used for interoperable state, tasking records, status, and observations rather than as a
  high-rate raw MAVLink transport

### Requirement: SemLink Feedback Is Dispositioned Before Acceptance

SemOps SHALL request and disposition SemLink feedback before accepting the MVP native companion contract.

#### Scenario: SemLink reviews the draft contract

- **WHEN** the native companion contract draft and fixtures are ready for review
- **THEN** SemOps shares them with SemLink before the MVP contract is accepted
- **AND** SemLink feedback is classified as blocking, friction, interop, or deferred
- **AND** blocking or friction feedback is resolved by a contract change, documented non-goal, or follow-up task before
  SemLink resumes integration implementation

#### Scenario: Feedback changes the contract

- **WHEN** SemLink identifies a contract field, header, timing rule, status shape, or fixture that duplicates companion
  runtime concepts or adds avoidable implementation friction
- **THEN** SemOps either revises the contract or records why the field remains necessary as a governance exception
- **AND** the standards mapping table is updated when the feedback changes field names, meanings, or projection shape

### Requirement: SemOps Projects SemLink State To CS API Through SemConnect

SemOps SHALL prove that accepted SemLink readback intent and status can be projected through SemConnect/CS API at the
SemOps standards edge without making SemConnect the command-authority layer.

#### Scenario: Accepted readback intent is projected

- **WHEN** SemOps accepts a SemLink `AUTOPILOT_VERSION` readback intent
- **THEN** the interop proof maps the intent to CS API-shaped System, ControlStream, Command, and status or event
  resources where those concepts fit
- **AND** the projection preserves MAVLink command/message IDs, target system/component IDs, correlation, sender or
  source provenance, status, no-transmit posture, and result/readback evidence
- **AND** SemConnect remains the standards bridge and conformance anchor rather than the authority that decides whether
  the command is safe to issue

#### Scenario: CS API mapping gap is found

- **WHEN** CS API/SemConnect cannot carry a SemLink readback fact without semantic loss or private shape overloads
- **THEN** the spike records either a C360 governance exception or a SemConnect follow-up
- **AND** SemOps does not add local CS API gateway behavior to hide the gap

### Requirement: Friction Score Controls CS API Promotion

SemOps SHALL score CS API/SemConnect suitability before promoting CS API into the SemLink-to-SemOps companion path.

#### Scenario: CS API suitability is reviewed

- **WHEN** the native contract fixtures and SemConnect projection proof are complete
- **THEN** the review records a Green, Amber, or Red score for CS API/SemConnect as a SemLink-to-SemOps path
- **AND** Green means CS API carries the request and status with less or equal friction than the native contract
- **AND** Amber means CS API remains the SemOps interop projection while the native contract remains the hot path
- **AND** Red means CS API projection loses meaning or requires private overloads, and follow-up work is routed before
  any interoperability claim expands
