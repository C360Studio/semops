## 1. OpenSpec Planning

- [x] 1.1 Create `semlink-multi-companion-cop-demo` as a focused SemOps change after archiving
      `revive-cop-product`.
- [x] 1.2 Record the SemOps/SemLink boundary: SemLink owns companion runtime and mesh mechanics; SemOps owns COP/GCS
      glass and cross-node operator view.
- [x] 1.3 Define demo acceptance requirements before implementation.
- [x] 1.4 Run strict OpenSpec validation for the new change and all specs.

## 2. SemLink Evidence Consumer

- [ ] 2.1 Add SemOps fixture(s) matching SemLink `single-node-companion-demo` and `simple-mesh-companion-demo`
      reports, with report kind and generated time preserved.
- [ ] 2.2 Add a small parser/normalizer that accepts SemLink demo report fields needed by SemOps and rejects unknown
      report kinds.
- [ ] 2.3 Add table-driven tests for single-node, multi-node, failed assertion, missing node identity, and raw MAVLink
      exclusion posture.

## 3. COP API/View Model

- [ ] 3.1 Define a SemOps companion-fleet view model for companion nodes, vehicle profile, vehicle counts, mesh
      readiness evidence, readback adapter status, and no-transmit posture.
- [ ] 3.2 Map SemLink evidence into the view model without exposing raw mesh payloads or raw MAVLink frames.
- [ ] 3.3 Add API or runtime-facade tests proving `n` companion nodes and vehicles are discoverable from fixture
      evidence.
- [ ] 3.4 Preserve SemLink `COMMAND_ACK` status and `AUTOPILOT_VERSION` result separation when readback evidence is
      present.

## 4. UI And Smoke Evidence

- [ ] 4.1 Render companion fleet evidence in source/inspector surfaces with node identity, vehicle profile, peer
      counts, watermarks, and no-transmit posture.
- [ ] 4.2 Add browser or component smoke assertions for at least three SemLink companion nodes.
- [ ] 4.3 Ensure fixture/report evidence is labelled as demo evidence and cannot be mistaken for live BlueOS,
      Navigator, hardware, or radio/mesh reliability.
- [ ] 4.4 Keep command execution, mesh topology controls, BlueOS controls, and raw MAVLink inspection out of the MVP
      UI surface.

## 5. Standards And Boundary Review

- [ ] 5.1 Recheck the SemConnect/CS API projection edge after the companion-fleet view model exists.
- [ ] 5.2 Record whether any SemLink report field creates a standards mapping gap or local C360 governance exception.
- [ ] 5.3 Run an adversarial review before claiming the demo proves `n` SemLink boats, drones, or rovers in SemOps.
- [ ] 5.4 Run `go test ./...`, `go build ./...`, and relevant browser/stack smoke gates when implementation lands.
