# SAPIENT Binary Descriptor Review

Date: 2026-06-20

## Decision

Accept descriptor-based binary protobuf preflight for the SAPIENT planning lane.

`pkg/adapters/sapient` now embeds the official Dstl BSI Flex 335 v2 proto source files, compiles
`sapient_msg/bsi_flex_335_v2_0/sapient_message.proto` with `github.com/bufbuild/protocompile`, decodes binary
`SapientMessage` payloads through dynamic protobuf descriptors, converts them through protobuf JSON, and validates the
result through the same SemOps preflight model used for JSON fixtures.

This approves binary protobuf preflight for the representative subset only. It does not approve Dstl harness
compliance, full SAPIENT message coverage, graph projection, generated Go bindings, hosted service work, or product
support claims.

## Objections Raised

- Dynamic descriptors are a good fit for avoiding a global `protoc`, but they still need pinned source provenance and
  license visibility. The proto source is therefore vendored with Dstl license text.
- The supported subset is still registration, status report, detection report, and task acknowledgement. Alert,
  alert acknowledgement, task, registration acknowledgement, and error remain unsupported.
- `protojson` is used as the bridge from dynamic messages into the existing preflight model. That reduces duplicate
  mapping code, but it means JSON/protobuf agreement tests must remain in place.
- Generated Go bindings may still be useful later for performance, static typing, or service boundaries. This review
  does not reject generated bindings; it simply removes them as a prerequisite for binary preflight.
- A decoded task acknowledgement is still control-plane evidence. It must not create command authority or graph writes
  without a separate ownership/indexing review.

## Evidence Checked

- Vendored Dstl v2 proto source files under `pkg/adapters/sapient/protos/sapient_msg`.
- Dstl Apache-2.0 license text under `pkg/adapters/sapient/protos/sapient_msg/LICENCE.txt`.
- `pkg/adapters/sapient/protobuf.go` descriptor compile, embedded resolver, dynamic protobuf decode, and preflight
  validation bridge.
- `pkg/adapters/sapient/protobuf_test.go` descriptor compile check, binary payload decode for registration/status/
  detection/task-ack, and binary missing-`reportId` rejection.
- `go test ./pkg/adapters/sapient -count=1` passes.

## Accepted Risks

- The proto source is now vendored and must be intentionally refreshed when Dstl updates BSI Flex 335.
- Dynamic descriptor compile adds a runtime dependency on `protocompile`; descriptor caching keeps normal use bounded.
- The current binary fixtures are generated from representative JSON shapes in tests rather than captured live SAPIENT
  traffic.

## Follow-Up Tasks

- Run or qualify the Dstl BSI Flex 335 v2 Test Harness before compliance language.
- Decide whether generated Go bindings are needed beyond dynamic descriptors.
- Add hosted adapter and graph projection only after ownership, indexing, command-authority, and source-health review.
- Evaluate whether a portable Linux/CI SAPIENT preflight suite should live in SemOps or a separate ecosystem tool.
