# SemLink Companion Command Readback Review

Scope: COP API readback for SemLink companion ArduPilot command intent.

Verdict: accept.

The slice keeps SemLink companion requests in the governed command-task read model and adds only visible claim posture.
The API now maps command-task `claim_posture` for SemLink-origin intents so GCS glass can inspect mesh-node readback
requests without implying local execution, companion transmit, CS API tasking, cancellation, retry, or arbitration
authority.

Boundary choices:

- SemLink detection is derived from command intent authority, provenance source, or `semlink://` source references.
- Command-task source remains `command` and owner remains `semops.command.intent`; SemLink provenance is visible through
  source refs and the posture text.
- The field is `omitempty`, so unrelated command tasks do not grow empty API noise.
- No route, handler, UI button, native transmitter, or companion transmitter was added.

Evidence:

- `internal/api/cop/graph_provider_test.go` now discovers a SemLink companion `AUTOPILOT_VERSION` command task by the
  existing command prefix and asserts owner, source ref, authority, local override policy, discovery count, and posture.
- `internal/api/cop/graph_provider.go` derives the posture in the read model only.
- `openspec/changes/revive-cop-product/specs/cop-ui-experience/spec.md` keeps command lifecycle UI read-only and adds
  the SemLink intent-only/no-transmit readback requirement.

Residual risk:

- The browser selected-entity inspector has fixture-backed command task coverage, but this slice stops at the COP API
  read model. A later UI fixture can assert `claim_posture` text once SemLink task fixtures enter the browser mock.
- There is still no authenticated SemLink ingress route, BlueOS extension call path, or live mesh-node evidence in
  SemOps.
