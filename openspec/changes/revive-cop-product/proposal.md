## Why

SemOps is old relative to current SemStreams, but it was one of the original drivers for the framework. The project
should be revived as the complete COP product: a large, bold data-fusion demo that makes governed multi-producer
state, structural-first operation, inference evidence, and civ-mil translation visible in one experience.

The checkout started unready to carry that product:

- It targeted the old `github.com/c360/semstreams` module path.
- It declared Go `1.25.3` while current SemStreams requires Go `1.26.3`.
- Its entrypoint and flow config were mostly stubs or old StreamKit-era assumptions.
- Its MAVLink parser, generator, and SITL work are valuable references but still need extraction behind modern
  projection and ownership contracts.

SemLink proves useful current patterns, but it will likely remain a basic GCS/demo. SemOps owns the complete COP
product going forward.

## What Changes

- Add a SemOps OpenSpec baseline for the COP revival.
- Establish SemOps as the product owner for the complete COP while treating SemLink as a reusable pattern source.
- Define the first canonical COP entity set and ownership model.
- Require governed feed adapters for MAVLink, TAK/CoT, CAP/EDXL, SAPIENT, ADS-B, KLV, and bidirectional CS API
  interop at the standards edge.
- Pin SemOps to SemStreams ADR-055/056: born-first entity creation, no auto-vivify, and explicit
  `ForeignEdgeClaim`-derived relationship writes.
- Add a feed validation and indexing ladder so every feed has explicit mock, simulator, replay, compliance, and
  `indexing_profile` evidence before it becomes a product claim.
- Treat orchestration, tier controls, and topology panels as hypotheses until they prove COP operator value.
- Require adversarial reviews at key stage boundaries so product value, compliance language, framework ownership,
  indexing choices, and demo credibility are challenged before broad implementation.
- Define a containerized demo stack and the criteria for what becomes a service.
- Add an explicit SemStreams ADR-055/056 breaking-tag compliance gate tracked by SemOps issue #1.
- Track SemStreams upstream pressure from real SemOps product needs.

## Capabilities

### New Capabilities

- `cop-product-ownership`: Makes SemOps the complete COP product and pins repo boundaries against SemStreams,
  SemConnect, and SemLink.
- `governed-feed-fusion`: Requires all feed adapters to canonicalize native formats into governed COP state with
  projection ownership, provenance, and confidence.
- `feed-validation-ladder`: Requires one-feed-at-a-time adoption with explicit compliance/mock evidence and
  indexing-profile/cardinality decisions.
- `adversarial-review-gates`: Requires stage-boundary reviews that actively challenge assumptions before work
  proceeds.
- `orchestration-scope-gate`: Prevents orchestration and topology UI from becoming assumed features before they
  prove operator value.
- `containerized-demo-infra`: Defines which parts of the COP run as services, libraries, or upstream framework
  candidates.

### Modified Capabilities

- Existing robotics/MAVLink processing becomes reference material only while it has extraction value, not product
  architecture.

## Impact

- SemOps architecture docs and tickets.
- Go module path, toolchain, package boundaries, and test strategy.
- MAVLink parser/generator/SITL code migration.
- Generated/replay MAVLink live graph smoke before PX4/SITL, UI, or second-feed expansion.
- New SemOps API and Svelte COP product surface.
- Docker Compose and scenario runner infrastructure.
- SemStreams upstream issues for proven manifest, escalation, provenance, confidence, spatial-temporal query,
  raw-lane, and edge/core patterns.
