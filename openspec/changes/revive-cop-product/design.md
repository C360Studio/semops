## Context

The COP revival has two simultaneous jobs:

- Build a compelling SemOps product that can stand on its own as an operator-facing HA/DR COP.
- Put real pressure on SemStreams so framework gaps are discovered by use, not by abstraction.

Current SemOps has useful robotics and MAVLink reference code, but the repo predates the current SemStreams module
path and contract model. Current SemLink has a good modern demo shape: bounded raw telemetry, current-state
projections, ownership contracts, indexing profiles, TAK support, CS API projection, and Svelte UI. SemLink should
stay a basic demo. SemOps should absorb the useful patterns and own the larger COP.

Claude's initial plan framed orchestration and a "thin COP shell" as if they were product features. That is not
accepted here. SemOps is the COP product. Orchestration may become useful, but only if a concrete operator workflow
proves it is more than visual complexity.

## Goals / Non-Goals

**Goals:**

- Make SemOps the complete COP product repo.
- Keep SemStreams as substrate and avoid product assumptions in framework code until proven reusable.
- Reuse SemLink patterns where they are valuable without making SemOps a SemLink wrapper.
- Salvage SemOps MAVLink/SITL depth into a modern adapter package.
- Define feed ownership and projection contracts before implementation spreads.
- Build a Phase 1 structural COP that is showable before all feeds and inference tiers exist.
- Gate orchestration/topology/escalation UI behind explicit operator-value evidence.
- Create upstream SemStreams asks only from concrete SemOps workflows or blockers.

**Non-Goals:**

- Preserve old SemOps APIs or package boundaries for compatibility.
- Move the complete COP product into SemLink.
- Claim SAPIENT, OGC, STANAG 4609, or ASTERIX conformance before tests prove it.
- Treat orchestration, tier placement, or topology panels as accepted features before proving their value.
- Build a general orchestration framework inside SemOps without a later upstream review.
- Put every deterministic mapper into its own container.

## Decisions

### 1. SemOps owns the complete COP product

SemOps owns the canonical COP model, operator UX, HA/DR scenario, feed set, source/provenance lens, fusion rules,
and product vocabulary.

SemLink can be copied from, imported from, or used as a staging reference when useful. It should remain a basic
demo unless a future decision explicitly changes that repo boundary.

### 2. Feed adapters are boundary services when deployment forces justify it

Adapters that own external protocols or different runtime placement can run as services:

- MAVLink over UDP/TCP/SITL.
- TAK/CoT over UDP/TCP/XML.
- CAP/EDXL ingestion.
- SAPIENT protobuf.
- ADS-B raw JSON first, ASTERIX later.
- KLV subset extraction.

The mapping logic inside those services should stay as libraries where possible so tests can run without Compose.

### 3. SemStreams projection contracts govern every graph write

Each feed or derived fusion flow gets a projection owner. Strict adapters reject malformed native data before graph
write. Loose adapters write best-effort evidence with confidence and provenance, but do not replace authoritative
predicates owned by stricter sources.

Fusion is a derived owner, not an invisible side effect of adapter code.

SemOps also follows SemStreams ADR-055/056 explicitly:

- Entity birth is born-first through typed create-with-triples requests.
- Updates assume the entity already exists; no adapter may rely on `triple.add` auto-vivify.
- Cross-entity relationships are declared by projection-contract foreign edges that derive
  `ownership.ForeignEdgeClaim` values.
- `EdgeNoBirthStub` requires a recorded review that proves the target has no independent producer.

### 4. Phase 1 stays structural and complete

The first demo should not wait for seven feeds and three tiers. Phase 1 is complete when MAVLink, TAK/CoT, and
CAP/EDXL produce live governed graph state and the COP shows map, source, provenance, feed health, and alerts.

### 5. Orchestration is a scope-gated hypothesis

Do not assume a deployment manifest, topology panel, or escalation policy is a product feature. Start by recording
health, source, provenance, and inference evidence. Promote orchestration into the product only if it answers a real
operator question better than simpler COP state.

Before hardening any reusable placement or escalation primitive, review it for SemStreams ownership.

### 6. SemOps creates upstream asks with evidence

Candidates for SemStreams upstream work include:

- Deployment manifest and tier placement schema, if the scope gate proves it useful.
- Escalation event/status vocabulary, if inference evidence proves a reusable transition model.
- Provenance and confidence conventions.
- Indexing profile or cardinality-policy gaps discovered by mixed COP feeds.
- Spatial-temporal query helpers.
- Raw-lane plus current-state projection guidance.
- Edge/core sync patterns.
- Tolerant-reader governance helpers.

Each ask must cite a SemOps workflow, failing test, missing primitive, or demo constraint.

### 7. Feeds enter one at a time through evidence gates

Feed order is MAVLink, TAK/CoT, CAP/EDXL, SemConnect CS API egress, ADS-B, SAPIENT, then KLV/STANAG 4609.

Every feed needs a parser gate, mock or simulator gate, projection gate, replay gate, and demo gate. Compliance gates
are required where a public suite, official schema, or documented interoperability test exists. If no compliance
surface is verified, the gap must be recorded before implementation starts.

The first SemStreams indexing-pressure question is whether entity boundaries are right. High-rate state should remain
`signal`, durable operational state should be `control`, advisory text should be `content`, and replay/native decode
detail should be `trace`. Do not request new framework profile semantics until SemOps proves that correct entity
boundaries are insufficient.

### 8. Key stages require adversarial review

SemOps should deliberately attack its own assumptions before stage transitions. Required review gates are:

- Framework contract modernization.
- COP entity and predicate model stabilization, including born-first and foreign-edge discipline.
- Each Phase 1 feed entering the structural stack.
- Orchestration, topology, or tier UI promotion.
- SAPIENT or KLV product commitment.
- Upstream SemStreams issue filing.

Reviews should challenge product value, protocol evidence, compliance wording, framework ownership, indexing profile
choice, cardinality risk, binary-handling claims, and demo credibility. The output is a short record with objections,
accepted risks, and follow-up tasks.

## Risks / Trade-offs

- Scope is large. Phase 1 must remain showable without ADS-B, SAPIENT, KLV, SemConnect, or semantic translation.
- SemOps could accidentally reimplement framework primitives. Keep framework-alignment review as a standing gate.
- SemLink code may be tempting to fork wholesale. Prefer pattern reuse and deliberate porting.
- Loose civilian feeds can corrupt trust if they replace authoritative facts. Enforce ownership and provenance early.
- Container sprawl can slow the demo. Start with a compact stack and split services when placement requires it.
- Mixed-shape feeds can blur indexing policy. Split entities by storage/cardinality shape before asking SemStreams
  for new profile semantics.
- Binary-video claims are risky. KLV remains a proof spike until a small fixture proves metadata extraction,
  binary-by-reference storage, and memory-bounded handling.
- Adversarial reviews can slow execution if they become generic meetings. Keep them evidence-based and tied to stage
  decisions, not broad design theater.

## Migration Plan

1. Modernize SemOps module path and Go toolchain against current SemStreams.
2. Add canonical COP entity and predicate contracts.
3. Move useful MAVLink parser, generator, and SITL code behind a clean adapter package.
4. Add structural projection writers and born-first contract tests.
5. Add the feed validation and indexing ladder for MAVLink, TAK/CoT, CAP, CS API egress, ADS-B, SAPIENT, and KLV.
6. Run adversarial reviews for framework modernization, COP model, and feed evidence before Phase 1 implementation.
7. Add first Compose stack with NATS, SemStreams, SemOps API, UI, scenario runner, and three feed adapters.
8. Build the source/provenance COP product surface.
9. Add ADS-B/SAPIENT and statistical track association.
10. Add KLV, SemConnect egress, and semantic translation.
11. Split edge/core placement after the single-stack demo is stable.

## Open Questions

- Should SemOps expose its own API first, or mirror SemLink's `/api/snapshot` and `/api/events` shape initially?
- Which COP predicates should immediately move to SemStreams vocabulary?
- How should confidence be represented in triples where source confidence and fusion confidence differ?
- Is a topology panel useful at all, or do source health and provenance answer the operator need?
- What is the minimum manifest metadata that avoids becoming a fake orchestrator?
- Which SAPIENT and KLV subsets are demo-grade but honest?
- Which feeds prove that current SemStreams indexing profiles need changes, versus better SemOps entity boundaries?
- What is the lightest review record that preserves adversarial value without slowing the demo cadence?
