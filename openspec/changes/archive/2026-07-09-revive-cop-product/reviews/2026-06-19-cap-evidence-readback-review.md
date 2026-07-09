# CAP Evidence Readback Review

Date: 2026-06-19

## Decision

Accept the CAP/EDXL slice into the structural COP as a parser, born-first append-evidence projection, graph writer, and
COP hazard readback gate.

Do not claim hosted CAP polling, NWS/IPAWS integration, EDXL coverage beyond CAP, update/cancel/expire lifecycle, or
CAP consumer conformance yet.

## Objections Raised

- CAP evidence could accidentally look like authoritative hazard state if the COP renders it as a polygon.
- The projection could drift back toward old auto-vivify habits because CAP has no enforceable write-fence claim.
- `content` indexing on hazard evidence is correct for advisory text, but future hazard lifecycle state will need a
  separate `control` projection.
- Fixture-only CAP parsing is not compliance evidence. Schema and consumer-rule validation are still missing.
- A default configured alert identifier is useful for demo readback but not a scalable discovery model.

## Evidence Checked

- `pkg/adapters/cap` parses required CAP fields, polygons, circles, geocodes, resources, and rejects malformed or
  incomplete alerts.
- `internal/projectors/cap` creates `hazard_area` entities before updates and writes only the CAP append-evidence
  predicate set.
- `internal/projectors/cap` graph writer uses SemStreams create/update mutation request subjects and stops on mutation
  failures.
- `internal/api/cop` maps `cop.hazard.evidence` JSON into the COP hazard view model with CAP provenance and feed
  health.
- `pkg/cop` contract tests still reject CAP ownership of `cop.hazard.geometry`, `cop.hazard.severity`, and
  `cop.hazard.status`.

## Accepted Risks

- CAP feed health is seeded by configured alert IDs rather than index-backed discovery.
- CAP circles are approximated into polygon rings for the current map layer.
- Expiry and cancel semantics are parsed but not yet enforced as COP stale/inactive lifecycle behavior.
- The current graph warning for append-evidence-only owners is expected until SemStreams fully separates evidence
  declarations from enforceable write-fence claims.

## Follow-Up Tasks

- Capture NWS and OASIS examples as deterministic CAP fixtures.
- Add XML schema and consumer-rule validation before conformance language appears in demos.
- Add update/cancel/expire lifecycle behavior and stale readback tests.
- Replace seed alert-ID readback with index-backed CAP discovery once SemStreams query support is ready.
- Decide whether authoritative hazard lifecycle state belongs in a CAP-owned control projection or a fusion-owned
  derived projection.
