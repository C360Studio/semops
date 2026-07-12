# Tasks

## 1. Spec

- [x] 1.1 Define the SemOps mesh demo claim for SemLink one-to-one companion
  artifacts.
- [x] 1.2 Gate live/radio/BlueOS/SITL/command claims separately from the
  deterministic mesh artifact.

## 2. Implementation

- [x] 2.1 Add or refresh a SemLink-generated one-to-one mesh artifact fixture
  with real SemLink source metadata.
- [x] 2.2 Add a SemOps smoke that imports the generated mesh artifact and
  asserts `active_companion_nodes == node_count`.
- [x] 2.3 Assert the COP companion fleet exposes `expected_summaries == 3`,
  three companion nodes, one vehicle per node, raw MAVLink exclusion,
  deterministic source fidelity, and no-transmit posture.
- [x] 2.4 Add browser or component coverage for the operator-facing mesh demo
  view.
- [x] 2.5 Document the cross-repo demo command sequence from SemLink artifact
  generation to SemOps COP display.

## 3. Validation

- [x] 3.1 Run focused SemOps parser/provider tests for SemLink artifacts.
- [x] 3.2 Run the COP browser/component smoke for the mesh demo surface.
- [x] 3.3 Run `openspec validate --all --strict`.
