# SemLink Mesh Node Auth Binding Review

Scope: authenticated caller binding for the hosted SemLink ArduPilot readback route.

Decision: accept a required mesh-node caller header and bind it to the request body before admission.

The trusted-header auth gate now requires `X-SemOps-SemLink-Mesh-Node-ID`. The COP API compares that value with the
request `mesh_node_id` after JSON decoding and before SemLink ingress admission, graph target lookup, or command-intent
graph writes. Successful responses include `authorized_mesh_node_id` so GCS-glass readback can show which authenticated
companion boundary submitted the intent.

Boundary choices:

- The mesh-node header is caller provenance for the API boundary, not SemStreams or mesh causality metadata.
- A trusted caller with `semlink.readback.intent` authority cannot request readback intent for a different mesh node.
- The binding still authorizes only governed `AUTOPILOT_VERSION` readback intent; it does not enable native MAVLink or
  companion transmit.

Evidence:

- `internal/api/cop/semlink_readback_auth.go` requires and validates `X-SemOps-SemLink-Mesh-Node-ID`.
- `internal/api/cop/semlink_readback.go` rejects mesh-node mismatch before ingress admission or graph writes.
- `internal/api/cop/handler_test.go` proves mismatch does not call ingress or writer.
- `cmd/semops/main_test.go` proves the hosted path returns `authorized_mesh_node_id` for accepted readback intent.

Residual risk:

- This does not establish cryptographic identity for a live SemLink node. The route still relies on an upstream trusted
  header boundary, and live BlueOS/Navigator trust establishment remains a future slice.
