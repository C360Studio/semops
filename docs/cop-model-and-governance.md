# COP Model And Governance

Status: initial test-backed baseline for `COP-002`, created on 2026-06-17.

Code source: `pkg/cop/contracts.go`

## Canonical Entity Set

| Entity | Purpose | First indexing shape |
| --- | --- | --- |
| `track` | Moving thing from MAVLink, TAK/CoT, ADS-B, SAPIENT, or fusion | `signal` |
| `asset` | Responder, platform, sensor host, infrastructure, or resource | `control` |
| `hazard_area` | Flood, fire, plume, debris, exclusion, or evacuation geometry | `content` or `control` |
| `sensor_footprint` | Observed area from drone, video, KLV, or other sensor metadata | `signal` |
| `alert` | Rule, source, or fusion alert with severity and active state | `control` |
| `task` | Operator intent, requested action, or assignment | `control` |
| `advisory` | Human-readable or semantic-tier advisory text | `content` |

## First Ownership Matrix

| Owner | Contract | Entity pattern | Mode | Profile |
| --- | --- | --- | --- | --- |
| `semops.feed.asset` | Source asset identity | `c360.*.cop.*.asset.*` | `replace-owned` | `control` |
| `semops.feed.mavlink` | MAVLink current track state | `c360.*.cop.mavlink.track.*` | `replace-owned` | `signal` |
| `semops.feed.tak` | TAK/CoT current track state | `c360.*.cop.tak.track.*` | `replace-owned` | `signal` |
| `semops.feed.tak` | TAK/CoT marker and task control state | `c360.*.cop.tak.task.*` | `replace-owned` | `control` |
| `semops.feed.tak` | TAK/CoT GeoChat and advisory text | `c360.*.cop.tak.advisory.*` | `replace-owned` | `content` |
| `semops.feed.cap` | CAP hazard/advisory evidence | `c360.*.cop.cap.hazard_area.*` | `append-evidence` | `content` |
| `semops.fusion.structural` | Fusion alert state | `c360.*.cop.fusion.alert.*` | `replace-owned` | `control` |

Strict feed owners are source-partitioned by the SemStreams entity `system` segment. This prevents MAVLink and TAK from
claiming the same `cop.track.position` cell over a wildcard `track` pattern.

TAK/CoT is intentionally one feed owner with multiple contracts. Operator and air-track positions stay in `signal`;
durable markers and task-like map control state stay in `control`; GeoChat text becomes `content`. Only TAK track
state declares the strict `cop.track.source` edge to a born source asset.

Loose CAP evidence does not own authoritative hazard geometry, severity, or status. It appends advisory text, source
references, evidence, observed time, and confidence until a deterministic hazard projector earns stricter ownership.

## ADR-055/056 Born-First Discipline

SemOps adapters must follow SemStreams ADR-055 and ADR-056 directly:

- Entity birth uses `graph.CreateEntityWithTriplesRequest`, with `MessageType` and `IndexingProfile` set.
- Updates use `graph.UpdateEntityWithTriplesRequest` against entities that are already born.
- No SemOps adapter may rely on `triple.add` or `triple.add_batch` auto-vivify to create missing entities.
- Every relationship written onto a different entity must be declared by a projection contract `ForeignEdge`, which
  derives a SemStreams `ownership.ForeignEdgeClaim`.
- The first MAVLink and TAK `cop.track.source` edges are `EdgeStrict` born-first edges. The target source asset must
  be born before the track edge is written.
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

| Family | Examples | Notes |
| --- | --- | --- |
| Track current state | `cop.track.position`, `cop.track.velocity`, `cop.track.status` | Strict source-owned state |
| Task/control state | `cop.task.name`, `cop.task.position`, `cop.task.status` | TAK markers and later assignments |
| Advisory content | `cop.advisory.text`, `cop.advisory.sender` | GeoChat, notes, and advisory text |
| Hazard evidence | `cop.hazard.advisory_text`, `cop.hazard.evidence`, `cop.hazard.source` | CAP starts append-only |
| Alert derived state | `cop.alert.severity`, `cop.alert.status`, `cop.alert.reason` | Fusion-owned derived facts |
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
- TAK binds track, task/control, and advisory/content contracts under the same feed owner without overlapping
  replace-owned predicates.
- Track foreign edges derive explicit ADR-056 `ForeignEdgeClaim` values with producer and target pattern.
- Overlapping `replace-owned` predicates are rejected.
- CAP evidence does not claim authoritative hazard state.
