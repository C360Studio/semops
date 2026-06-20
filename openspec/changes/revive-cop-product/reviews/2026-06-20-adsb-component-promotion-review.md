# ADS-B Component Promotion Review

Date: 2026-06-20

## Decision

Do not promote the current ADS-B scenario replay seam into a SemStreams component package yet.

The current ADS-B path is an opt-in deterministic scenario harness: OpenSky-shaped fixture snapshots are replayed
inside `cmd/semops-scenario-runner` through `internal/adapters/adsb`, then projected into governed graph state with a
SemStreams-minted `semops.feed.adsb` owner token. That is useful product evidence, but it is not a hosted external
feed service.

The component boundary becomes mandatory when SemOps adds live OpenSky polling, readsb/dump1090 files, TCP/UDP
receiver input, ASTERIX, or any other continuously hosted ADS-B ingress. At that point the input stage must be a
SemStreams input component with declared network, file, or request ingress ports, registered raw snapshot payloads,
and telemetry. Decode/project/fusion stages must be SemStreams processor components with tappable output ports and
declared graph request ports.

## Objections Raised

- Wrapping scenario fixtures in components now would create compliance theater. The value of SemStreams components is
  lifecycle, config, ports, topology, and operational telemetry for hosted flows, not making a unit-test harness look
  distributed.
- A future ADS-B service may start from file-tail receiver output, authenticated OpenSky polling, or a local receiver
  stream. Choosing one component shape before that product decision risks hardening the wrong port model.
- ADS-B is high-rate enough that backpressure and replay posture matters. Live promotion should include SemStreams
  `Health()`, `DataFlow()`, Prometheus metrics, and explicit evidence before adding `pkg/buffer`, `pkg/cache`, or
  JetStream durability.

## Evidence Checked

- `internal/adapters/adsb` already provides parser, bounded raw capture, replay append, projection, graph writes,
  restart birth reconciliation, and pollable health for deterministic snapshots.
- `cmd/semops-scenario-runner` opts into ADS-B with `SEMOPS_SCENARIO_ADSB_FIXTURE=true` and registers
  `semops.feed.adsb` only for that scenario path.
- `openspec/changes/revive-cop-product/feed-evidence/adsb.md` keeps live mode explicitly future-scoped.
- SemStreams `v1.0.0-beta.113` component ports include `NetworkPort`, `NATSPort`, `NATSRequestPort`, and `FilePort`,
  which are enough to describe likely live ADS-B ingress options once SemOps chooses one.

## Follow-Up Tasks

- Keep ADS-B scenario replay as an adapter harness until a live ingress decision exists.
- When live ADS-B starts, choose the first input force explicitly: OpenSky poller, readsb/dump1090 file tail, receiver
  TCP/UDP, or ASTERIX.
- Add ADS-B raw snapshot and decoded state payload registry types before any hosted ADS-B component publishes stream
  ports.
- Use SemStreams issue #309 outcomes to decide whether ADS-B needs JetStream, bounded `pkg/buffer`, or cache support
  across live component edges.
