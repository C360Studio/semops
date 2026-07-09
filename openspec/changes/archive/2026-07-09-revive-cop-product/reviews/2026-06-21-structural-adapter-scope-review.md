# Structural Adapter Scope Review

Date: 2026-06-21
Scope: Closing task 6.2 and splitting broader EDXL from the implemented Phase 1 structural adapters

## Decision

Close task 6.2 for the implemented Phase 1 structural adapter set: MAVLink, TAK/CoT, and CAP.

Do not treat that closure as a broader EDXL claim. CAP is the only EDXL-family artifact currently supported in the
structural COP path, and even CAP remains scoped as parser/projection/readback and opt-in component evidence rather
than default live NWS/IPAWS service support.

## Evidence

- MAVLink has componentized UDP input, decoder processor, projector processor, graph writer, restart reconciliation,
  live graph smoke, and hosted UDP-to-COP snapshot smoke.
- TAK/CoT has UDP/TCP input components, decoder processor, projector processor, graph writer, live graph smoke, and
  hosted UDP-to-COP snapshot readback for track/task/advisory state.
- CAP has parser fixtures, lifecycle replay, born-first append-evidence graph writer, componentized HTTP poller,
  decoder, projector, direct live graph smoke, and Caddy-routed COP snapshot readback.
- The scenario runner and shared-airspace smoke now prove those structural feeds are product-visible together in the
  local Compose COP.

## Boundary

Broader EDXL support remains future feed-validation work. It should not enter Phase 1 by implication from `CAP/EDXL`
wording. Any EDXL-DE, EDXL-HAVE, EDXL-RM, or other EDXL-family support needs its own product force, authoritative
fixtures or schemas, entity model, projection ownership contract, indexing/cardinality review, and COP readback gate.

## Follow-Up

Track broader EDXL as task 6.58 until a specific product need justifies it. If no near-term need appears, leave it as
roadmap scope rather than MVP debt.
