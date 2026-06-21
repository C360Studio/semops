# Component Prometheus Telemetry Review

## Verdict

Accept the initial hosted component telemetry slice.

SemOps now exports running feed component health and flow through Prometheus rather than a bespoke telemetry API. The
collector samples SemStreams `component.Discoverable` `Health()` and `DataFlow()` values, registers into a
SemStreams `metric.MetricsRegistry`, and is exposed by `cmd/semops` at `/metrics`. Caddy proxies `/metrics`, so the
one-command stack smoke verifies the same ingress shape operators and dashboards will use in dev.

## Evidence

- `internal/componentmetrics` exports `semops_component_health_status`,
  `semops_component_flow_messages_per_second`, `semops_component_flow_bytes_per_second`,
  `semops_component_flow_error_rate`, and last-activity/uptime metadata.
- `internal/app.App.ComponentMetricSources()` exposes only started runtime component instances, preserving the
  SemStreams component lifecycle boundary.
- `scripts/cop-stack-smoke.sh` waits on direct and Caddy-routed SemOps metrics and passes the Caddy URL into the
  hosted COP smoke.
- `TestHostedCOPComponentPrometheusMetricsReflectFeedFlow` stimulates MAVLink and CoT UDP input and asserts
  component health, message flow, and last activity for MAVLink, TAK/CoT, ADS-B, and SAPIENT when enabled.

## Adversarial Notes

- Architect: This is observability, not orchestration. It should not become a COP shell or runtime-control surface
  without a separate operator-value review.
- Go reviewer: The local collector is intentionally thin. If another product needs this shape, SemStreams should own
  a reusable Discoverable-to-Prometheus helper.
- Svelte reviewer: The UI should not grow a raw Prometheus-scraping browser panel by default. A UI view can be backed
  by Prometheus or a small derived status facade after we decide which operator question it answers.
- Technical writer: This closes first-order hosted component flow evidence, not full backpressure evidence. Queue
  depth, drop counts, retry/redelivery pressure, lag, and stale-source policy still gate JetStream, `pkg/buffer`, or
  `pkg/cache` promotion.
