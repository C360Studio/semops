# SAPIENT Generated Bindings Decision Review

Date: 2026-06-23
Scope: whether SemOps needs generated Go bindings for Dstl BSI Flex 335 v2 protobufs after adding dynamic descriptor
binary preflight support

## Decision

Do not add generated SAPIENT Go bindings for the current SemOps COP MVP path.

The existing descriptor-based path compiles the vendored Dstl BSI Flex 335 v2 proto sources with
`github.com/bufbuild/protocompile`, decodes binary `SapientMessage` payloads with dynamic protobuf descriptors, and
validates the result through the same preflight model used by JSON fixtures. That is enough for parser preflight, raw
replay, SemStreams HTTP input -> decoder component flow, and the reviewed absolute-location detection graph projector.

Generated bindings should be reopened only when SemOps needs at least one of:

- Product service mode that must author or transform a broad SAPIENT message surface.
- Outbound SAPIENT tasking or command acknowledgements where typed construction semantics matter.
- Exact typed protobuf round-trip behavior beyond decode-to-preflight validation.
- Broad fixture corpus coverage where dynamic field handling becomes more fragile than generated code.
- Measured performance pressure from dynamic descriptors.

If generated bindings become necessary, use a reproducible buf-based workflow, not ad hoc `protoc` invocation. Record
the Dstl proto source commit, generator versions, generated package paths, and how v1/v2 drift is surfaced.

## Red-Team Findings

1. Generated bindings would add false maturity today.

   Generated structs can make the product look like it supports the full SAPIENT surface even though SemOps currently
   supports a bounded preflight subset and one reviewed projection contract.

2. Dynamic descriptors preserve the version boundary with less churn.

   The vendored official proto layout remains the source of truth. The current tests already prove the descriptor can
   decode representative binary payloads and route them through the same validation path as JSON fixtures.

3. Tasking changes the calculus.

   If SemOps starts creating SAPIENT tasks or service responses, typed generated messages may become safer than
   dynamic construction. That should happen alongside command authority, priority, freshness, and harness review.

4. Generation is a supply-chain decision.

   If generated code enters the repo, the generator and source commit are part of the compliance evidence. Without
   that record, generated files become stale noise.

## Evidence Accepted

- `pkg/adapters/sapient/protobuf.go` embeds and compiles Dstl BSI Flex 335 v2 proto sources with `protocompile`.
- `ParseBinaryMessage` decodes dynamic protobuf payloads into the same validated preflight model as JSON.
- SAPIENT runtime component and graph projection tests pass without generated bindings.
- The current SAPIENT projection scope is absolute-location detection reports only.

## Follow-Ups

- Keep generated binding work deferred until product service mode, outbound tasking, broader protobuf round-trip
  semantics, or performance profiling creates a concrete need.
- Pair any future generated-binding implementation with Dstl harness scope and command-authority review.
