# SemLink Companion UI Readback Review

Scope: browser fixture and selected-entity readback for SemLink companion ArduPilot command intent posture.

Verdict: accept.

The slice carries the SemLink command-task posture from the COP API contract into the Svelte GCS-glass fixture and
selected-entity inspector. It makes the no-transmit boundary visible to operators without adding execute, retry,
cancel, arbitration, CS API tasking, or native/companion transmit controls.

Boundary choices:

- The fixture adds a second command task for `AUTOPILOT_VERSION`; the TAK marker task and existing CS API/local command
  fixture remain intact.
- The task is non-spatial, so map point/polygon/ray behavior does not change; only task selection and inspector
  evidence expand.
- The selected inspector shows `local_override_policy` and `claim_posture` as readback metadata, not action controls.
- Playwright asserts the SemLink task row, provenance source ref, SemLink authority, local override posture, and
  no-native/no-companion-transmit text.

Evidence:

- `npm run check`
- `npm run test`
- `npm run test:e2e`
- `npm run build`

Residual risk:

- This is fixture-backed browser evidence. A later hosted stack smoke can assert the same `claim_posture` after a live
  SemLink ingress route or BlueOS extension call path exists.
