# KLV Hosted Runtime Review

Scope: opt-in hosted KLV media-reference input, demux, MISB decode, and projector composition.

## Decision

Accept the first hosted KLV runtime composition as an opt-in local-media proof path.

The hosted app now wires a SemStreams component flow:

1. KLV media-reference input.
2. KLV FFmpeg/ffprobe demux processor.
3. MISB ST 0601 decode processor.
4. KLV governed graph projector.

KLV remains disabled by default. Enabling it registers the KLV `sensor_footprint` ownership contract, starts the
projector/decoder/demux subscriptions before the media-ref input publishes, and exposes each stage through the same
component health/flow source used by the Prometheus and COP runtime facades.

## Adversarial Notes

### High: Local media reference is not live media ingress

The media-ref input scans a configured local path/glob once and publishes file references. This is enough for a local
fixture/runtime proof, but it is not a directory watcher, object-storage notification consumer, RTP/RTSP receiver, or
video relay.

Resolution: keep `SEMOPS_KLV_ENABLED=false` by default and keep live media receiver/service work behind a separate
product and security review.

### High: Runtime wiring is not STANAG 4609 conformance

The flow can project decoded MISB ST 0601 subset frames into governed graph state, but official STANAG 4609
conformance remains blocked until validated fixtures or a funded lab/validator path exists.

Resolution: use "engineering support for the tested subset" language only.

### Medium: Binary bounds are still FFmpeg/path dependent

The demux stage has max extract bytes, max packet bytes, max packet count, materialization bytes, and bounded probe
output, but a configured local file can still be large and slow to process.

Resolution: keep opt-in runtime tests free of large media, keep public-sample smoke local/provenance-gated, and add
operator/runtime alerts before live media is enabled.

### Medium: Storage-reference materialization is not hosted yet

The demux and decoder components support bounded materializer interfaces, but the hosted runtime currently wires only
local file references.

Resolution: treat SemSource/object-storage materialization as a later integration gate.

## Verification

- `go test ./internal/components/klv`
- `go test ./internal/stack`
- `go test ./internal/app`

## Follow-Ups

- Add a one-command opt-in KLV stack smoke only when the fixture story is legally and operationally clean.
- Add live media ingress and storage materializer design review before enabling non-local media references.
- Keep footprint polygons, thumbnails/video player, and CS API egress for KLV under separate reviews.
