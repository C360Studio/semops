# ADS-B Runtime Flow Review

Scope: opt-in hosted ADS-B OpenSky-compatible HTTP component chain

## Decision

Accept ADS-B app-runtime wiring as an opt-in SemStreams component flow. `cmd/semops` may compose the ADS-B HTTP
poller input, decoder processor, and graph projector processor when `SEMOPS_ADSB_ENABLED=true`.

This approves runtime composition, config/env coverage, replay capture, and ownership registration for local
provider-shaped HTTP fixtures. It does not approve default live OpenSky enablement, receiver/readsb/dump1090 support,
ASTERIX, or cross-source aircraft association.

## Adversarial Findings

- Runtime ownership must not silently broaden the first-phase owner set. ADS-B ownership is appended only when the
  hosted ADS-B flow is enabled, matching the scenario-runner opt-in pattern.
- The app must not call ADS-B adapter internals directly. The runtime path composes SemStreams input and processor
  components over NATS subjects and uses `internal/stack.NewADSBPlanWriter` for graph request ports.
- OpenSky-compatible HTTP polling is not the same as ADS-B product support. The path can run against local fixtures
  and custom URLs; default Compose keeps it disabled.
- Raw-lane caps are part of the safety contract. ADS-B can stress buffering and indexing, so runtime config exposes
  record and byte caps before higher-rate receiver work.
- Replay capture is evidence, not canonical graph state. Raw snapshots remain outside canonical tracks; aircraft
  current state stays source-partitioned and association remains fusion-owned.

## Accepted Evidence

- `internal/app` now parses `SEMOPS_ADSB_*` config and validates enabled-runtime settings.
- App startup starts ADS-B projector, decoder, and HTTP poller components behind `SEMOPS_ADSB_ENABLED=true`.
- Runtime ownership includes `semops.feed.adsb` only for enabled ADS-B hosting.
- Local provider-shaped HTTP fixtures drive poller -> decoder -> projector writes and JSONL replay capture.
- `compose.cop.yml` passes ADS-B runtime env through while defaulting `SEMOPS_ADSB_ENABLED=false`.

## Follow-Ups

- Prove the ADS-B opt-in runtime path in the full Compose smoke when Docker resources permit.
- Add local receiver/readsb/dump1090 input components before claiming receiver support.
- Add Prometheus/backpressure evidence before raising ADS-B polling rate or increasing fan-out.
- Keep live OpenSky credential/rate-limit behavior behind a skipped-by-default live smoke.
