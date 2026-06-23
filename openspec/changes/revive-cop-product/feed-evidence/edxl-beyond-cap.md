# EDXL Beyond CAP Feed Evidence

Status: later feed-validation gate, not Phase 1 scope.

## Decision

CAP is the only EDXL-family format admitted to the Phase 1 COP feed ladder. Broader EDXL formats must not inherit
CAP parser, projection, or service evidence by association.

SemOps may promote a broader EDXL family only when a concrete product need selects the format and a new gate records
its parser, fixture, projection, ownership, indexing, lifecycle, and compliance evidence.

## Deferred Families

The current revival does not implement or claim support for:

- EDXL-DE distribution envelopes.
- EDXL-HAVE hospital availability exchange.
- EDXL-RM resource messaging.
- Any broader emergency-management payload merely because CAP support exists.

## Promotion Triggers

A broader EDXL gate requires at least one product force:

- A HADR partner needs hospital, shelter, resource, or logistics state that is actually represented by a specific EDXL
  family.
- A standards-facing proposal requires explicit EDXL-family interoperability beyond CAP alerts.
- A deployed integration source emits a concrete EDXL payload with a legal fixture and operational owner.

## Required Evidence

Before any broader EDXL family enters the structural stack, SemOps must record:

- Native schema or parser evidence for the selected EDXL family.
- Legal deterministic fixtures or captured provider samples.
- SemStreams component ports, payload registry entries, and lifecycle behavior.
- Governed projection contracts with explicit ownership mode, indexing profile, freshness, and provenance.
- Whether the feed is append-evidence, source-owned current state, durable control state, or an interop envelope.
- Replay and stale-source behavior.
- Adversarial review of scope, operator value, conformance wording, and graph/index cardinality.

## Guardrails

- CAP HTTP polling, CAP parser validation, and CAP hazard projection do not prove broader EDXL support.
- Broader EDXL should not be modeled as CAP advisory text unless the product explicitly treats it as human-readable
  evidence only.
- Distribution envelopes and resource messages must not overwrite stricter tactical source truth.
- Service capability, webhook hosting, and emergency-alerting authority remain separate product claims.
