# Framework Modernization Impact Review

Date: 2026-06-17

Status: accepted stage gate for `COP-001`

Roles:

- Architect: challenge framework ownership, API contracts, and migration order.
- Go reviewer: challenge compile realism, stale imports, concurrency/lifecycle assumptions, and test gates.
- Svelte reviewer: challenge whether backend migration creates accidental UI/orchestration commitments.
- Technical writer: challenge whether the migration story is explicit enough to prevent false demo claims.

## Decision

Proceed with SemOps modernization, but do not perform a bulk import rename.

The first code slice must prove the current SemStreams projection, ownership, graph mutation, and component surfaces
from SemOps with a narrow compile-time contract test. Old StreamKit/BaseProcessor wiring must be quarantined or
retired before it is reconnected to the COP product path.

SemOps should treat the existing MAVLink codec, generator, and SITL scaffolding as salvageable domain code. It should
not treat the old processor, entity store, ObjectStore tests, or flow configuration as an architecture to preserve.

## Evidence Checked

- `go.mod` declares `module github.com/c360/semops`, Go `1.25.3`, and a local replace for
  `github.com/c360/semstreams => ../semstreams`.
- Adjacent SemStreams declares `module github.com/c360studio/semstreams` and Go `1.26.3`.
- Local Go is `go1.25.3`, so a full current SemStreams build is blocked until the toolchain is updated.
- SemOps imports stale `github.com/c360/semstreams` packages across entity, MAVLink, rule, payload, and migrated
  test packages.
- SemOps imports `github.com/c360/streamkit` in the MAVLink processor, payload graphing code, metrics/error tests,
  and migrated ObjectStore tests. No adjacent `streamkit` module is present in `/Users/coby/Code/c360`.
- Current SemStreams exposes `component.LifecycleComponent`, `pkg/projection.Contract`,
  `pkg/ownership.Registration`, `graph.CreateEntityWithTriplesRequest`, `graph.UpdateEntityWithTriplesRequest`,
  and `message.Triple` as the modern target surface.
- Current SemStreams no longer exposes the old `pkg/interfaces/store`, `NewNATSEntityStore`,
  `processor/base`, or old robotics payload package shape that SemOps expects.
- `pkg/processors/mavlink/processor.go.backup` still contains an older BaseProcessor path and should not be used as
  migration guidance.
- The current checkout has no UI tree, so backend modernization should not recreate flow-runtime control surfaces.

## Adversarial Objections

### Objection 1: Bulk rename would look productive but create false progress

Changing every `github.com/c360/semstreams` import to `github.com/c360studio/semstreams` will not compile because
the package shapes changed. The missing store, processor, and robotics package paths are contract changes, not
spelling changes.

Follow-up: make the first test import only stable modern surfaces: `graph`, `message`, `pkg/projection`,
`pkg/ownership`, and `component`.

### Objection 2: StreamKit assumptions are architectural debt, not a dependency gap

The old processor and payload graphing path assumes BaseProcessor, component metadata, metrics, errors, and
ObjectStore behaviors that are not the current SemStreams contract. Recreating a local StreamKit compatibility layer
would drag the old flow-runtime model back into the product.

Follow-up: quarantine legacy processor and migrated tests before the new COP projection path is wired.

### Objection 3: The Go toolchain blocks honest compile claims

SemStreams requires Go `1.26.3`; local Go is `1.25.3`. Until the toolchain is updated, SemOps cannot honestly claim
that `go test ./...` proves compatibility with the current framework checkout.

Follow-up: update the toolchain or record the compile gate as blocked. Do not mark `COP-001` complete while the
toolchain mismatch remains.

### Objection 4: Module path modernization needs a repo-level decision

The SemOps remote is under `C360Studio/semops`, but `go.mod` still declares `github.com/c360/semops`. Moving to
`github.com/c360studio/semops` is probably correct for consistency, but it changes any internal import paths and
future downstream references.

Follow-up: include the SemOps module path change in `COP-001`, with a focused diff and no unrelated package moves.

### Objection 5: Current-state graph projection must come before more feeds

MAVLink parser depth is valuable, but adding TAK, CAP, ADS-B, SAPIENT, or KLV before one modern projection contract
passes will multiply stale patterns. The first structural feed should prove raw-lane by reference plus current-state
entity writes before another adapter enters the stack.

Follow-up: keep feed evidence records informational until `COP-001` and the first `COP-002` contract tests pass.

### Objection 6: UI scope must not backdoor orchestration

Backend modernization can accidentally revive old flow topology, tier toggles, or runtime control panels because the
old README and config emphasize flows. That would make the UI answer framework questions before it answers operator
questions.

Follow-up: backend APIs should expose COP snapshots, provenance, source state, health, and scenario status first.
Topology and orchestration APIs require a later UX review.

## Accepted Risks

- Some legacy packages may be deleted or moved before their full salvage value is known. This is acceptable because
  the repository is explicitly greenfield and the salvageable MAVLink codec/tests can be preserved separately.
- The first compile gate may be small and feel unsatisfying. This is acceptable because its job is to prove the
  modern SemStreams contract shape before rewiring high-risk packages.
- SemOps may need to file SemStreams asks around projection/indexing only after the first current-state entity path
  exposes concrete friction. Narrative pressure is not enough.

## Next Allowed Code Slice

1. Update SemOps module metadata to the modern SemOps and SemStreams module paths.
2. Align the Go toolchain with SemStreams, or leave a blocked compile note if the local toolchain cannot be updated.
3. Add a narrow contract test that imports and validates a SemOps-owned projection contract using current SemStreams
   `projection`, `ownership`, `graph`, `message`, and `component` packages.
4. Quarantine legacy StreamKit/BaseProcessor packages and migrated ObjectStore tests from the product compile path.
5. Only then start MAVLink codec salvage into a modern package boundary.

## Result

`COP-001` remains open. Task `2.5` can be marked complete because the migration blast radius has been reviewed, but
implementation must stay narrow until the compile gate is real.
