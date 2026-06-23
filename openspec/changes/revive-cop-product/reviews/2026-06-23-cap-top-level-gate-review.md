# CAP Top-Level Evidence Gate Review

Date: 2026-06-23

## Decision

Close the top-level CAP XML schema, NWS sample, and lifecycle gate for MVP/demo planning.

This closure means SemOps has enough bounded CAP evidence to avoid accidental conformance or service claims:

- CAP 1.2 namespace and consumer-rule validation runs in the parser gate.
- An opt-in XSD/sample smoke exists for developer-supplied CAP 1.2 schemas, local XML samples, and replay JSONL.
- One captured NWS active-alert sample has a recorded local XSD parse smoke with hashes.
- A committed derived CAP lifecycle fixture travels with the demo and covers alert, update, cancel, and expired-alert
  replay without depending on live NWS.

This does not close CAP consumer conformance, NWS/IPAWS integration, default hosted CAP service support, or captured
provider lifecycle coverage.

## Evidence Accepted

- `pkg/adapters/cap` rejects wrong or missing CAP 1.2 namespaces and invalid CAP-shaped consumer fields before graph
  projection.
- `pkg/adapters/cap` includes `TestCAPSchemaSmokeWithLocalSamples`, skipped unless local schema/sample environment is
  supplied.
- `docs/cap-schema-smoke.md` records the 2026-06-23 NWS active-alert smoke, schema hash, sample hash, and command.
- `fixtures/cap/lifecycle/hadr-flood.jsonl` is manifest-listed derived story data and is checked against
  `LifecycleFixtureRecords`.
- CAP projection remains append-evidence and does not own authoritative hazard geometry, severity, or status.

## Claim Boundary

Allowed language:

- "CAP parser, consumer-rule, opt-in schema/sample, lifecycle replay, graph projection, and COP readback evidence."
- "One recorded local NWS active-alert XSD smoke."
- "Portable derived CAP lifecycle replay fixture."

Rejected language:

- "CAP consumer conformant."
- "NWS/IPAWS integrated."
- "Default live CAP service."
- "Captured NWS lifecycle corpus."
- "Authoritative hazard lifecycle ownership."

## Remaining Product Work

- Capture and review real NWS/IPAWS/vendor update, cancel, and expired-alert samples.
- Decide provider stale-source policy from captured provider behavior, not only deterministic local HTTP tests.
- Add provider lifecycle evidence before enabling CAP by default in hosted stacks.
- Add a separate CAP consumer-profile or interoperability gate before conformance wording.
