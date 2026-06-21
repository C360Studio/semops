# Phase 1 Structural Demo Adversarial Review

Date: 2026-06-21
Scope: Demo credibility, monitoring, and graph/index cardinality after hosted component-flow, Playwright, active status
polling, and shared-airspace smoke slices

## Decision

Accept the current stack as credible internal MVP evidence for a local, fixture-backed, governed COP structural demo.

Do not describe it as live-feed production readiness, standards conformance, automated deconfliction, or Phase 1
external signoff yet. The stack now proves a meaningful SemStreams-as-framework story: SemOps starts through Compose,
uses SemStreams ownership/born-first graph writes, composes feed boundaries as input/processor components, serves a
Caddy-routed COP snapshot/UI, actively polls scenario state, and can show HADR ground/civil-alert context plus ADS-B
air-picture state in one operator snapshot.

## Findings

### High: Full stack evidence is still local-fixture evidence

The one-command smoke now covers MAVLink UDP, TAK/CoT UDP, CAP graph evidence, ADS-B HTTP fixture readback, SAPIENT
decoded-stream preflight, same-origin Caddy/UI/API plumbing, and shared-airspace snapshot coexistence. That is enough
for internal MVP confidence.

It is not live OpenSky reliability, live NWS/IPAWS behavior, TAK server behavior, SAPIENT conformance, PX4/SITL
fidelity, KLV binary streaming, or CS API bridge evidence. Public language must keep "fixture-backed", "opt-in", and
"component-flow evidence" visible.

### High: Ownership boundaries are good, but easy to regress

The ADS-B shared-airspace path deliberately uses the hosted ADS-B component flow, not a second scenario-runner
`semops.feed.adsb` writer. Keep that boundary. Two independent processes claiming the same strict owner would create
the exact ambiguity the owner-token/born-first migration was meant to remove.

Future scenario-runner ADS-B replay should either run with the hosted ADS-B flow disabled or use a deliberately
separate owner/source contract after review.

### Medium: Monitoring improved, but component telemetry is not yet asserted

The smoke now actively polls `/scenario/status`, detects failure, detects stale progress, and dumps Compose diagnostics.
That closes the biggest operational blind spot.

We still do not assert SemStreams component `Health()`/`DataFlow()` metrics or Prometheus counters for feed throughput,
lag, drop pressure, retry pressure, or stale source behavior across all hosted components. The stack can fail faster,
but it does not yet tell us whether high-rate operation needs JetStream ports, `pkg/buffer`, or `pkg/cache`.

Update: `2026-06-21-component-prometheus-telemetry-review.md` closes the first-order hosted component health/flow gap
with Caddy-routed `semops_component_*` Prometheus metrics and stack-smoke assertions. Queue depth, drop pressure,
retry/redelivery pressure, and high-rate backpressure evidence remain open.

### Medium: Prefix discovery is working, but cardinality pressure is only first-order

The COP API uses SemStreams prefix discovery and surfaces source/type cardinality plus at-limit pressure. That is the
right architectural shape for the SemStreams index profile stress test.

The remaining risk is mixed-feed scale: duplicate tracks, many CAP lifecycle/evidence records, noisy CoT chat, and
future SAPIENT/KLV detections can all create bursty cardinality. Current evidence proves bounded fixture volumes, not
large operational volumes. Keep `indexing_profile` choices explicit and treat every new projected entity family as an
index/cardinality review point.

### Medium: Browser coverage now matches the plumbing baseline

The Playwright smoke catches API/UI drift for ADS-B feed/discovery readback and selection provenance. This is the
right start.

It does not yet cover the live Caddy stack in-browser, accessibility, mobile sizing, alert interactions, or high-rate
map update behavior. UI claims should remain "first usable COP surface" rather than mature operator workstation.

### Low: 6.2 wording is stale relative to implementation

The task "Add MAVLink, TAK/CoT, and CAP/EDXL structural adapters" remains open, but the repo now has concrete
MAVLink, TAK/CoT, and CAP component/projector paths plus live graph smokes. Clarify whether "EDXL" means broader EDXL
family work beyond CAP before closing or splitting it.

## Required Follow-Ups

- Run the full `bash scripts/cop-stack-smoke.sh` on a healthy Docker window before any external demo claim.
- Add component telemetry assertions once SemStreams exposes enough hosted flow metrics for reliable checks.
- Decide whether task `6.2` should close as complete for MAVLink/TAK/CAP or split into a broader EDXL follow-up.
- Keep SAPIENT graph projection blocked behind portable harness or Dstl harness qualification, ownership/indexing
  review, and product-support decision.
- Keep KLV/STANAG 4609 as a proof spike until binary fixture extraction and memory-bounded streaming behavior are
  demonstrated.
- Keep CS API command/control work behind the command-impedance gate: TTL, priority, authority, local override,
  idempotency, replay, and async status reconciliation.

## Verdict

Proceed with the MVP build. The current stack is strong enough to continue adding operator and feed depth, but not
strong enough for uncaveated external conformance or live-feed claims.
