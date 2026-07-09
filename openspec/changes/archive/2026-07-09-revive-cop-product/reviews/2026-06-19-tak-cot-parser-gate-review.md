# TAK/CoT Parser Gate Adversarial Review

Date: 2026-06-19

## Decision

Accept `pkg/adapters/cot` as the SemOps-local native parser gate for TAK/CoT fixture work. Do not promote TAK/CoT to a
structural feed adapter until SemOps owns transport replay, projection, graph writes, and stale-data behavior.

## Evidence Checked

- `pkg/adapters/cot/cot.go` decodes and encodes a minimal Cursor on Target XML subset with no graph dependencies.
- `pkg/adapters/cot/cot_test.go` covers SemLink-style ALPHA/BRAVO operator seed shapes, a North Gate marker,
  GeoChat remarks fallback, air-track marshal/unmarshal, type classifiers, and malformed input rejection.
- `go test ./pkg/adapters/cot` passes.
- `go test ./...` passes.

## Objections

- This is not TAK Server behavior, federation, auth, mission package support, tasking, or full TAK interoperability.
- No public TAK/CoT compliance suite has been verified, so claim language must remain fixture/replay/interoperability
  tested.
- Parser success does not prove SemStreams ownership, born-first source assets, indexing profile assignment, or COP UI
  feed state.
- GeoChat currently proves text extraction only. It does not decide whether chat becomes content evidence, task state,
  or an operator-message entity.
- Stale times are parsed, but no stale-data policy or runtime freshness downgrade exists for TAK yet.

## Accepted Risks

- Port a clean-room subset from SemLink as prior art because SemOps needs product-owned feed depth and SemLink remains
  a basic demo.
- Keep malformed XML rejection in the native parser gate before transport listeners can flood logs or graph writers.
- Defer entity ID derivation and source-asset birth discipline until the projection gate, where ownership semantics can
  be tested directly.

## Follow-Up Tasks

- Add SemOps-owned UDP/TCP fixture replay for ALPHA/BRAVO operators, marker, GeoChat, malformed XML, duplicate UID, and
  stale event cases.
- Add a TAK projector that births source assets first, writes source-partitioned track entities, and keeps GeoChat/text
  out of high-rate signal entities.
- Add graph smoke and COP UI evidence before marking TAK as a structural stack feed.
