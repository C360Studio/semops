# SAPIENT Runtime Graph Promotion Review

Date: 2026-06-23
Scope: opt-in SAPIENT decoded-message graph projector component and hosted runtime wiring

## Decision

Accept a graph-producing SAPIENT runtime path only for the already-reviewed absolute-location detection contract.

`cmd/semops` may compose HTTP input -> decoder -> projector when `SEMOPS_SAPIENT_ENABLED=true` and
`SEMOPS_SAPIENT_GRAPH_ENABLED=true`. Runtime ownership registers `OwnerSAPIENT` only under that graph flag, and the
projector must receive SemStreams-minted owner tokens from the registry/bind path.

This does not approve SAPIENT product support, SAPIENT conformance, a SemOps-hosted SAPIENT service, tasking, alert
acknowledgements, lifecycle state, association, UTM conversion, range/bearing conversion, Apex middleware behavior, or
full-message coverage.

## Red-Team Findings

1. A second graph flag is required.

   `SEMOPS_SAPIENT_ENABLED=true` is a preflight/runtime-ingress switch. It must not silently register ownership or
   write graph state. The graph-producing path is separately gated by `SEMOPS_SAPIENT_GRAPH_ENABLED=true`.

2. Default fixtures must not fake graph support.

   `/sapient/messages` remains the deterministic task-ack decoded-stream fixture for Compose smoke. Graph development
   uses `/sapient/detections` or another detection-producing source explicitly, so a tasking/control message cannot
   masquerade as track state.

3. Existing-birth reconciliation must stay born-first.

   The projector may retry a create conflict as an update only after the projector marks the native SAPIENT detection
   as already born. It must not fall back to auto-vivify or direct `triple.add` behavior.

4. Component promotion is not service promotion.

   A SemStreams processor component with graph request ports proves flow/runtime shape. It does not prove SAPIENT
   sessions, command authority, Apex interop, compliance, or product-hosted service behavior.

5. Non-detection messages remain decoded evidence.

   Task acknowledgements and future tasking/control messages should become separate `control` contracts after command
   authority review. They must not reuse the detection track projector.

## Evidence Accepted

- `internal/components/sapient` adds a decoded-message graph projector processor with SemStreams stream input and graph
  request output ports.
- `internal/projectors/sapient` adds a graph writer that uses SemStreams create/update request subjects and surfaces
  mutation failures for born-first reconciliation.
- `internal/app` registers SAPIENT ownership and composes the projector only when
  `SEMOPS_SAPIENT_GRAPH_ENABLED=true`.
- `cmd/semops-feed-fixtures` now exposes `/sapient/detections` as deterministic absolute-location graph-development
  input while keeping `/sapient/messages` for preflight task-ack smoke.

## Follow-Ups

- Run or qualify the Dstl BSI Flex 335 v2 Test Harness before any SAPIENT compliance language.
- Keep range/bearing, UTM, tasking, alert acknowledgements, detection lifecycle, association, and Apex middleware
  behavior as separate reviewed contracts.
- Keep full-stack SAPIENT detection graph smoke explicitly opt-in; the accepted smoke gate is recorded in
  `2026-06-23-sapient-graph-stack-smoke-review.md`.
