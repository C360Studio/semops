# HTTP Feed Fixture Smoke Review

Date: 2026-06-21
Scope: ADS-B and SAPIENT opt-in hosted HTTP component-flow smoke evidence

## Decision

Accept a local HTTP fixture-provider service for one-command Compose smoke coverage, with strict claim boundaries.

`cmd/semops-feed-fixtures` is test infrastructure. It can serve deterministic OpenSky-compatible ADS-B JSON and a
representative SAPIENT task acknowledgement so the hosted `cmd/semops` runtime can exercise SemStreams component
lifecycle, port, payload-registry, NATS stream, graph writer, and COP readback paths without live network access.

This does not promote ADS-B or SAPIENT into default product feeds. ADS-B remains opt-in and OpenSky-compatible only.
SAPIENT remains preflight-only: raw input and decoded output stream, no graph projector, no owner claim, and no
conformance language.

## Risks

- A fixture provider can be mistaken for live feed support. Docs and smoke names must keep "local fixture" visible.
- Defaulting the stack smoke to enable ADS-B/SAPIENT increases Compose cost and diagnosis surface. Failures must stay
  isolated by test name: COP snapshot for ADS-B, decoded stream for SAPIENT.
- The fixture service has no distroless container healthcheck. The smoke actively polls `/healthz` from the host after
  Compose starts rather than relying on passive service start ordering.
- SAPIENT decoded-stream success does not prove product service behavior, sessions, tasking, projection ownership, or
  the Dstl harness.

## Evidence

- `cmd/semops-feed-fixtures` serves `/healthz`, `/adsb/states`, and `/sapient/messages`.
- `scripts/cop-stack-smoke.sh` enables `SEMOPS_ADSB_ENABLED=true` and `SEMOPS_SAPIENT_ENABLED=true` against local
  fixture URLs, while Compose keeps both runtime feeds disabled by default.
- `internal/smoke/cop` now verifies ADS-B HTTP component flow through the Caddy-routed COP snapshot.
- `internal/smoke/sapient` subscribes to `semops.feed.sapient.decoded` and decodes the payload with the registered
  SemStreams payload registry.
- The first full-stack run caught a SemOps lifecycle bug: hosted components inherited the startup/connect timeout
  context and stopped shortly after startup. `internal/app` now gives long-running components an app-owned runtime
  context that is canceled by `App.Close`, with a regression test proving component flow continues after startup
  context cancellation.

## Follow-ups

- Component-runtime health/flow metrics assertions for ADS-B/SAPIENT are now covered by
  `2026-06-21-component-prometheus-telemetry-review.md`; keep queue/drop/retry/backpressure metrics open as the next
  observability gate.
- Decide whether ADS-B next needs authenticated OpenSky mode, local receiver/readsb/dump1090 input components, or
  ASTERIX fixtures.
- Keep SAPIENT graph projection blocked behind ownership/indexing review and portable harness or official Dstl harness
  evidence.
