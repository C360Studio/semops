# COP Model And Governance

Status: initial test-backed baseline for `COP-002`, created on 2026-06-17.

Code source: `pkg/cop/contracts.go`

## Canonical Entity Set

| Entity | Purpose | First indexing shape |
| --- | --- | --- |
| `track` | Moving thing from MAVLink, TAK/CoT, DJI, ADS-B, SAPIENT, or fusion | `signal` |
| `asset` | Responder, platform, sensor host, infrastructure, or resource | `control` |
| `hazard_area` | Flood, fire, plume, debris, exclusion, evacuation, or weather hazard geometry | `content` or `control` |
| `sensor_footprint` | Observed area from drone, video, DJI/KLV, or other sensor metadata | `signal` |
| `alert` | Rule, source, or fusion alert with severity and active state | `control` |
| `task` | Operator intent, requested action, or assignment | `control` |
| `advisory` | Human-readable or semantic-tier advisory text | `content` |
| `weather_observation` | Localized tactical weather variable sample | `signal` |
| `association` | Derived evidence that two source-owned tracks may represent the same object | `control` |

## First Ownership Matrix

| Owner | Contract | Entity pattern | Mode | Profile |
| --- | --- | --- | --- | --- |
| `semops.feed.asset` | Source asset identity | `c360.*.cop.*.asset.*` | `replace-owned` | `control` |
| `semops.feed.mavlink` | MAVLink current track state | `c360.*.cop.mavlink.track.*` | `replace-owned` | `signal` |
| `semops.feed.mavlink` | MAVLink command ACK/readback state | `c360.*.cop.mavlink.task.*` | `replace-owned` | `control` |
| `semops.feed.tak` | TAK/CoT current track state | `c360.*.cop.tak.track.*` | `replace-owned` | `signal` |
| `semops.feed.tak` | TAK/CoT marker and task control state | `c360.*.cop.tak.task.*` | `replace-owned` | `control` |
| `semops.feed.tak` | TAK/CoT GeoChat and advisory text | `c360.*.cop.tak.advisory.*` | `replace-owned` | `content` |
| `semops.feed.adsb` | ADS-B/OpenSky-shaped aircraft current state | `c360.*.cop.adsb.track.*` | `replace-owned` | `signal` |
| `semops.feed.sapient` | SAPIENT absolute-location detection current state | `c360.*.cop.sapient.track.*` | `replace-owned` | `signal` |
| `semops.feed.cap` | CAP hazard/advisory evidence | `c360.*.cop.cap.hazard_area.*` | `append-evidence` | `content` |
| `semops.feed.klv` | KLV-derived sensor/frame-center state | `c360.*.cop.klv.sensor_footprint.*` | `replace-owned` | `signal` |
| `semops.feed.weather` | Tactical weather samples | `c360.*.cop.weather.weather_observation.*` | `replace-owned` | `signal` |
| `semops.command.intent` | Command intent | `c360.*.cop.command.task.*` | `replace-owned` | `control` |
| `semops.fusion.structural` | Fusion alert state | `c360.*.cop.fusion.alert.*` | `replace-owned` | `control` |
| `semops.fusion.structural` | Cross-source track association evidence | `c360.*.cop.fusion.association.*` | `replace-owned` | `control` |
| `semops.fusion.structural` | Operator review audit for association evidence | `c360.*.cop.fusion.association_review.*` | `replace-owned` | `control` |

Strict feed owners are source-partitioned by the SemStreams entity `system` segment. This prevents MAVLink and TAK from
claiming the same `cop.track.position` cell over a wildcard `track` pattern.

MAVLink is intentionally split between `signal` track state and `control` command ACK/readback state. COMMAND_ACK
projection is evidence that a native command lifecycle event was observed, not proof that SemOps has outbound command
authority. MAVLink track state declares the strict `cop.track.source` edge to a born source asset; MAVLink command
tasks declare the strict `cop.task.target` edge to the same born source asset.

Command intent is owned by the SemOps control plane rather than any native feed. It records desired state, authority,
priority, expiry, correlation, idempotency, requested-by, and status fields for later CS API/local-operator ingress.
Native feed drivers reconcile those desired tasks asynchronously and publish ACK/status evidence under their own
contracts.

TAK/CoT is intentionally one feed owner with multiple contracts. Operator and air-track positions stay in `signal`;
durable markers and task-like map control state stay in `control`; GeoChat text becomes `content`. Only TAK track
state declares the strict `cop.track.source` edge to a born source asset.

Loose CAP evidence does not own authoritative hazard geometry, severity, or status. It appends advisory text, source
references, evidence, observed time, and confidence until a deterministic hazard projector earns stricter ownership.

Weather observation evidence is source-partitioned and signal-profiled. It owns localized variable samples with query
shape, geometry, valid time, model time, freshness, unit, provenance, and confidence. It does not own CAP-style hazard
authority, route decisions, task state, or operator advisories.

Track association evidence is fusion-owned. Source feeds continue to own their track current state; the fusion owner
records strict source-track edges, confidence, algorithm identity, distance/time evidence, and source references
without merging or mutating the original tracks. COP readback exposes those records as inspectable association
evidence, not as merged identity state.

Association review is also fusion-owned, but it is deliberately non-authoritative. The review audit records
acknowledge/challenge decisions with `reviewer_role=operator.unverified`, `authority_scope=local.display_only`, and
`conflict_policy=latest_review_wins_display_only`. That lets the COP show local human-in-the-loop review without
changing association scores, merging identities, driving command execution, or publishing upstream CS API status.

SAPIENT detection evidence is currently narrower than SAPIENT product support. The first contract owns
absolute-location detection track state only, rejects range/bearing and UTM projection until those semantics are
reviewed, and declares no association or tasking foreign edges.

## ADR-055/056 Born-First Discipline

SemOps adapters must follow SemStreams ADR-055 and ADR-056 directly:

- Entity birth uses `graph.CreateEntityWithTriplesRequest`, with `MessageType` and `IndexingProfile` set.
- Updates use `graph.UpdateEntityWithTriplesRequest` against entities that are already born.
- No SemOps adapter may rely on `triple.add` or `triple.add_batch` auto-vivify to create missing entities.
- Every relationship written onto a different entity must be declared by a projection contract `ForeignEdge`, which
  derives a SemStreams `ownership.ForeignEdgeClaim`.
- The first MAVLink and TAK `cop.track.source` edges plus command-intent and MAVLink `cop.task.target` edges are
  `EdgeStrict` born-first edges. The target source asset must be born before the track or task edge is written.
- `EdgeNoBirthStub` is allowed only after an adversarial review proves the target has no independent producer and the
  contract names the producer message type plus target pattern.
- SemOps issue #1 tracks the live SemStreams breaking-tag proof for this policy; the first generated-frame MAVLink
  graph smoke passed on 2026-06-17.
- Follow-up clean-stack smokes registered SemOps COP owners, enrolled static owners for heartbeat, and used typed
  SemStreams `OwnerToken` values minted by the registry/bind path for MAVLink writes.
- The hosted `cmd/semops` composition root now registers COP owners before composing the MAVLink adapter, and passes
  typed owner tokens through to projectors.
- The next hardening gates are scenario-runner replay plumbing, transport hosting, and multi-feed graph smoke
  expansion.

Runtime code must treat owner tokens as ownership substrate credentials, not human-authored adapter config. The
composition root registers projection contracts, gets typed `ownership.OwnerToken` values from SemStreams
registry/bind results, and lets projectors serialize `OwnerToken.Wire()` only at the graph mutation request boundary.

SemStreams accepted the SemOps feedback to add typed, opaque owner-token minting through the registry/bind-result path
and to split append-evidence declarations from enforceable ownership/write-fence claims. SemOps now consumes the typed
token path for MAVLink writes and TAK/CoT projection-plan tests.

## Predicate Conventions

Predicate names are product-local until a reusable SemStreams need is proven.

Command intent also carries priority, correlation ID, idempotency key, requested-by, status, provenance, and strict
target-edge fields. Native feed ACK/readback evidence stays under feed-owned contracts.

| Family | Examples | Notes |
| --- | --- | --- |
| Track current state | `cop.track.position`, `cop.track.velocity`, `cop.track.status` | Strict source-owned state |
| Task/control state | `cop.task.name`, `cop.task.position`, `cop.task.status` | TAK markers and later assignments |
| Command intent | `cop.task.desired_state`, `cop.task.authority`, `cop.task.expires_at` | Desired tasking state |
| Advisory content | `cop.advisory.text`, `cop.advisory.sender` | GeoChat, notes, and advisory text |
| Hazard evidence | `cop.hazard.advisory_text`, `cop.hazard.evidence`, `cop.hazard.source` | CAP/weather alerts start append-only |
| Weather evidence | `cop.weather.value`, `cop.weather.variable`, `cop.weather.query_shape`, `cop.weather.query_geometry` | Tactical weather signal |
| Media evidence | `cop.media.ref`, `cop.media.kind`, `cop.media.hash`, `cop.media.time_range` | Candidate only until DJI/KLV media fixtures prove shared vocabulary |
| Alert derived state | `cop.alert.severity`, `cop.alert.status`, `cop.alert.reason` | Fusion-owned derived facts |
| Association review audit | `cop.association_review.reviewer_role`, `cop.association_review.authority_scope`, `cop.association_review.conflict_policy` | Display/audit-only local review semantics |
| Provenance | source, confidence, observed-at, source-ref predicates | Candidate upstream convention |

## Upstream Candidates

Do not file upstream SemStreams vocabulary issues yet. Keep these as candidates until implementation pressure produces
failing SemOps tests or awkward duplicated code:

- A generic provenance source predicate.
- A generic confidence predicate and confidence range convention.
- A generic observed-at or source-time predicate.
- A generic source-reference predicate for bounded raw lanes and replay artifacts.
- Raw-lane plus current-state projection guidance for high-rate telemetry.
- Spatial helper conventions for WKT/GeoJSON position, footprint, and hazard geometry predicates.

## Test Gates

`pkg/cop/contracts_test.go` verifies:

- The canonical entity set is stable.
- First-phase contracts validate and derive SemStreams ownership claims.
- Strict, tolerant, and fusion owners use the expected write modes and indexing profiles.
- MAVLink and TAK strict track contracts are source-partitioned.
- Command intent carries authority, priority, expiry, correlation, idempotency, requested-by, and desired-state fields
  without claiming native feed telemetry or ACK authority.
- MAVLink binds signal track and control command-task contracts under the same feed owner without overlapping
  `replace-owned` predicates.
- TAK binds track, task/control, and advisory/content contracts under the same feed owner without overlapping
  replace-owned predicates.
- ADS-B and SAPIENT track contracts are source-partitioned, signal-profiled, and do not claim association foreign
  edges.
- Fusion track association evidence is control-profiled, source-partitioned under `fusion`, and uses strict track
  edges back to already-born source tracks.
- Fusion association-review evidence is control-profiled, source-partitioned under `fusion`, and carries
  non-authoritative local review role, scope, and conflict policy.
- Track, command-intent, and MAVLink command-task foreign edges derive explicit ADR-056 `ForeignEdgeClaim` values with
  producer and target pattern.
- Overlapping `replace-owned` predicates are rejected.
- CAP evidence does not claim authoritative hazard state.
- KLV sensor-footprint evidence owns sensor/frame-center state without claiming footprint polygons.
- Weather observation evidence uses `signal` indexing and does not claim hazard, alert, task, or route-decision
  authority.
