# CAP Component Promotion Review

Date: 2026-06-20

## Decision

Do not promote the current CAP scenario replay and direct graph smoke into a SemStreams component package yet.

The current CAP slice is valid parser, projection, replay, readback, and born-first append-evidence proof. It is not a
hosted CAP feed service. SemOps should create CAP input and processor components only when Phase 1 or product scope
adds hosted polling, webhook ingestion, IPAWS/NWS/vendor integration, or continuous alert-source health.

When promoted, the CAP boundary must follow SemStreams lifecycle rules:

- Pollers, webhooks, watched files, or vendor feed listeners are input components.
- CAP XML parsing and validation are processor components that emit decoded CAP payloads.
- CAP projection is a processor component that writes governed graph mutations through request ports.
- Raw and decoded CAP payloads are registered `message.BaseMessage` payloads.
- Component health, `DataFlow()`, Prometheus telemetry, lag, drop, retry, redelivery, and stale-source signals drive
  backpressure choices before SemOps adds local buffers, caches, or JetStream durability.

SemStreams currently has `TimerPort`, `FilePort`, `NetworkPort`, NATS, JetStream, KV, and request ports, but no
first-class external HTTP client or polling port metadata. SemOps filed SemStreams issue #310 to track reusable
HTTP polling/client port metadata for CAP/NWS, OpenSky ADS-B, SAPIENT/Apex, and similar feeds. Until that exists, a
CAP poller can be modeled as `TimerPort` cadence plus endpoint/auth/cache/retry config and raw CAP stream output.
Webhook ingestion can use a network/request-facing input component, and captured alert fixtures can use file input.

## Objections Raised

- The word "adapter" can hide that the current CAP path is in-process replay and smoke evidence, not hosted ingress.
- Wrapping deterministic fixtures in `internal/components/cap` would satisfy a checklist but teach the wrong
  lifecycle. Components are needed when there is hosted feed behavior to manage.
- Polling NWS/IPAWS is not just parse logic. It has cadence, user-agent, cache, rate-limit, retry, timeout, stale
  source, retention, and audit behavior that should be visible in component config and telemetry.
- CAP remains append-only advisory evidence. Hosted ingestion must not turn CAP into authoritative hazard truth by
  accident.
- External HTTP polling is likely a reusable SemStreams framework need rather than a SemOps-only invention.

## Evidence Checked

- `pkg/adapters/cap` CAP 1.2 parser and local lifecycle fixtures.
- `internal/projectors/cap` born-first hazard append-evidence projection.
- `internal/smoke/cap` skipped live graph smoke for born-first append-evidence behavior.
- `internal/scenario` CAP lifecycle replay through the current projector and graph writer.
- SemStreams `component` port inventory in `v1.0.0-beta.113`, including `TimerPort`, `FilePort`, `NetworkPort`,
  `NATSPort`, `NATSRequestPort`, `JetStreamPort`, and KV ports.
- SemStreams issue #309 for component backpressure telemetry.
- SemStreams issue #310 for external HTTP polling/client port metadata.

## Accepted Risks

- CAP remains structurally visible in the current Compose smoke through scenario replay and direct graph smoke rather
  than through a hosted CAP feed service.
- If Phase 1 chooses live public-alert ingestion, a CAP component package becomes mandatory before SemOps claims
  hosted CAP service support.
- The interim `TimerPort` plus config-schema modeling is less expressive than a true SemStreams HTTP polling/client
  port, but it keeps cadence, config, health, and raw output visible.

## Follow-Up Tasks

- Keep CAP parser/projector/readback gates independent of hosted ingress.
- Add `internal/components/cap` only when hosted polling, webhook, watched-file, or vendor feed input is in scope.
- Comment back on SemStreams issue #310 after CAP, ADS-B, or SAPIENT produces a concrete component design that proves
  the reusable port metadata shape.
- Keep CAP schema and consumer-rule validation separate from hosted service claims.
