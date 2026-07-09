# CAP Schema And Sample Smoke Review

Date: 2026-06-23
Scope: Opt-in CAP 1.2 XSD validation and captured-sample replay evidence

## Decision

Accept an opt-in CAP schema/sample smoke as the next standards-evidence hook. The smoke requires a developer-supplied
CAP 1.2 XSD, local CAP XML files or SemOps replay JSONL records, and `xmllint`. It validates each local sample against
the supplied schema and then parses it through the SemOps CAP adapter.

This is not CAP consumer conformance and does not clear NWS/IPAWS/vendor integration claims.

## Findings

1. Do not vendor schema or public alert captures by accident.

   Official schema copies and captured NWS/IPAWS/vendor alerts may have provenance, licensing, or operational context
   concerns. Keep local schema and sample directories ignored until a separate fixture review clears what may be
   committed.

2. XSD validation is necessary but not sufficient.

   A CAP document can be schema-valid while still violating product expectations, provider lifecycle semantics, stale
   data policy, or projection ownership boundaries. The SemOps parser and projector gates still need to run.

3. Live NWS is not deterministic CI.

   The hosted CAP poller can capture provider-shaped replay records, but default CI should consume local fixtures or
   skip the smoke. Live NWS remains opt-in and should preserve user-agent, cache, and rate-limit behavior.

4. CAP remains append-evidence.

   Stronger sample/schema evidence does not turn CAP into authoritative hazard truth. CAP continues to append hazard
   and advisory evidence unless a later hazard-state model earns ownership.

## Follow-Ups

- Capture a small NWS alert/update/cancel or expired-alert sample set with provenance and hashes.
- Record a passing local schema/sample smoke before using CAP schema evidence in demo materials.
- Add a separate consumer-profile or interoperability gate before claiming CAP consumer conformance.
