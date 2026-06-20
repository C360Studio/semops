# SAPIENT Artifact Discovery Review

Date: 2026-06-20

## Decision

Accept that authoritative public SAPIENT artifacts exist and are strong enough to plan a SemOps SAPIENT lane:

- GOV.UK names SAPIENT as a Dstl/MOD-owned architecture and points to BSI Flex 335, protobuf files, the SAPIENT Test
  Harness, and SAPIENT Middleware.
- BSI Flex 335 V2:2024 is the current public message structure, format, and content anchor.
- Dstl publishes BSI Flex 335 v2 protobuf definitions, a v2 Test Harness, sample true/false message corpora, and Apex
  middleware.

Do not start SAPIENT graph projection or claim SAPIENT compliance yet. The next approved step is parser-only BSI Flex
335 v2 fixture work, followed by a scoped Dstl harness run or a documented non-compliance demo decision.

## Objections Raised

- A public official harness is not the same thing as an automation-friendly Linux CI gate. The Dstl v2 harness README
  states Windows-only system support, .NET 6 SDK, PostgreSQL 12, and manual configuration.
- A portable SemOps preflight suite would be valuable, but it would not replace official compliance evidence unless
  Dstl, BSI, or another accepted authority recognizes it.
- Apex middleware routes, validates, archives, replays, and exposes API state, but SemOps must still own COP product
  semantics: ownership, provenance, freshness, command authority, indexing, and source-health behavior.
- BSI Flex is iterative and may change, so generated protobuf bindings must make version drift visible.
- SAPIENT tasking and alert acknowledgement are control-plane behaviors; treating them like ordinary detection state
  would create unsafe write authority and stale-command risks.

## Evidence Checked

- GOV.UK SAPIENT guidance, last updated 2024-09-06.
- BSI Flex 335 V2:2024 public description page.
- `dstl/SAPIENT-Proto-Files` with `bsi_flex_335_v1_0`, `bsi_flex_335_v2_0`, and Apache-2.0 license text except where
  noted.
- `dstl/BSI-Flex-335-v2-Test-Harness` README, license, sample message corpus, validators, and solution tree.
- `dstl/Apex-SAPIENT-Middleware` README, configuration shape, protobuf validation options, replay/API behavior, and
  Apache-2.0 license language.
- `dstl/SAPIENT-Middleware-and-Test-Harness` README, which marks that older combined repository obsolete and points to
  the split v2 harness and Apex middleware.

## Accepted Risks

- SemOps cannot run the official harness in normal Linux CI without a Windows/VM/manual path or additional work.
- Initial parser fixtures may prove message handling without proving SAPIENT component compliance.
- Official sample JSON corpora may have redistribution or attribution details that need review before vendoring.
- A future portable preflight suite may become useful enough that it deserves its own repository or cross-product home.

## Follow-Up Tasks

- Add parser-only BSI Flex 335 v2 protobuf fixtures before any graph projection work.
- Run or qualify the Dstl v2 Test Harness before SAPIENT compliance claims.
- Evaluate a portable Linux/CI-friendly SAPIENT preflight suite as an ecosystem contribution.
- Keep Apex middleware as an interop/service-shape reference, not as the owner of SemOps product semantics.
