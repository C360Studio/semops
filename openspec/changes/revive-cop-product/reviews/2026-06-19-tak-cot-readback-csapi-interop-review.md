# TAK/CoT Readback And CS API Interop Review

Date: 2026-06-19

## Decision

Accept the TAK/CoT readback slice as a structural claim:

- The CoT adapter can write born-first graph state for source assets, tracks, tasks, and advisories.
- The live graph smoke can verify track, task, advisory, ownership, and foreign-edge readback against SemStreams.
- The hosted stack can receive MAVLink and CoT over UDP, then expose graph-backed state through the Caddy-routed COP
  snapshot API.
- The UI can render and select CoT-derived tracks, tasks, and advisories through the same map and inspector contract
  as other COP entities.

Accept the CS API strategy as native core plus standards bridge:

- Native adapters remain first-class because operational feeds arrive in native formats and carry source semantics.
- CS API ingress/egress is a standards interface at the edge, not the internal SemOps architecture.
- SemConnect remains the conformance anchor unless SemOps is explicitly rechartered to own a CS API gateway product.

## Findings

- Seed UID readback is a valid demo gate, not a product discovery model. The graph provider still queries configured
  MAVLink system IDs and CoT UIDs directly.
- CoT stale-state behavior is evaluated at COP read time from observation timestamps, which is sufficient for the
  first operator view. It does not yet model TAK stale semantics, contact session state, or federation lifecycle.
- The Compose smoke now has two native feeds in the stack, but CAP is still the first loose civilian feed needed to
  test append-evidence and advisory/hazard behavior.
- CS API should not become a shortcut for raw ingestion. If a source already speaks CS API, consume it through an
  ingress adapter that still maps into SemOps governed graph contracts.

## Accepted Risks

- The first CoT readback path uses known seed UIDs to avoid inventing a graph discovery/index query before the product
  need is explicit.
- UI task/advisory layers are selectable map points and inspector rows, not full tasking or chat workflows.
- The CS API bridge language is aspirational until SemOps has enough structural graph state to run ingress/egress
  mapping tests and SemConnect conformance deltas.

## Follow-Ups

- Add index-backed discovery or query support for CoT entities before scaling beyond deterministic seed fixtures.
- Add CAP/EDXL graph projection and UI readback to exercise loose evidence, hazards, and advisory text.
- Add command/tasking governance before claiming CS API actuation parity or native command authority.
- Keep future CS API work in SemConnect unless SemOps receives an explicit product charter for a standards gateway.
