## Context

SemLink now exposes two useful demo evidence shapes:

- `single-node-companion-demo`, which proves one companion runtime, `/register_service`, `/api/evidence`, command
  safety posture, simulator-only command evidence, and SemOps readback adapter compatibility.
- `simple-mesh-companion-demo`, which proves multiple local companion runtimes, peer counts, selected-state catch-up,
  watermarks, bounded diffs, TTL merge posture, and raw MAVLink exclusion.

SemOps already has the native SemLink readback contract, command/readback projection, no-transmit posture, COP API,
and map-first UI foundations. The missing product proof is the consumer side: can SemOps make a useful GCS/COP view
from SemLink's evidence without pulling SemLink's runtime responsibilities into SemOps?

## Goals / Non-Goals

**Goals**

- Prove SemOps can consume SemLink report/API evidence for `n` companion nodes.
- Preserve SemLink as the companion runtime, mesh, BlueOS/Navigator/Pi, and local command-safety owner.
- Preserve SemOps as the COP/GCS glass, cross-node operator view, command authority/admission, and interop projection
  owner.
- Start with deterministic fixtures or generated reports before any live SemLink/SemOps compose stack.
- Make the browser proof inspectable: node identity, vehicle profile, vehicle counts, mesh posture, readback status,
  and no-transmit posture must be visible.

**Non-Goals**

- Run SemLink as a subprocess from SemOps.
- Implement SemLink mesh sync, MAVLink telemetry ports, BlueOS service registration, or companion command rules in
  SemOps.
- Require SemConnect or CS API in the SemLink-to-SemOps hot path.
- Claim live hardware, BlueOS, Navigator, ArduPilot SITL, or radio/mesh reliability from fixture-only evidence.
- Add executable command controls to the COP UI.

## Decisions

### 1. SemOps consumes reports first

The first SemOps implementation should read SemLink report/API-shaped fixtures, not live SemLink processes. That
creates a hold-out consumer proof against SemLink's completed e2e surface and lets the SemOps API/UI contract stabilize
before a compose stack adds timing, auth, and network concerns.

### 2. The demo view model is curated COP glass

SemOps should expose a product view model, not a raw SemLink report passthrough. The view model should preserve:

- `kind`, `generated_at`, and `vehicle_profile`;
- companion node identity and vehicle counts;
- peer count, watermark count, summary counts, bounded diff counts, and TTL merge posture for mesh reports;
- SemOps readback adapter status and correlation evidence when present;
- command posture showing simulator-only evidence and no native or companion hardware transmit authority;
- raw MAVLink exclusion as a policy fact, not a hidden implementation detail.

### 3. The UI proof is fleet inspection, not topology control

The browser should let an operator see and inspect companion nodes and their vehicles in the COP source/inspector
surfaces. It should not expose mesh topology controls, peer management, BlueOS controls, or command execution. Mesh
counts and watermarks are evidence badges; SemLink remains the runtime owner.

### 4. Standards projection stays at the SemOps edge

The native SemLink evidence remains the demo input. SemOps may project accepted readback/status/result facts through
SemConnect/CS API for standards interoperability, but this change must not make CS API a SemLink runtime dependency or
hide a missing native field behind a private CS API shape.

## Risks / Trade-offs

- A report fixture can drift from SemLink if SemLink changes its e2e shape. The mitigation is to keep the fixture
  small, named by report kind, and reviewed against SemLink's generated artifacts before live compose work.
- A UI could overstate fixture evidence as live mesh readiness. The smoke must label fixture/report evidence and keep
  hardware, BlueOS, Navigator, and radio reliability claims out of scope.
- A raw report passthrough would be faster, but it would couple the COP UI to SemLink internals. A curated view model
  keeps SemOps product ownership intact.

## Implementation Order

1. Add SemLink report fixtures or a tiny report parser package and a table-driven consumer test.
2. Map parsed reports into a COP companion-fleet view model.
3. Expose the view through the COP API or snapshot/runtime facade.
4. Add UI/smoke assertions for `n` nodes, vehicle profile, peer/watermark evidence, readback status, and no-transmit
   posture.
5. Only then wire a live SemLink/SemOps compose demo if fixture evidence is stable.
