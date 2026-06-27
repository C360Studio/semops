# SemConnect CS API Read-Side Interop Evidence

Status: read-side standards egress candidate for MVP; write-side ingress and tasking remain stretch goals.

## Positioning

SemOps should treat OGC Connected Systems API as an interface, not as the internal COP architecture.

The product core should ingest native formats close to their operational source, project them into SemStreams governed
graph state, and preserve source-specific semantics, provenance, freshness, and ownership. A CS API bridge belongs at
the standards edge: useful for systems that already publish CS API, for consumers that need CS API, and for formal
conformance evidence once SemOps has meaningful Systems, Datastreams, Observations, Deployments, and Events to expose.

This "native core plus standards bridge" posture gives SemOps three advantages:

- **Velocity:** MAVLink, TAK/CoT, CAP, ADS-B, SAPIENT, and KLV capabilities can be adopted without waiting for every
  native detail to fit a standards schema first.
- **Semantic flexibility:** SemStreams graph state can model multi-feed fusion, ownership, confidence, provenance, and
  agent-facing evidence without being constrained to a standards exchange shape.
- **Risk isolation:** If CS API mappings or versions move, the bridge changes. The COP core and native adapters do not
  become coupled to an external standards lifecycle.

CS API still matters. It buys decoupling for standards-aware clients, a possible plug-and-play ecosystem for vendors
that already expose CS API, and a unified tasking/actuation vocabulary. For the MVP, SemOps should prioritize
read-side egress: projecting governed COP state into CS API-shaped Systems, Datastreams, Observations, Deployments,
and System Events through SemConnect. Write-side ingress, Command, and ControlStream handling remain stretch goals
until command authority, TTL, priority, local override, and native safety gates are deliberately reopened.

## Decision

- Native adapters remain first-class for feeds whose native protocols carry product-critical semantics.
- CS API read-side egress is the MVP priority and remains a standards-facing view over SemOps-owned graph state.
- CS API ingress is a stretch adapter boundary for systems that already speak CS API. When implemented, it maps into
  the governed COP model through the same ownership, provenance, freshness, and indexing discipline as native feeds.
- CS API tasking or actuation is a stretch command boundary. It must route through SemOps command authority and
  governance rather than bypassing native command safety.
- CS API tasking must be asynchronous at the SemOps boundary: accept or reject the external request quickly, persist a
  governed desired-state or command-intent record, and let native drivers reconcile actual tactical execution.
- SemConnect remains the conformance anchor unless the organization explicitly recharters SemOps to own a CS API
  gateway product.

## Local Evidence

- `internal/egress/csapi` maps SemOps COP snapshot state into CS API-shaped read resources.
- `internal/egress/semconnect` maps that read model into deterministic SemConnect HTTP request specs for the
  read-side resource families SemOps can currently project, then executes those specs through an HTTP client boundary
  that can target SemConnect without direct graph or NATS writes.
- `cmd/semops-semconnect-fixture` builds a deterministic read-side COP snapshot fixture, projects it through the CS API
  read model and SemConnect request plan, and emits JSON evidence for dry-run, success, or partial gateway failure.
- `scripts/semconnect-service-stack.sh` starts SemConnect as a SemOps-consumable service without Team Engine: it brings
  up the SemStreams tiered stack plus `cs-api-server`, exposes CS API on `http://127.0.0.1:48080`, and leaves ETS
  conformance as a separate gate.
- `/Users/coby/Code/c360/semconnect/conformance/run.sh` boots NATS, `semstreams-backend`, `cs-api-server`, and
  Team Engine, seeds fixtures, invokes the ETS, and archives TestNG reports plus service logs.
- `/Users/coby/Code/c360/semconnect/conformance/README.md` states the current Stage 56 pinned suite is green:
  `total=137 passed=137 failed=0 skipped=0`.
- On 2026-06-27, SemConnect commit `15a6db6` (`fix(cs-api): align semstreams ADR-060 replies`) passed the full ETS
  harness against SemStreams `v1.0.0-beta.116`: `total=137 passed=137 failed=0 skipped=0`; the foreign-edge bake also
  passed.
- The SemConnect harness exercises real graph reads/writes, observation publish/readback, artifact storage,
  discovery, OpenAPI, content negotiation, and claimed CS API conformance classes.
- SemLink already demonstrates a SemConnect CS API bridge pattern for current vehicle/COP state.

## External Evidence

- OGC describes Connected Systems API as a bridge between static feature resources and dynamic data. Part 1 covers
  Systems, Procedures, Deployments, Sampling Features, Subsystems/Components, and Property Definitions. Part 2 covers
  Dynamic Feature Properties, Data Streams, Observations, Control Streams, Commands, Command Status, and System Events,
  plus streaming and snapshot mechanisms.
- SemConnect uses OGC Team Engine and the Botts CS API ETS as its conformance harness.

## Gates

### Egress Gate

Target command after SemOps exposes structural graph state:

```bash
go test ./internal/egress/...
```

Service-mode smoke, without Team Engine:

```bash
scripts/semconnect-service-stack.sh up
go run ./cmd/semops-semconnect-fixture -base-url http://127.0.0.1:48080
scripts/semconnect-service-stack.sh down
```

Current service-mode result on 2026-06-27:

- `scripts/semconnect-service-stack.sh up` successfully starts NATS, the SemStreams statistical backend, and
  `cs-api-server` at `http://127.0.0.1:48080`.
- After updating SemConnect to the current SemStreams ADR-060 mutation/query response contract, the SemOps fixture
  returns `status=passed`.
- The fixture executes five planned CS API requests and receives `201` for `/systems`, `/deployments`,
  `/datastreams`, `/datastreams/{id}/observations`, and `/systemEvents`.
- `scripts/semconnect-service-stack.sh down` tears the stack down cleanly after the smoke. ETS conformance remains a
  separate SemConnect harness gate.
- The matching SemConnect ETS gate was also run separately on the pushed SemConnect fix (`15a6db6`) and remained
  `total=137 passed=137 failed=0 skipped=0` against SemStreams `v1.0.0-beta.116`.

Acceptance:

- SemOps can map a COP asset/platform to a CS API System.
- SemOps can map a sensor and current observation to CS API datastream/observation surfaces.
- SemOps can map deployments and system events without creating product claims for unsupported native feeds.
- SemOps can preserve source provenance and ownership when projecting to egress.
- SemOps can emit a deterministic SemConnect HTTP request plan for `/systems`, `/deployments`, `/datastreams`,
  `/datastreams/{id}/observations`, and `/systemEvents` using SemConnect-compatible content types and six-token
  resource IDs.
- SemOps can execute that request plan against an HTTP server boundary and stop with partial evidence on the first
  SemConnect error response.
- SemOps can run `go run ./cmd/semops-semconnect-fixture -dry-run` for deterministic bridge evidence, or pass
  `-base-url` to drive a SemConnect-compatible HTTP gateway without publishing directly to graph or NATS subjects.
- The service-mode bridge smoke starts only NATS, a SemStreams backend, and `cs-api-server`; it does not start
  Team Engine or claim OGC ETS conformance.
- CS API egress remains a view and does not decide SemOps indexing profiles.
- CS API egress does not create Command, ControlStream, or write-side ingress behavior.

### Stretch Ingress Gate

Target command after SemOps has a CS API ingress adapter boundary:

```bash
go test ./internal/adapters/csapi
```

Acceptance:

- SemOps can map CS API System, Datastream, Observation, Deployment, and SystemEvent inputs into canonical COP state.
- CS API ingress does not auto-vivify graph entities outside SemStreams born-first mutation contracts.
- Source provenance records that the data arrived through CS API without erasing the upstream system identity.
- Indexing profiles are selected by SemOps entity semantics, not by the fact that the transport was CS API.
- CS API Command or ControlStream input routes through SemOps command authority rather than directly mutating
  feed-owned state.

### Stretch Command Impedance Gate

Target command after SemOps promotes command intent beyond the pure planner gate:

```bash
go test ./internal/commands ./internal/adapters/csapi -run Command
```

Acceptance:

- CS API Command or ControlStream POST handling validates the request and returns an immediate standards-shaped
  accepted or rejected response without opening a synchronous native radio/session dependency.
- Accepted tasking becomes a governed desired-state or command-intent record with source, authority, priority,
  TTL/deadline, target entity, idempotency key, correlation ID, and audit provenance.
- SemOps command-intent admission rejects unresolved targets, expired intents, and duplicate idempotency keys before
  the CS API bridge returns a standards-shaped accepted response.
- Native MAVLink, TAK/CoT, SAPIENT, or future feed-specific drivers reconcile command intent into protocol-specific
  action only after command authority, safety, and local operator override rules pass.
- Actual state, command acknowledgement, execution progress, timeout, cancellation, rejection, supersession, and
  failure are recorded as graph state or evidence that the CS API bridge can translate back to Command Status or
  System Event surfaces.
- Local SemOps operator intent, upstream federated intent, stale commands, duplicate commands, conflicting priorities,
  lost native links, and partial execution all have deterministic policies before live tasking is demonstrated.
- Command TTL windows and freshness checks prevent old upstream instructions from becoming live native actions after
  reconnect or replay.
- The bridge remains a standards interface; it does not decide native safety, priority arbitration, or actuation
  eligibility on its own.

### Harness Gate

Target command in the SemConnect checkout:

```bash
./conformance/run.sh
```

Acceptance:

- The harness runs end to end and archives TestNG XML.
- Any pass/fail count is read from TestNG output, not inferred from `go test`.
- If SemOps-specific ingress or egress changes SemConnect mappings, the conformance delta is recorded.

### Replay Gate

Target artifact:

- A SemOps graph fixture or COP snapshot read model with one asset/platform, one hosted sensor, one datastream, one
  observation, one deployment, and one system event.

Acceptance:

- The fixture projects deterministically through SemConnect.
- The fixture runner records status, resource counts, request results, deferred write-side surfaces, and the reminder
  that SemConnect ETS remains the conformance acceptance gate.
- SemConnect conformance remains a separate acceptance gate.

## Known Gaps

- SemOps now has an internal read-side egress model that maps COP snapshot state into CS API-shaped Systems,
  Datastreams, Observations, Deployments, and System Events, plus a deterministic SemConnect HTTP request-plan export,
  HTTP executor, and fixture-runner CLI for those read-side resources. It is not yet a SemConnect ETS conformance
  fixture.
- The lightweight service-mode stack starts successfully and the SemOps fixture passes against SemConnect, but this is
  bridge smoke evidence rather than OGC ETS conformance evidence.
- SemOps does not yet expose a CS API ingress adapter.
- SemOps now defines a command-intent graph contract plus pure planner/admission/arbitration tests for required
  fields, target resolution, expiry, duplicate idempotency, local override, authority ranking, and per-target priority
  selection. The guarded batch path projects accepted/superseded command-intent status before exposing accepted native
  execution candidates. SemOps also constrains command-intent lifecycle vocabulary and transition validation before
  any CS API/UI/native status handler exists, and can build pure `cancel_requested` updates for active command intents.
  Native status reconciliation can map feed readback evidence such as MAVLink COMMAND_ACK into constrained status-only
  command-intent updates. Deadline reconciliation can map stale requested commands to `expired` and stale active
  commands to `timeout`. SemOps now carries a manifest-listed synthetic command lifecycle replay fixture for route
  cancellation, accepted survey timeout, and stale-command expiry; it is useful for CS API/UI/native-status plumbing
  tests, but is not CS API service support, hosted scheduling, native cancellation acknowledgement, native transmit, or
  live actuation evidence. The COP API/UI can now read command task state by prefix and show read-only lifecycle
  details, but it still has no execute/cancel UI, CS API request handler, scheduler, native cancellation
  acknowledgement, or live actuation path. Hosted deadline scheduling, CS API request handling, native cancellation
  acknowledgement, and live actuation remain open.
- CS API read-side egress should not block Phase 1 structural COP.
- CS API write-side ingress and tasking are stretch goals, not MVP gates.
- Native adapter support can be strong while CS API projection is still incomplete; keep those claims separate.
- CS API conformance can be green while a native feed remains only fixture/replay-tested; keep those claims separate.

## Adversarial Feed-Entry Questions

- Are we using CS API as an interface instead of making it the core SemOps architecture?
- Does native ingestion preserve operational velocity for HADR-style data arriving in the format agencies brought?
- Does the bridge preserve ownership, provenance, freshness, and indexing decisions made by SemOps graph contracts?
- Does tasking through CS API route through command authority rather than bypassing native safety controls?
- Are TTL, priority, idempotency, cancellation, supersession, local override, and partial-execution cases specified
  before any CS API command reaches a native driver?
- Does the conformance result come from the actual harness output?

## Source Links

- OGC API - Connected Systems overview: <https://ogcapi.ogc.org/connectedsystems/>
- OGC Connected Systems SWG repository: <https://github.com/opengeospatial/ogcapi-connected-systems>
- OGC Team Engine: <https://github.com/opengeospatial/teamengine>
