# Feed Service Promotion And CS API Review

Date: 2026-06-19

## Decision

Accept the feed roadmap hardening as the current planning gate.

SemOps should keep native adapters first-class and bounded for the MVP, while preserving service seams for feeds that
will eventually need SemOps-owned servers or gateways. CS API remains a standards-facing bridge through SemConnect
unless SemOps is explicitly rechartered to own a CS API gateway product.

## Evidence Checked

- `docs/feed-product-roadmap.md`
- `docs/feed-validation-and-indexing-ladder.md`
- `openspec/changes/revive-cop-product/feed-evidence/semconnect-csapi-egress.md`
- `openspec/changes/revive-cop-product/specs/feed-validation-ladder/spec.md`
- `openspec/changes/revive-cop-product/specs/cop-product-ownership/spec.md`
- OGC API - Connected Systems overview and SWG repository

## Adversarial Findings

- CS API core-risk: rejected. CS API is valuable for ingress/egress, dynamic data, command/control, snapshots, and
  conformance, but making it the internal COP language would slow native HADR feed adoption and move SemOps ownership
  decisions into an external exchange schema.
- Adapter dead-end risk: reduced. The service-promotion matrix now names the future server/gateway path for TAK/CoT,
  CAP, CS API, ADS-B, SAPIENT, and KLV without expanding MVP scope.
- Command bypass risk: reduced. CS API ControlStream and Command input must route through SemOps command authority,
  native safety checks, audit, and replay before feed-owned state changes.
- Compliance overclaim risk: reduced. CS API conformance remains SemConnect/harness evidence; it does not imply
  native feed conformance for MAVLink, TAK/CoT, CAP, ADS-B, SAPIENT, or KLV.
- Binary claim risk: still open. KLV remains a proof spike until small fixtures prove extraction, binary-by-reference
  storage, and memory-bounded handling.

## Accepted Risks

- The bridge is still planning evidence, not implemented SemOps CS API ingress or egress.
- The OGC resource family mapping is coarse until SemOps has stable asset, sensor, deployment, datastream,
  observation, command, and event graph fixtures.
- SemConnect currently remains the conformance anchor; rechartering SemOps to own a CS API gateway would require a
  separate product decision.

## Follow-Ups

- Build CS API ingress/egress fixtures only after structural graph state is stable enough to map one platform, hosted
  sensor, datastream, observation, deployment, system event, and command/control path.
- Keep TAK Server, SAPIENT service, ADS-B receiver, and KLV media-pipeline promotion behind product-force review.
- File SemStreams asks only when SemOps evidence shows framework-level gaps in ownership, provenance, indexing,
  spatial-temporal query, or raw-lane guidance.
