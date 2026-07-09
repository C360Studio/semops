# Scenario Control Guard Review

Date: 2026-06-26

Scope: scenario start/reset/pause/resume boundary after manifest-backed checkpoint evaluation.

## Decision

Add a fail-closed scenario control guard before adding any executable scenario controls or UI buttons.

The browser-facing stack may expose `GET /scenario/controls` so operators, tests, and reviewers can see that
start/reset/pause/resume are intentionally blocked. `POST /scenario/controls` may accept a supported action name, but
it must reject the request with policy evidence until SemOps has a reviewed `operator_scenario_control` checkpoint and
an implemented control executor.

## Objections

1. Visible scenario controls can imply operator workflow maturity that the product has not earned.
2. Reset/replay can become a hidden direct-injection path if it bypasses feed-boundary product evidence.
3. Pause/resume semantics can create false confidence unless expected COP state, runtime flow, and checkpoint
   readback stay deterministic.
4. A control endpoint without authentication or authority arbitration would be confused with command-control or
   mission execution authority.

## Evidence Checked

- Product scenario status now exposes checkpoint evaluations and only reports product checkpoint readiness from
  feed-boundary, zero-mutation runner evidence.
- The one-command stack smoke loads the checkpoint manifest and verifies COP snapshot, runtime feed, and Prometheus
  readback through Caddy after scenario status reports product readiness.
- The UI still renders scenario status as read-only evidence.
- No `operator_scenario_control` checkpoint exists in the Phase 1 HADR manifest.

## Accepted Boundary

- `GET /scenario/controls` is allowed as a visible guardrail.
- `POST /scenario/controls` rejects supported actions with `accepted=false`.
- The Caddy-routed endpoint is smoke-tested as blocked.
- This guard does not satisfy operator scenario-control, command-control, CS API, provider, simulator, or standards
  claims.

## Follow-Ups

- Add a reviewed `operator_scenario_control` checkpoint before any executable control action.
- Add authentication and multi-authority conflict policy before controls affect graph state, scenario state, command
  execution, upstream CS API status, or compliance workflows.
- Keep UI buttons absent until a UX review proves the operator job and failure handling.
