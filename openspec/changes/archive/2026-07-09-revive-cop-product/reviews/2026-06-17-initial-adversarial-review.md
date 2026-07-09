# Initial Adversarial Review

Date: 2026-06-17

Change: `revive-cop-product`

Status: complete for the planning baseline. Future stage reviews remain required.

## Decision

Proceed with framework modernization, feed evidence inventory, and MAVLink salvage planning.

Do not proceed with broad Phase 1 adapter implementation, orchestration UI, SAPIENT product work, KLV product
claims, or upstream SemStreams issues until the evidence gates in this review are satisfied.

## Review Roles

- Program Manager: check delivery order, blockers, and demo credibility.
- Architect: challenge product/framework ownership, entity boundaries, and service boundaries.
- Go reviewer: challenge module/toolchain migration, adapter contracts, concurrency, and raw-lane behavior.
- Svelte reviewer: challenge COP-first UI scope and topology/orchestration promotion.
- Technical writer: challenge compliance language, standards claims, and review-record traceability.

## Evidence Checked

- SemOps COP architecture baseline.
- Feed validation and indexing ladder.
- OpenSpec requirements for product ownership, governed feed fusion, feed validation, orchestration scope, container
  infrastructure, and adversarial review gates.
- Ticket chain `COP-001` through `COP-008`.
- Local SemOps MAVLink parser, generator, tests, and SITL code identified during the planning pass.
- Local SemLink TAK seeding path identified during the planning pass.
- Local SemSource media/video support identified during the planning pass.
- Public evidence for MAVLink, PX4 SITL, ArduPilot SITL, CAP, NWS alerts, OpenSky, jMISB, and `klvdata`.

## Objections

### 1. Phase 1 could still become too broad

MAVLink, TAK/CoT, and CAP are the right first feeds, but even those can sprawl if each adapter grows full protocol
coverage before a governed projection contract exists.

Resolution: keep Phase 1 feed slices narrow. Each adapter starts with parser, replay, current-state projection,
source/provenance view, and stale-data behavior.

Follow-up: `COP-007`, `COP-003`, `COP-004`.

### 2. Index profile pressure is real, but new framework semantics are premature

The current SemStreams profiles may be enough if SemOps splits high-rate state, durable operational state, textual
content, and replay traces into separate entities. A new profile or field-level policy request would be premature
before mixed-feed fixtures prove the need.

Resolution: require profile/cardinality decisions per projected entity type and only file upstream asks after at
least two feed fixtures show clean boundaries are insufficient.

Follow-up: `COP-007`, `COP-006`.

### 3. SAPIENT evidence is not ready

No public SAPIENT compliance suite, authoritative protobuf, ICD, validator, or sample fixture was verified.

Resolution: SAPIENT remains evidence-gated. Do not design around guessed schemas.

Follow-up: `COP-007`, `COP-008`.

### 4. KLV is the highest honesty risk

SemSource can help with media metadata and keyframes, and external KLV libraries exist, but the current SemSource
video path still needs proof for KLV extraction, storage-by-reference, and memory-bounded binary handling.

Resolution: KLV remains a proof spike. No "streaming binary" claim until a small fixture proves the path.

Follow-up: `COP-007`, `COP-008`.

### 5. TAK/CoT should not be described as conformant

The local TAK seed path is useful for mocks and replay, but no public TAK/CoT compliance suite was verified.

Resolution: describe TAK as fixture/replay/interoperability-tested until a real conformance surface is found.

Follow-up: `COP-007`, `COP-004`.

### 6. Orchestration can still sneak back in through deployment metadata

Even with scope gates, topology/tier metadata can become an attractive but low-value UI feature.

Resolution: deployment metadata can exist for operations and monitoring, but topology, tier placement, and
orchestration UI require a dedicated operator-value review before promotion.

Follow-up: `COP-005`, `COP-008`.

## Accepted Risks

- The repo is currently stale enough that `go test ./...` is not a useful first gate until the SemStreams module path
  and Go toolchain are modernized.
- MAVLink and TAK first will bias the demo toward moving tracks and operator markers, so CAP must remain in Phase 1
  to prove loose civilian warning ingestion early.
- CS API egress is intentionally not on the critical path for Phase 1, even though it has the strongest formal
  conformance harness, because SemOps needs governed graph state before egress is meaningful.

## Required Next Actions

1. Complete `COP-001`: current SemStreams module path and Go toolchain modernization.
2. Complete `COP-002`: canonical COP model and predicate ownership matrix.
3. Continue `COP-007`: concrete feed evidence records for MAVLink, TAK/CoT, and CAP.
4. Start `COP-003` only after MAVLink current-state projection tests are written against real frames.
5. Keep `COP-004` blocked until at least two Phase 1 feeds have parser, replay, projection, and indexing evidence.

## Review Outcome

Planning baseline passes with constraints.

The next allowed work is narrow: framework modernization, COP model governance, and feed evidence records. The
review does not approve broad adapter implementation, orchestration UI, SAPIENT implementation, KLV claims, or
upstream SemStreams issues.
