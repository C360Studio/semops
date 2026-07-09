# SemSource Binary Fixture Handoff Review

Date: 2026-06-22

## Decision

SemOps cannot currently provide SemSource with a real legal KLV, STANAG 4609, or SKG binary fixture. SemSource may use
a deliberately synthetic binary fixture for its SemStreams governed-graph migration, but that fixture is storage and
governance proof only.

This approves a synthetic fixture for proving binary-by-reference storage, governed metadata publication, owner-token
use, declared predicates, indexing profiles, and memory-bounded handling. It does not approve KLV/STANAG 4609,
SAPIENT, SKG, streaming-binary, parser, or protocol conformance claims.

Update: SemSource's governed SemStreams migration has aligned with this boundary. It treats opaque binary handling as
substrate evidence and leaves KLV/MISB/STANAG/SAPIENT/SKG interpretation, parser behavior, translation, and
conformance to SemOps or a SemOps-owned worker. SemOps accepts that as the right product split.

2026-06-23 qualification update: SemSource now carries the opaque synthetic binary proof this review requested. The
proof stores raw bytes by reference, publishes governed metadata for hash, size, byte range, storage reference, and
proof findings, uses trace-indexed SemStreams state, and keeps raw binary out of graph triples. SemOps accepts that as
closing the SemSource storage/governance portion of the proof spike, not as KLV parser, live media, video-service,
streaming-binary, or STANAG 4609 evidence.

## Objections Raised

- A synthetic fixture can easily become accidental compliance theater. The fixture metadata and docs must say
  `protocol_claim=none` or equivalent.
- SKG is not yet defined in the SemOps feed evidence. Until clarified, treat it as binary source-knowledge-graph
  pressure rather than a protocol with conformance meaning.
- Proving local file storage is not the same as proving production media streaming. Any store that falls back to
  full-file reads must be called out before binary-scale claims.
- A KLV/SKG product gate still needs a legal representative sample, parser strategy, metadata extraction tests, and
  separate review.

## Evidence Checked

- `openspec/changes/revive-cop-product/feed-evidence/klv-stanag4609.md` says no SemOps KLV adapter exists, SemSource is
  not a tested KLV parser, no small legal fixture has been identified, and the media path is not proven for KLV or
  streaming binary.
- `docs/feed-validation-and-indexing-ladder.md` keeps KLV/STANAG 4609 as a stretch proof spike and names SemSource as
  a candidate sidecar only.
- SemSource's governed SemStreams migration spec asks to clarify the SemOps binary/SKG fixture target before adding a
  binary-source service proof.
- SemSource now documents a `SemOps Binary Source Proof Boundary`: opaque synthetic binary storage/governance only,
  with KLV/MISB/STANAG/SAPIENT/SKG interpretation assigned to SemOps or a SemOps-owned worker.

## Accepted Risks

- SemSource can continue migration work without waiting for SemOps to find or generate a legal protocol fixture.
- The first fixture will not exercise real protocol parsing or MISB/STANAG semantics.
- The storage/governance proof may need to be replaced or supplemented when SemOps chooses the real KLV/SKG parser
  lane.

## Follow-Up Tasks

- Keep SemSource fixture metadata explicit that the fixture is synthetic and carries no protocol conformance claim.
- Identify or generate a legal tiny video-plus-KLV/SKG fixture before KLV/SKG feed acceptance.
- Design the SemOps-owned KLV/MISB worker boundary: storage-reference input, demux/parser choice, derived-fact output,
  and conformance gate.
- Re-run adversarial review before any demo, README, proposal, or UI copy uses protocol or streaming-binary language.
