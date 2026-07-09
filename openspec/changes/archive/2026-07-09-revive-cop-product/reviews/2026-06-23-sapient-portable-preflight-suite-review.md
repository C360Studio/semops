# SAPIENT Portable Preflight Suite Review

Date: 2026-06-23
Scope: whether SemOps should create or depend on a Linux/CI-friendly SAPIENT preflight suite

## Decision

Treat a portable SAPIENT preflight suite as a valuable ecosystem contribution, not an MVP blocker and not a compliance
substitute.

SemOps already has bounded evidence for the COP MVP SAPIENT subset: JSON fixture parsing, dynamic descriptor-based
binary protobuf decode, raw replay, SemStreams HTTP input -> decoder component flow, and an opt-in absolute-location
detection graph projector. That evidence is enough to continue product development with explicit non-compliance
language.

The official Dstl BSI Flex 335 v2 Test Harness remains the compliance-facing path. A portable suite can improve
developer confidence, make Linux CI practical, and help SemStreams/SemOps catch parser and governance regressions, but
it should be labeled as developer preflight or interoperability evidence unless an accepted authority recognizes it.

## Red-Team Findings

1. A portable suite is easy to overclaim.

   The value is repeatability, not authority. Demo and proposal language must not imply that passing a SemOps or
   ecosystem preflight suite is equivalent to Dstl harness compliance.

2. Fixture provenance is the hard part.

   Official true/false corpora may carry redistribution or attribution constraints. Synthetic fixtures are useful, but
   they must be labeled as synthetic and scoped to storage/governance/parser proof rather than protocol conformance.

3. Linux CI should focus on failure modes.

   The suite should cover required-field failures, malformed binary payloads, JSON/protobuf parity, replay stability,
   and graph-write rejection before it grows broad happy-path message coverage.

4. SemStreams value should be visible.

   A good portable suite should exercise registered payloads, raw lanes, component flow, owner-token graph writes, and
   telemetry where applicable, not just standalone parser functions.

## Evidence Accepted

- The Dstl harness is documented separately as Windows-focused, .NET 6-based, and PostgreSQL 12-dependent.
- SemOps has descriptor-based binary decode and JSON preflight tests from vendored official BSI Flex 335 v2 proto
  sources.
- SemOps SAPIENT runtime component and graph projection gates remain fixture-backed and explicitly scoped.

## Follow-Ups

- Revisit a portable suite after the MVP feed set stabilizes or when SemStreams needs a cross-product regression pack.
- If created, keep it separate from official SAPIENT compliance claims and record fixture provenance per corpus.
