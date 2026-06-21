# SemConnect CS API Interop Evidence

Status: standards-facing bidirectional interop candidate after structural graph state exists.

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
that already expose CS API, and a unified tasking/actuation vocabulary. SemOps should use those benefits at the edge
without forcing disaster-response feed ingestion through a standards driver before the operator can see the data.

## Decision

- Native adapters remain first-class for feeds whose native protocols carry product-critical semantics.
- CS API ingress is allowed for systems that already speak CS API, but it maps into the governed COP model through the
  same ownership, provenance, freshness, and indexing discipline as any native feed.
- CS API egress is the standards-facing view over SemOps-owned graph state; ingress is the standards-facing input
  adapter for systems that already speak CS API.
- CS API tasking or actuation must route through SemOps command authority and governance rather than bypassing native
  command safety.
- CS API tasking must be asynchronous at the SemOps boundary: accept or reject the external request quickly, persist a
  governed desired-state or command-intent record, and let native drivers reconcile actual tactical execution.
- SemConnect remains the conformance anchor unless the organization explicitly recharters SemOps to own a CS API
  gateway product.

## Local Evidence

- `/Users/coby/Code/c360/semconnect/conformance/run.sh` boots NATS, `semstreams-backend`, `cs-api-server`, and
  Team Engine, seeds fixtures, invokes the ETS, and archives TestNG reports plus service logs.
- `/Users/coby/Code/c360/semconnect/conformance/README.md` states the current Stage 55 pinned suite is green:
  `total=137 passed=137 failed=0 skipped=0`.
- The SemConnect harness exercises real graph reads/writes, observation publish/readback, artifact storage,
  discovery, OpenAPI, content negotiation, and claimed CS API conformance classes.
- SemLink already demonstrates a CS API bridge pattern for current vehicle/COP state.

## External Evidence

- OGC describes Connected Systems API as a bridge between static feature resources and dynamic data. Part 1 covers
  Systems, Procedures, Deployments, Sampling Features, Subsystems/Components, and Property Definitions. Part 2 covers
  Dynamic Feature Properties, Data Streams, Observations, Control Streams, Commands, Command Status, and System Events,
  plus streaming and snapshot mechanisms.
- SemConnect uses OGC Team Engine and the Botts CS API ETS as its conformance harness.

## Gates

### Ingress Gate

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

### Command Impedance Gate

Target command after SemOps has command-intent graph contracts:

```bash
go test ./internal/commands ./internal/adapters/csapi -run Command
```

Acceptance:

- CS API Command or ControlStream POST handling validates the request and returns an immediate standards-shaped
  accepted or rejected response without opening a synchronous native radio/session dependency.
- Accepted tasking becomes a governed desired-state or command-intent record with source, authority, priority,
  TTL/deadline, target entity, idempotency key, correlation ID, and audit provenance.
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

### Egress Gate

Target command after SemOps exposes structural graph state:

```bash
go test ./internal/egress/csapi
```

Acceptance:

- SemOps can map a COP asset/platform to a CS API System.
- SemOps can map a sensor and current observation to CS API datastream/observation surfaces.
- SemOps can map deployments and system events without creating product claims for unsupported native feeds.
- SemOps can preserve source provenance and ownership when projecting to egress.
- CS API egress remains a view and does not decide SemOps indexing profiles.

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

- A SemOps graph fixture with one asset/platform, one hosted sensor, one datastream, one observation, one deployment,
  and one system event.

Acceptance:

- The fixture projects deterministically through SemConnect.
- SemConnect conformance remains a separate acceptance gate.

## Known Gaps

- SemOps does not yet expose the canonical graph state needed for meaningful CS API ingress or egress.
- SemOps does not yet define command-intent graph contracts, TTL/deadline semantics, priority/authority arbitration,
  local override policy, cancellation/supersession semantics, or native actuation reconciliation.
- CS API interop should not block Phase 1 structural COP.
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
