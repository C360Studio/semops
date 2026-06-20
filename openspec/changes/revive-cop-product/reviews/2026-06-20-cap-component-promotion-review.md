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

SemStreams `v1.0.0-beta.114` delivered `component.HTTPClientPort` for reusable external HTTP client and polling
metadata. A hosted CAP poller should use `HTTPClientPort` for method, URL pattern, auth reference, contact policy, and
interface metadata, plus a sibling `TimerPort` referenced by `TriggerPort` when cadence is timer-driven. Webhook
ingestion can use a network/request-facing input component, and captured alert fixtures can use file input.

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
- SemStreams `component` port inventory in `v1.0.0-beta.114`, including `HTTPClientPort`, `TimerPort`, `FilePort`,
  `NetworkPort`, `NATSPort`, `NATSRequestPort`, `JetStreamPort`, and KV ports.
- SemStreams issue #309 for component backpressure telemetry.
- SemStreams issue #310, now closed by the `HTTPClientPort` contract.

## Accepted Risks

- CAP remains structurally visible in the current Compose smoke through scenario replay and direct graph smoke rather
  than through a hosted CAP feed service.
- If Phase 1 chooses live public-alert ingestion, a CAP component package becomes mandatory before SemOps claims
  hosted CAP service support.
- The first hosted CAP poller still needs runtime implementation, but the external HTTP dependency now has a
  framework-visible port descriptor instead of living only in config schema.

## Follow-Up Tasks

- Keep CAP parser/projector/readback gates independent of hosted ingress.
- Add `internal/components/cap` only when hosted polling, webhook, watched-file, or vendor feed input is in scope.
- Use `HTTPClientPort` plus `TimerPort` for hosted HTTP polling components and comment upstream only if the first
  concrete CAP, ADS-B, or SAPIENT design exposes a remaining framework gap.
- Keep CAP schema and consumer-rule validation separate from hosted service claims.
