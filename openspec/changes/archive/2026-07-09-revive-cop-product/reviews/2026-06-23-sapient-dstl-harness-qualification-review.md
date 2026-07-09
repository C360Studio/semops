# SAPIENT Dstl Harness Qualification Review

Date: 2026-06-23

## Decision

Close task `5.27` as a qualification, not as a harness pass.

SemOps will continue the MVP SAPIENT lane as explicit non-compliance developer evidence: JSON/protobuf parser
preflight, bounded raw replay, SemStreams input/decoder component flow, and opt-in absolute-location graph projection.
The official Dstl BSI Flex 335 v2 Test Harness remains the compliance-facing path and must be run, or replaced by an
accepted authority result, before SemOps uses SAPIENT compliance or product-support language.

## Evidence Checked

- GOV.UK identifies SAPIENT as a Dstl/MOD-owned architecture and links BSI Flex 335, Dstl protobuf files, the SAPIENT
  Test Harness, and SAPIENT Middleware.
- GOV.UK says the SAPIENT Test Harness is used during development and testing to evaluate message compliance with BSI
  Flex 335 v2.
- `dstl/SAPIENT-Proto-Files` publishes `bsi_flex_335_v2_0` protobuf sources and describes the BSI Flex 335 interface
  scope.
- `dstl/BSI-Flex-335-v2-Test-Harness` describes the harness as version `5.2.4`, BSI Flex 335 v2 only, Windows-only
  for supported operation, .NET 6-based, PostgreSQL 12-dependent, manually configured, Apache-2.0 except where noted,
  and supplied as-is without support.
- SemOps already has developer evidence for the MVP subset: representative JSON fixtures, dynamic descriptor-based
  protobuf decode from vendored Dstl proto sources, raw replay, SemStreams component-flow preflight, and
  `SEMOPS_SAPIENT_GRAPH_ENABLED=true` absolute-location graph projection.

## Objections Raised

1. A qualification can be mistaken for a pass.

   The task text and docs must say the harness was not run for MVP. No demo, README, roadmap, or proposal language may
   say or imply local harness success.

2. The Dstl harness is real but operationally awkward for normal CI.

   Its Windows/.NET/PostgreSQL 12/manual-configuration shape makes it useful for a scoped validation event, not as the
   default Linux developer gate.

3. A portable suite remains valuable but not authoritative.

   A SemOps or ecosystem preflight suite can catch parser, replay, component, and governance regressions. It should
   stay labeled developer evidence unless Dstl, BSI, or another accepted authority recognizes it.

4. Graph projection can make preflight look like product support.

   The current graph path is explicitly opt-in and limited to absolute-location detection reports. It must not be
   treated as service hosting, tasking, lifecycle, association, UTM, range/bearing, or middleware compatibility.

## Claim Boundary

Allowed language:

- SemOps has SAPIENT developer preflight for representative BSI Flex 335 v2 JSON and protobuf payloads.
- SemOps has opt-in fixture-backed SAPIENT absolute-location detection projection through SemStreams components.
- The Dstl BSI Flex 335 v2 Test Harness is the current compliance-facing validation path.

Blocked language:

- SemOps is SAPIENT compliant.
- SemOps has passed the Dstl harness.
- SemOps provides SAPIENT product support or a hosted SAPIENT service.
- The SemOps portable fixtures or future portable preflight suite are official compliance evidence.

## Future Harness Gate

A future compliance-facing SAPIENT task must record:

- Dstl harness repository/release or accepted authority source.
- Harness operating environment, including OS, .NET SDK, PostgreSQL, and locale.
- Corpus and message-role scope.
- SemOps component mode under test: parser-only, input/decoder, graph projection, service mode, or tasking.
- Pass/fail evidence, limitations, and known issues.
- Claim-language review before release notes, demo copy, or proposal text changes.

## Verification

- `go test ./pkg/adapters/sapient ./internal/components/sapient ./internal/projectors/sapient ./internal/fixturemanifest`
- `openspec validate revive-cop-product --strict`
- `git diff --check`
