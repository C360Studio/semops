# CAP Live Graph Smoke Review

Date: 2026-06-19

## Decision

Accept the CAP live graph smoke as SemStreams graph-contract evidence for the CAP slice.

Do not claim CAP consumer conformance, NWS/IPAWS service integration, hosted polling, authoritative hazard lifecycle,
or complete CAP update/cancel/expire behavior from this gate.

## Objections Raised

- A passing live graph smoke could be mistaken for CAP standards conformance.
- CAP append-evidence writes could accidentally become an ownership/write-fence claim over hazard state.
- A prefix-discovery readback might hide duplicate CAP entities if the entity ID derivation is unstable.
- Triple-order assumptions could make the update evidence assertion flaky because create and update evidence may both
  exist on the same entity.
- The stack smoke could become too broad to diagnose if CAP failures are not isolated from hosted MAVLink/CoT checks.

## Evidence Checked

- `internal/smoke/cap/live_graph_test.go` skips unless `SEMOPS_CAP_LIVE_GRAPH_NATS_URL` points at a live SemStreams
  graph stack.
- The smoke registers first-phase COP ownership and uses registry/bind-derived owner tokens before writing.
- The create plan births the CAP `hazard_area` entity before the update plan appends evidence.
- The test fails on `entity_not_found` or `foreign_edge_dropped` mutation errors.
- The readback uses `graph.query.prefix` and selects the CAP hazard entity by ID because SemStreams may also return
  hierarchy container entities under the same prefix.
- The update assertions are append-aware and do not rely on triple order.
- The negative assertion verifies CAP did not write `cop.hazard.geometry`, `cop.hazard.severity`, or
  `cop.hazard.status`.
- `scripts/cop-stack-smoke.sh` runs the CAP live graph smoke after the hosted MAVLink/CoT snapshot smokes and direct
  MAVLink/CoT graph smokes, while preserving an independent `SEMOPS_CAP_LIVE_GRAPH_NATS_URL` override.

## Accepted Risks

- SemStreams may still log an append-evidence owner warning until evidence-contribution declarations are fully
  separated from enforceable ownership claims.
- The smoke uses a deterministic local CAP fixture, not a captured NWS fixture or OASIS conformance suite.
- CAP lifecycle behavior is visible as evidence JSON only; no stale/inactive state transition is enforced yet.
- The CAP feed is not hosted as a poller or webhook service.

## Follow-Up Tasks

- Capture NWS CAP samples as deterministic fixtures with explicit source provenance.
- Add XML schema and consumer-rule validation before any CAP conformance language appears in the demo.
- Add update/cancel/expire lifecycle projection and COP stale/inactive readback.
- Decide whether authoritative hazard lifecycle belongs to a CAP-owned control projection or a fusion-owned derived
  projection.
- Revisit SemStreams evidence-declaration ergonomics after typed evidence contributions land upstream.
