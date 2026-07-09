# Runtime Feed Flow UI Review

Scope: `GET /api/cop/runtime` and source-card runtime flow readout.

## Decision

Accept a read-only COP runtime facade derived from the same SemStreams component `Health()` and `DataFlow()` sources
used by Prometheus. The UI may merge that facade into source cards as compact source-health evidence.

This does not approve a topology panel, flow editor, component control surface, browser NATS client, or orchestration
shell.

## Findings

### High: Runtime flow must not become an alternate telemetry standard

SemOps already exposes Prometheus metrics for component health and flow. The browser endpoint is useful because source
cards need a small view model, but it must remain derived from component discovery rather than becoming a second metric
collection framework.

Resolution: `GET /api/cop/runtime` reads the same `ComponentMetricSources()` provider as the Prometheus collector and
returns feed/component view-model JSON only.

### Medium: Runtime-only feeds can overclaim product support

SAPIENT currently has preflight component flow but no graph projection, owner claim, or conformance claim. Showing it
in the UI is valuable as framework evidence, but it could be mistaken for a fully supported COP feed.

Resolution: runtime-only rows use `component-flow` state and component health/throughput language. They do not add
graph entities, map layers, or product feed claims.

### Medium: Stale source state must be visible

If stale components are also unhealthy, a generic degraded label can hide the more actionable source condition.

Resolution: feed rollup prioritizes explicit `stale` component status before generic degraded status.

### Low: Browser fallback should degrade softly

Local UI development can use fixture snapshot fallback without a running SemOps API. Runtime absence should not make
the first screen empty.

Resolution: runtime loading returns `null` on failure and the page still renders snapshot or fixture state.

## Verification Expectations

- Go tests cover runtime rollup and handler JSON.
- Vitest covers runtime client success/failure and compact rate formatting.
- Playwright covers source cards with ADS-B graph/discovery state plus component flow, and SAPIENT runtime-only flow.
- Stack smoke asserts both Prometheus `semops_component_*` samples as the operational metrics gate and the
  Caddy-routed `/api/cop/runtime` facade as the browser contract gate.

## Follow-Ups

- Add queue depth, drop/backpressure, and buffer/cache pressure once SemStreams exposes a stable component telemetry
  contract for those signals.
- Add operator wording review after source-health state gets noisy under larger mixed-feed loads.
