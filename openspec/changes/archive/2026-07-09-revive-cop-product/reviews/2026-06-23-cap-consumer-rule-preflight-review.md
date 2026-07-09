# CAP Consumer Rule Preflight Review

Date: 2026-06-23
Scope: CAP 1.2 namespace and consumer-rule validation before XML schema or NWS sample claims

## Decision

Accept CAP namespace and consumer-rule validation as a useful standards-evidence step, but do not treat it as formal
CAP conformance.

The adapter now rejects wrong or missing CAP 1.2 namespaces and validates the CAP-shaped fields SemOps currently
depends on: top-level `status`, `msgType`, and `scope`; required `info` category/event/urgency/severity/certainty;
`effective`/`expires` ordering; and required `areaDesc` for area evidence. This keeps malformed or non-CAP-shaped
alerts from reaching graph projection.

## Red-Team Findings

1. This is not full XSD validation.

   Go's standard XML decoder is still the parser boundary, and SemOps is not running the official CAP 1.2 schema. The
   claim must remain namespace/consumer-rule preflight until a formal XSD validation path exists.

2. This is not NWS provider evidence.

   The current fixtures are deterministic NWS-shaped HA/DR samples. A captured NWS sample set for alert/update/cancel
   and provider stale behavior is still required before claiming NWS integration evidence.

3. CAP should remain append-evidence.

   Stronger parser validation does not change ownership semantics. CAP still appends hazard/advisory evidence and
   does not own authoritative hazard geometry, severity, or status predicates.

4. Consumer rules should follow product scope.

   The validator should stay aligned with fields SemOps projects or displays. A broad CAP conformance engine belongs
   behind a separate schema/consumer-profile gate.

## Evidence Accepted

- `pkg/adapters/cap.Parse` rejects missing or wrong CAP 1.2 namespaces.
- `Alert.Validate`, `Info.Validate`, and `Area.Validate` reject invalid CAP 1.2 enum-shaped fields and missing required
  fields before projection.
- `go test ./pkg/adapters/cap` covers malformed XML, missing identifiers, namespace failures, invalid enums, invalid
  `expires` ordering, missing `areaDesc`, invalid polygons, and invalid circles.

## Follow-Ups

- Add formal CAP 1.2 XSD validation if SemOps needs schema-backed CAP conformance wording.
- Capture real NWS CAP samples and replay update/cancel/expire behavior before claiming NWS integration evidence.
