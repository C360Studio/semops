# UI Inventory Review

Date: 2026-06-17

Change: `revive-cop-product`

Status: complete for the current checkout and reachable Git history.

## Decision

Start the SemOps COP UI from scratch.

Do not attempt to restore a flow-runtime control UI as the product surface. If older SemOps UI artifacts exist outside
this checkout, treat them as historical reference only unless they contain reusable fixtures, copy, or operator
workflow notes.

## Evidence Checked

- Current SemOps tree contains no `ui`, `web`, `frontend`, `dashboard`, `src`, `package.json`, Vite, Svelte, React,
  or TypeScript frontend files.
- Reachable Git history contains no deleted frontend assets matching common UI paths or package manifests.
- `configs/robotics-flow.json` preserves the old flow-runtime model: component IDs, raw subjects, processors,
  indexers, and a SOSA API service.
- `README.md` frames SemOps around StreamKit flow configuration, robotics adapters, SOSA APIs, and backend module
  migration.
- `cmd/semops/main.go` is a backend lifecycle stub with TODOs for configuration, SemStreams clients, adapters, API,
  and monitoring.

## Interpretation

The old product center of gravity was operational flow plumbing: users could reason about inputs, processors,
subjects, indexers, and service endpoints. That was useful during early SemStreams exploration, but it is not the
right first screen for a data-fusion COP.

Operators should first see the situation: tracks, assets, hazards, alerts, tasks, source health, provenance, and
staleness. Flow topology and runtime controls may still exist for administrators or demo operators, but they are not
accepted product features until an adversarial UX review proves they answer a concrete operator question.

## Salvage

Potentially useful:

- The old flow config as a migration clue for adapter/process/index/API expectations.
- Service and component names as hints for backend responsibilities.
- Any later-discovered screenshots or notes as anti-regression examples of what confused operators.

Do not salvage:

- Flow-control UI as the primary COP shell.
- Component topology as the default first viewport.
- Runtime tuning controls before source health, provenance, and stale-data behavior exist.

## Required Follow-Up

1. Keep `COP-005` as a clean-sheet Svelte 5 COP surface.
2. Add fixture snapshot tests for map, source lens, provenance lens, and alert state before UI implementation.
3. Require adversarial UX review before adding topology, tier, orchestration, or flow-control panels.
4. If old UI artifacts are found outside this repo, inventory them without adopting their structure by default.
