# CAP Component Promotion Review

Date: 2026-06-20

## Decision

Promote the first hosted HTTP poller, raw-alert decoder, and born-first graph-projector boundary into a SemStreams
component package.

The current CAP slice is valid parser, projection, replay, readback, born-first append-evidence proof, and now a
deterministic local HTTP poller/decoder/projector component proof. It is still not a default live CAP feed service,
NWS/IPAWS integration, or CAP conformance claim. SemOps should keep default live-provider enablement, webhook
ingestion, vendor integration, and continuous alert-source health behind later product gates.

When promoted, the CAP boundary must follow SemStreams lifecycle rules:

- Pollers, webhooks, watched files, or vendor feed listeners are input components.
- CAP XML parsing and validation are processor components that emit decoded CAP payloads.
- CAP projection is a processor component that writes governed graph mutations through request ports.
- Raw and decoded CAP payloads are registered `message.BaseMessage` payloads.
- Component health, `DataFlow()`, Prometheus telemetry, lag, drop, retry, redelivery, and stale-source signals drive
  backpressure choices before SemOps adds local buffers, caches, or JetStream durability.

SemStreams `v1.0.0-beta.114` delivered `component.HTTPClientPort` for reusable external HTTP client and polling
metadata. A hosted CAP poller should use `HTTPClientPort` for method, URL pattern, auth reference, contact policy, and
interface metadata, plus a sibling `TimerPort` referenced by `TriggerPort` when cadence is timer-driven. SemOps filed
SemStreams issue #312 so the flowgraph can eventually classify `TimerPort` as a first-class cadence boundary instead
of leaving it stream-shaped. Webhook ingestion can use a network/request-facing input component, and captured alert
fixtures can use file input.

## Objections Raised

- The word "adapter" can hide that the current CAP path is in-process replay and smoke evidence, not hosted ingress.
- Wrapping deterministic fixtures alone in `internal/components/cap` would satisfy a checklist but teach the wrong
  lifecycle. The accepted component package is scoped to hosted HTTP polling, raw decoding, and governed graph
  projection; scenario replay remains separate.
- Polling NWS/IPAWS is not just parse logic. It has cadence, user-agent, cache, rate-limit, retry, timeout, stale
  source, retention, and audit behavior that should be visible in component config and telemetry.
- CAP remains append-only advisory evidence. Hosted ingestion must not turn CAP into authoritative hazard truth by
  accident.
- External HTTP polling is likely a reusable SemStreams framework need rather than a SemOps-only invention.

## Evidence Checked

- `pkg/adapters/cap` CAP 1.2 parser and local lifecycle fixtures.
- `internal/projectors/cap` born-first hazard append-evidence projection.
- `internal/components/cap` lifecycle HTTP poller, decoder, and projector with raw/decoded payload registration,
  `HTTPClientPort`, `TimerPort`, stream ports, graph request ports, config schema, health, and flow metrics.
- `internal/smoke/cap` skipped live graph smoke for born-first append-evidence behavior.
- `internal/scenario` CAP lifecycle replay through the current projector and graph writer.
- SemStreams `component` port inventory in `v1.0.0-beta.114`, including `HTTPClientPort`, `TimerPort`, `FilePort`,
  `NetworkPort`, `NATSPort`, `NATSRequestPort`, `JetStreamPort`, and KV ports.
- SemStreams issue #309 for component backpressure telemetry.
- SemStreams issue #310, now closed by the `HTTPClientPort` contract.

## Accepted Risks

- CAP remains structurally visible in the current Compose smoke through scenario replay and direct graph smoke rather
  than through a default hosted live CAP feed service.
- The first hosted CAP component chain now has opt-in product runtime wiring, but it still needs provider fixtures,
  stale-source behavior, and alert lifecycle review before SemOps claims hosted CAP service support.
- The external HTTP dependency now has a framework-visible port descriptor instead of living only in config schema;
  cadence visibility still needs SemStreams issue #312.

## Follow-Up Tasks

- Keep CAP parser/projector/readback gates independent of hosted ingress.
- Keep `internal/components/cap` scoped to hosted polling, decoding, and born-first projection, and keep hosted CAP
  disabled by default until provider fixtures and stale-source behavior are proven.
- Use `HTTPClientPort` plus `TimerPort` for hosted HTTP polling components and track SemStreams issue #312 for
  first-class timer/cadence flowgraph semantics.
- Keep CAP schema and consumer-rule validation separate from hosted service claims.
