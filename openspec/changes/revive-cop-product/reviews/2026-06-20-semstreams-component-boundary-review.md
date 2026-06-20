# SemStreams Component Boundary Review

Date: 2026-06-20

## Decision

Accept SemStreams component lifecycle, flowgraph, port, payload-registry, and config-schema patterns as the hosted
feed-service boundary for SemOps.

Pure feed packages under `pkg/adapters` remain library code for native decode, validation, raw-lane, and replay
behavior. Hosted feed services must be composed as SemStreams input and processor components. Transport listeners
publish registered raw `message.BaseMessage` payloads on declared stream output ports; parser/decoder processors
consume those ports and publish decoded outputs that other components can tap; projection processors write governed
graph mutations only through declared request ports. Components expose config schemas, health, and flow metrics. Graph
writes continue to use SemStreams projection, ownership, graph mutation, indexing-profile, and retry-aware
request/reply contracts.

SemOps should also treat SemStreams utility packages as part of the framework surface. `natsclient`, `pkg/errs`,
`pkg/cache`, and `pkg/buffer` are preferred starting points for common runtime concerns before SemOps grows local
copies.

Backpressure and telemetry must stay visible at component boundaries. Plain NATS stream ports are acceptable for local
first smokes, but durable or high-rate edges should promote to JetStream ports, bounded `pkg/buffer`, or `pkg/cache`
only when component `Health()`, `DataFlow()`, Prometheus metrics, queue depth, drop count, retry/redelivery pressure,
or replay requirements prove the need. SemStreams issue #309 now tracks the reusable framework gap for richer
backpressure metrics across component flow ports.

This review also deletes the stale `configs/robotics-flow.json` file because it preserved raw subject topology from an
old StreamKit/BaseProcessor-era model and polluted the active SemStreams flowgraph and component story.

## Objections Raised

- Focusing only on graph ingest underuses SemStreams as a framework. Component lifecycle, flowgraph, ports, payload
  registry, config schema, health, and flow metrics are framework contracts too.
- A package named `adapters` can hide two meanings: pure codec/replay library versus hosted feed component. Specs must
  keep that distinction explicit.
- Raw NATS subjects are necessary wire details for SemStreams graph request/reply, but SemOps must not treat subject
  plumbing as a substitute for component metadata, flowgraph edges, payload registrations, ownership binding, and
  config.
- A monolithic "adapter service" would make it harder to tap intermediate outputs. The flow needs explicit transport
  input, decoder processor, and projection/fusion processor stages.
- Leaving stale flow JSON in the repo gives future agents an attractive but wrong architecture.

## Evidence Checked

- SemStreams `component.LifecycleComponent`, `Discoverable`, `Port`, `NetworkPort`, `NATSPort`, `NATSRequestPort`,
  `component/flowgraph`, `message.BaseMessage`, and `payloadregistry.Registry` interfaces.
- SemOps `internal/graphrequest.NATSRequester`, which wraps SemStreams `natsclient.RequestWithRetry`.
- SemOps projectors and graph writers for MAVLink, TAK/CoT, CAP, and ADS-B.
- SemOps ownership registration through `ownership.EnsureBuckets` and `projection.BindAndHeartbeat`.
- Stale `configs/robotics-flow.json` raw-subject flow configuration.
- `internal/components/mavlink` concrete input -> decoder processor -> projection processor component package.
- `internal/components/cot` concrete UDP/TCP input -> decoder processor -> projection processor component package.
- `internal/contracts/semstreams_contract_test.go` component-contract guard, now pointed at the production MAVLink and
  TAK/CoT components rather than a parallel skeleton.

## Accepted Risks

- MAVLink and TAK/CoT runtime ingress now use concrete SemStreams input and processor components. ADS-B scenario replay
  remains an in-process harness by design; live ADS-B ingress, hosted CAP if promoted, and SAPIENT still need the same
  treatment before SemOps can claim those hosted feed services are component-managed.
- The graph writer code still names SemStreams graph mutation subjects directly. That is acceptable as the graph API
  wire boundary, but component ports must describe those resources when services are promoted.
- CAP, future live ADS-B, and SAPIENT paths need lifecycle review before they become hosted feed services.

## Follow-Up Tasks

- Wrap live ADS-B ingress, hosted CAP if promoted, and future SAPIENT feed boundaries as SemStreams input and processor
  components.
- Audit feed-runtime helpers against SemStreams utilities (`natsclient`, `pkg/errs`, `pkg/cache`, `pkg/buffer`) before
  adding or expanding SemOps-local equivalents.
- Wire hosted component metrics into Prometheus and use lag/drop/retry evidence before adding local buffers, caches,
  or JetStream durability to a flow edge; reconcile the resulting needs with SemStreams issue #309.
- Register raw and decoded feed payload types in SemStreams payload registries and emit `message.BaseMessage`
  envelopes on stream output ports.
- Compose feed topology through SemStreams flowgraph edges so every declared output port remains tappable by another
  component.
- Route component config through SemStreams `component.ConfigSchema` rather than SemOps-only env parsing where the
  component framework can own it.
- Add adversarial review before claiming SemOps has a complete SemStreams component-hosted runtime.
