# SemOps Project Context

## Purpose

SemOps is the complete data-fusion common operating picture product for C360 operational demonstrations. It uses
SemStreams as the governed semantic substrate and turns many tactical, robotic, sensor, civilian, and standards
feeds into one operator-facing COP.

SemOps is allowed to be bold. This repo may break old SemOps assumptions while it revives the product around the
current SemStreams projection, ownership, indexing, rule, and tiering patterns.

## Product Boundary

- SemOps owns the COP product, HA/DR scenario, feed adapters, canonical COP model, UI, fusion rules, scenario
  playback, and product-scoped governance.
- SemStreams owns framework substrate: NATS/JetStream, graph mutation/query APIs, projection contracts, ownership
  claims, indexing profiles, rules, lifecycle primitives, shared utility packages, and reusable tier infrastructure.
- SemConnect owns OGC Connected Systems API bridge behavior and conformance claims unless SemOps is explicitly
  rechartered to own that gateway product.
- SemLink remains a basic GCS/demo proving selected patterns. SemOps may reuse or port anything useful from it, but
  SemOps owns the complete COP product going forward.

## Standing Technical Conventions

- Raw high-rate feed data stays on bounded raw lanes.
- Graph writes represent current state, durable events, provenance, confidence, and relationship evidence.
- Every graph writer declares SemStreams projection and ownership contracts before integration.
- Runtime code prefers SemStreams utility packages such as `natsclient`, `pkg/errs`, `pkg/cache`, and `pkg/buffer`
  before adding SemOps-local equivalents.
- Strict feeds reject malformed messages at the adapter boundary.
- Loose feeds use tolerant readers, then write strict governed projections with confidence and provenance.
- Structural behavior is the default tier. Statistical and semantic work must be explicit, justified, observable,
  and reversible when its justification no longer applies.
- Services become containers when they own protocol boundaries, placement, scaling, secrets, cost, or a distinct
  failure domain. Deterministic mappers stay libraries until those forces appear.

## Initial Product Slice

The first product slice is a structural HA/DR COP:

- MAVLink, TAK/CoT, and CAP/EDXL feed boundaries.
- Canonical entities for tracks, assets, hazard areas, sensor footprints, alerts, tasks, and advisories.
- A SemOps API and Svelte 5 COP product surface with map, source, provenance, and alert lenses.
- A scripted scenario runner so the demo is repeatable.
- A first-class list of upstream SemStreams asks created only from concrete SemOps pressure.
