# COP Model Governance Review

Date: 2026-06-17

Status: accepted stage gate for `COP-002`

Roles:

- Architect: challenge entity boundaries, ownership modes, and source partitioning.
- Go reviewer: challenge contract validation, overlap tests, and SemStreams API usage.
- Svelte reviewer: challenge whether the model supports operator provenance views without topology creep.
- Technical writer: challenge whether predicates and upstream candidates are named without overclaiming maturity.

## Decision

Accept the first COP model as a test-backed baseline, not a final ontology.

The baseline defines seven canonical entities and five first-phase owned contracts: source asset identity, MAVLink
track state, TAK/CoT track state, CAP hazard evidence, and deterministic fusion alerts. Strict feed owners are
source-partitioned by the SemStreams entity `system` segment. CAP remains append-only evidence. Fusion owns derived
alert state separately from source facts.

SemOps also accepts SemStreams' breaking-change direction now: no auto-vivify, ADR-055 born-first entity creation,
and ADR-056 projection contracts that derive explicit `ForeignEdgeClaim` values for cross-entity relationships.

## Evidence Checked

- `pkg/cop/contracts.go`
- `pkg/cop/contracts_test.go`
- `internal/contracts/semstreams_contract_test.go`
- `docs/cop-model-and-governance.md`
- `openspec/changes/revive-cop-product/specs/governed-feed-fusion/spec.md`
- `go test ./...`

## Adversarial Objections

### Objection 1: Strict feeds could silently collide on track predicates

If MAVLink and TAK both owned `cop.track.position` over `c360.*.cop.*.track.*`, SemStreams ownership would correctly
reject the overlap. That rejection is useful, but a product model that starts with guaranteed collision is bad design.

Resolution: strict feed contracts now use source-partitioned entity patterns such as `c360.*.cop.mavlink.track.*`
and `c360.*.cop.tak.track.*`.

### Objection 2: CAP evidence on `hazard_area` may blur advisory text and hazard state

CAP can carry useful warning text before SemOps has a deterministic hazard geometry. If CAP owns geometry, severity,
or status too early, it can overwrite better source or fusion state.

Resolution: CAP starts with `append-evidence` only and explicitly avoids `cop.hazard.geometry`,
`cop.hazard.severity`, and `cop.hazard.status`.

### Objection 3: Provenance and confidence look framework-shaped

`cop.provenance.source`, `cop.provenance.confidence`, and `cop.provenance.observed_at` are probably not SemOps-only
forever. Moving them upstream too early would freeze names before real feeds prove the shape.

Resolution: keep them product-local for Phase 1 and list them as upstream candidates only after failing SemOps tests
or duplicated adapter code prove the reusable need.

### Objection 4: Alert affected entity may become multi-valued

`cop.alert.affected_entity` is currently fusion-owned current state. A single alert may later affect several tracks,
assets, or hazard areas.

Resolution: accept the first contract for simple deterministic alerts. Revisit mode or representation when the first
scenario creates multi-entity alerts.

### Objection 5: Spatial values are not type-checked yet

The current contract names position and geometry predicates, but it does not yet prove WKT, GeoJSON, CRS, or temporal
window semantics.

Resolution: defer spatial typing to the first MAVLink/TAK/CAP projector tests. Do not file a SemStreams spatial
helper ask until those tests expose concrete friction.

### Objection 6: Rebuild could accidentally recreate auto-vivify dependencies

The old MAVLink battery and payload graphing paths relied on implicit target creation for `HAS`/`POWERED` style
edges. Because SemOps is mid-revival, this should be broken now rather than migrated.

Resolution: deleted the old active battery/rule/payload paths from the worktree and added born-first requirements.
New adapters must birth target entities before update/edge writes, and any cross-entity write must derive a
SemStreams `ForeignEdgeClaim` from the projection contract.

## Follow-Ups

1. Add real projector tests that create `graph.CreateEntityWithTriplesRequest` values from MAVLink and TAK fixtures.
2. Decide whether CAP hazard evidence should project to `hazard_area`, `advisory`, or both after sample fixtures are
   parsed.
3. Revisit provenance/confidence as an upstream SemStreams ask only after two feed adapters duplicate the pattern.
4. Add spatial value fixtures before claiming map-ready geometry.
5. Add adapter tests that prove source assets are born before `cop.track.source` edges are written.

## Result

`COP-002` can move to review. The next allowed implementation work is MAVLink salvage behind the new `pkg/cop`
contract boundary, with TAK/CAP remaining evidence-gated until their parser/replay fixtures are in place.
