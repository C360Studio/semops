# SAPIENT Runtime Preflight Review

Scope: opt-in hosted SAPIENT HTTP input -> decoder preflight chain

## Decision

Accept SAPIENT app-runtime wiring as an opt-in SemStreams preflight component flow. `cmd/semops` may compose the
SAPIENT HTTP input and decoder processor when `SEMOPS_SAPIENT_ENABLED=true`.

This approves runtime lifecycle wiring, env/config coverage, replay capture, and raw/decoded stream publication for
local provider-shaped HTTP fixtures. It does not approve SAPIENT product support, conformance, graph projection,
`OwnerSAPIENT`, graph request ports, or tasking/control semantics.

## Adversarial Findings

- Runtime hosting must not create implied product support. The app path is named and documented as preflight, requires
  an explicit URL when enabled, and Compose defaults it off.
- Runtime ownership must not broaden. The SAPIENT preflight flow does not append ownership contracts, mint an owner
  token, or subscribe a decoded graph projector path.
- HTTP/Apex-shaped polling is not conformance. The Dstl harness remains the compliance gate, and the current path only
  proves local raw/decoded stream handling.
- Replay is native evidence, not canonical state. JSON/protobuf bytes remain on bounded raw lanes and replay stores
  until a projection ownership/indexing review approves graph entities.
- Protobuf support remains descriptor-based preflight. Generated bindings and full-message coverage stay separate
  gates.

## Accepted Evidence

- `internal/app` now parses `SEMOPS_SAPIENT_*` config and validates enabled-runtime settings.
- App startup starts only the SAPIENT decoder processor and HTTP input component behind `SEMOPS_SAPIENT_ENABLED=true`.
- Local provider-shaped HTTP fixtures drive raw -> decoded preflight streams and JSONL replay capture.
- Runtime ownership remains unchanged when SAPIENT preflight is enabled.
- `compose.cop.yml` passes SAPIENT runtime env through while defaulting `SEMOPS_SAPIENT_ENABLED=false`.

## Follow-Ups

- Run or qualify the official Dstl BSI Flex 335 v2 Test Harness before compliance language appears.
- Decide whether generated Go bindings are needed beyond dynamic descriptors.
- Review source identity, ownership, indexing, and service mode before introducing `OwnerSAPIENT` or graph projection.
- Consider a portable Linux/CI SAPIENT preflight suite as an ecosystem contribution.
