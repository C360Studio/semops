## ADDED Requirements

### Requirement: SemOps Ingests SemLink-Generated Demo Artifacts

SemOps SHALL provide a local handoff path for SemLink-generated companion demo artifacts before requiring a
long-running SemLink/SemOps orchestration stack.

#### Scenario: Generated SemLink artifact is accepted

- **WHEN** SemOps imports a SemLink-generated companion demo artifact from a file, CLI input, or equivalent local
  handoff
- **THEN** it preserves the SemLink report kind, generated time, vehicle profile, companion node identities, vehicle
  counts, assertion summary, no-transmit posture, and source provenance
- **AND** it labels the evidence as SemLink-generated rather than committed fixture evidence
- **AND** it does not require SemOps to start, stop, supervise, or health-check SemLink, ArduPilot SITL, BlueOS,
  Navigator, or SemLink mesh peers

#### Scenario: Unsupported or overclaimed artifact is rejected

- **WHEN** a generated artifact uses an unsupported report kind, malformed payload, stale generated time under an
  enabled freshness policy, or a SITL/hardware fidelity claim without simulator or source metadata
- **THEN** SemOps rejects the artifact or marks it unusable for demo proof
- **AND** the rejection explains whether the failure is report-shape, freshness, source-fidelity, assertion, or
  governance posture related

### Requirement: First ArduPilot Fidelity Proof Uses One SITL-Backed SemLink Node

SemOps SHALL support a mixed live-demo ladder where one SemLink node can be backed by no-Gazebo ArduRover / ArduPilot
SITL while the fleet proof remains `n >= 3` SemLink companion nodes.

#### Scenario: One SITL-backed node is displayed

- **WHEN** a SemLink-generated artifact identifies one companion node as backed by ArduPilot SITL telemetry
- **THEN** SemOps renders that node as SITL-backed ArduPilot evidence
- **AND** the fleet view still exposes at least three SemLink companion nodes when deterministic or generated mesh
  evidence is present
- **AND** the UI distinguishes SITL-backed nodes from deterministic, fixture, generated-only, BlueOS, Navigator,
  hardware, and radio/mesh reliability evidence

#### Scenario: N SITLs are required only for N live vehicles

- **WHEN** a demo claim says `n` live ArduPilot vehicles, `n` live ArduPilot SITL instances, or per-node live vehicle
  fidelity
- **THEN** each claimed live node must have a distinct SITL or hardware-adjacent vehicle source with distinct MAVLink
  identity and route evidence
- **AND** deterministic mesh summaries or peer watermarks alone cannot satisfy that live-vehicle claim
- **AND** SemOps may still claim `n` SemLink companion nodes when only one node is SITL-backed, provided that mixed
  fidelity is explicit

### Requirement: Generated Artifact COP Glass Is Curated And Non-Transmitting

SemOps SHALL render generated SemLink/SITL artifacts through the curated companion-fleet view model without broadening
command authority.

#### Scenario: Operator inspects generated SITL evidence

- **WHEN** the COP displays a generated SemLink artifact with SITL-backed node evidence
- **THEN** the operator can inspect companion node identity, vehicle profile, vehicle count, source-fidelity label,
  peer count, watermark posture, assertion state, readback adapter status, and no-transmit posture
- **AND** readback adapter status, `COMMAND_ACK`, and `AUTOPILOT_VERSION` result evidence remain separate if present
- **AND** the UI states that native execution, companion hardware transmit, mission upload, mode change, arm/disarm,
  offboard control, and hardware command authority are not granted by this demo

#### Scenario: SemOps does not expose SemLink runtime controls

- **WHEN** SemOps renders SemLink-generated companion or SITL evidence
- **THEN** it does not expose SemLink process controls, SITL launch controls, peer management, mesh topology editing,
  BlueOS service lifecycle controls, Navigator readiness controls, raw MAVLink forwarding, or companion command-rule
  controls
- **AND** CS API/SemConnect remains a SemOps interoperability projection edge, not a required runtime dependency for
  SemLink-to-SemOps artifact ingest

### Requirement: Live Demo Smoke Uses SemLink Output

SemOps SHALL provide smoke evidence that the browser can display a SemLink-generated artifact, not only a static
SemOps-authored fixture.

#### Scenario: Browser smoke proves generated artifact handoff

- **WHEN** a smoke imports a SemLink-generated companion artifact
- **THEN** the browser shows at least three SemLink companion nodes
- **AND** it marks at least one node as SITL-backed when the artifact contains that source metadata
- **AND** it shows the artifact as SemLink-generated evidence with source provenance
- **AND** it preserves no-transmit posture and avoids `n` live ArduPilot vehicle claims unless the artifact contains
  `n` distinct live vehicle sources
