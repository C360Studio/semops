# Weather Hosted Runtime Review

Scope: opt-in hosted weather fixture input -> decoder -> graph-projector runtime wiring.

## Decision

Accept the weather hosted runtime only as local point-forecast fixture evidence. The runtime registers the
`semops.feed.weather` owner before composing the SemStreams file input, decoder, and projector components, uses
registered payloads and declared ports, and exposes component health/flow metrics when enabled.

This does not promote weather to a live provider, OGC EDR conformance target, cache/stale policy, tactical UI layer,
spatial weather payload, or route-safety authority.

## Findings

### High: Fixture hosting can be mistaken for live weather support

The committed Open-Meteo-shaped fixture is useful for proving storage, ownership, projection, and runtime composition.
It is not evidence of live Open-Meteo, NWS, MSC GeoMet, or radar/tile reliability.

Resolution: keep `SEMOPS_WEATHER_ENABLED=false` by default in Compose and describe the runtime as fixture-backed
point-forecast evidence only.

### High: Freshness metadata is not a provider stale-data policy

The projector has a freshness window for deciding whether projected observations are fresh enough to write, but that
is not equivalent to provider-contact state, rate-limit handling, cache invalidation, or operator-visible stale-source
policy.

Resolution: keep cache/stale policy as a later live-provider gate.

### Medium: Spatial EDR fixtures are parser evidence, not runtime evidence

Area, trajectory, and corridor fixtures prove parser preflight shape. The hosted runtime currently promotes only
point-forecast payloads.

Resolution: require spatial runtime payloads, graph projection, and UI semantics before route-weather or standards
interop claims exceed point retrieval.

### Medium: Stack smoke is still missing for weather

Unit and app-runtime tests prove composition in process. The one-command stack does not yet opt into weather and read
the projected observations back through Caddy.

Resolution: add stack smoke only after the product wants weather visible in the operator COP or runtime source cards.

## Verification

- `go test ./internal/stack`
- `go test ./internal/app`
- `go test ./internal/components/weather`

## Follow-Ups

- Add live-provider poller and provider-contact health only after source/cache policy is accepted.
- Add spatial runtime payloads before route-weather UI or safety logic.
- Add Caddy-routed graph readback smoke before weather becomes demo-visible by default.
