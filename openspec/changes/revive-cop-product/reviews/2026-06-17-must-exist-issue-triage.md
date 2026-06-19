# ADR-055/056 Must-Exist Issue Triage

Date: 2026-06-17

Trigger:

- SemOps GitHub issue #1: ADR-055/056 must-exist flip compliance.

Decision:

- Treat issue #1 as a sequencing gate for the revival.
- Prioritize generated or replay MAVLink live graph smoke before PX4/SITL, UI, or second-feed expansion.
- Keep PX4/SITL as feed-fidelity and command/control evidence, not as the first born-first compliance proof.

Evidence Checked:

- The legacy paths named in the issue are absent locally after the revival cleanup:
  `pkg/processors/mavlink/rules/battery_monitor.go` and the ignored MAVLink payloads no longer exist.
- Local SemOps docs and OpenSpec cite ADR-055/056 explicitly.
- `pkg/cop` contracts derive ForeignEdgeClaim records for `cop.track.source`.
- MAVLink projector tests prove source asset birth before track birth and no repeated strict source edge on updates.
- MAVLink graph writer and structural wiring tests use `create_with_triples` and `update_with_triples`, not
  `triple.add`.
- 2026-06-17 generated-frame live graph smoke passed against SemStreams `configs/graph-backend.json`:
  `SEMOPS_MAVLINK_LIVE_GRAPH_NATS_URL=nats://127.0.0.1:55438 go test ./internal/smoke/mavlink -v`.

Adversarial Findings:

- Architect: The design is compliant, but GitHub cannot see the evidence until the local revival commits are pushed
  or opened as a PR.
- Go reviewer: The generated-frame smoke now exercises a live SemStreams graph path and polls graph state. The next
  clean-stack proof must add owner-registry and counter assertions.
- Go reviewer: Restart/replay remains a genuine operational risk because projector birth knowledge is in memory.
  Read-back, checkpointing, or idempotent create handling is required before process-restart safety claims.
- Technical writer: The plan must separate graph-contract compliance from simulator fidelity so PX4/SITL does not
  block the breaking-tag gate.

Accepted Risks:

- The live smoke can start from generated frames instead of PX4/SITL because the question is graph semantics, not
  vehicle simulator fidelity.
- Dropped-foreign-edge evidence may depend on SemStreams exposing a counter or state surface in the current tag; if
  absent, the smoke must record that limitation and check the best available graph/error response.

Follow-Up:

- Add COP-009 as the issue #1 tracker.
- Keep the generated/replay MAVLink live graph smoke as the first SemStreams graph gate.
- Update GitHub issue #1 with design evidence after the local branch is pushed or opened as a PR.
- Add explicit owner registration, heartbeat, and clean-stack graph-ingest counter coverage.
- Keep PX4/SITL evidence open for live command/control and interoperability claims.
