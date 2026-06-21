# ADS-B Component Promotion Review

Date: 2026-06-21

## Decision

Promote the first ADS-B hosted ingress shape into a SemStreams component package, but keep live runtime hosting and
receiver support gated.

`internal/components/adsb` is acceptable now because it is no longer wrapping only deterministic scenario replay. It
models a real hosted ingress option: an OpenSky-compatible HTTP polling input component that emits raw snapshot
`message.BaseMessage` payloads to a stream, followed by decoder and graph-projector processor components. The chain
declares `HTTPClientPort`, `TimerPort`, NATS stream ports, NATS request ports, config schema, health, flow metrics,
payload registry entries, replay capture, and born-first graph projection behavior.

This does not claim default live OpenSky service support. The tests use local HTTP fixtures, and product runtime
wiring remains a later opt-in decision.

## Objections Raised

- A component package could be mistaken for live feed reliability. The docs and evidence must say that OpenSky live
  credentials, rate limits, stale policy, and runtime wiring remain separate gates.
- OpenSky is only one ingress force. Local readsb/dump1090 files, receiver TCP/UDP, and ASTERIX still need their own
  input components if chosen; they should not be squeezed through the HTTP poller.
- ADS-B can be high-rate enough to stress SemStreams flow telemetry, buffering, and cache choices. The first package
  exposes `DataFlow()` and stale health, but runtime backpressure decisions must be made with real feed metrics.
- Cross-source aircraft association remains fusion-owned. The ADS-B projector must not emit association/source
  foreign edges just because an aircraft looks related to MAVLink, SAPIENT, or operator tracks.

## Evidence Checked

- `internal/components/adsb` publishes raw OpenSky-shaped snapshots from an HTTP input component and decodes them
  through registered payload types.
- The decoder captures replay records before publishing decoded snapshots and captures malformed snapshots before
  returning parse failures.
- The graph projector uses SemStreams create/update request-port metadata and reconciles `entity_already_exists`
  create conflicts into update projection.
- `internal/contracts` now has a framework-level guard for ADS-B lifecycle, `HTTPClientPort`, `TimerPort`, payload
  registry, flowgraph stream edges, and graph request ports.
- Package tests use `httptest.Server`, not external OpenSky network access.

## Follow-Up Tasks

- Decide whether the next ADS-B runtime step is opt-in OpenSky HTTP hosting or local receiver/readsb/dump1090 input.
- Add Prometheus/backpressure evidence before increasing live ADS-B rate or fan-out.
- Keep ASTERIX as a separate binary/radar-like proof spike.
- Keep SAPIENT graph projection and hosted component work behind its source-owner and harness review gates.
