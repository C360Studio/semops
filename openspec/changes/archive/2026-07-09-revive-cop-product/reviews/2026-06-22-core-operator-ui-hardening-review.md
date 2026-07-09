# Core Operator UI Hardening Review

Scope: Source cards, runtime-flow readout, keyboard selection, narrow viewport behavior, and topology/tier UI gate.

## Decision

Accept the first UI hardening slice. `SourceCard.svelte` owns the source-card rendering boundary, `sourceHealth.ts`
owns snapshot/runtime/discovery merge logic, and Playwright now covers the core operator loop across desktop and narrow
viewport paths.

Reject topology, tier, orchestration, and flow-control panels for the MVP. They remain deferred until a review proves a
specific operator decision that source health, runtime flow, provenance, alerts, and scenario state do not answer.

## Findings

### High: Source-card state had too much page-local logic

The page was combining snapshot feed state, runtime flow, source messages, and discovery diagnostics inline. That made
it too easy for later feed work to regress source/provenance wording or runtime-only feed boundaries.

Resolution: extract a source-card component and pure merge helpers. Unit/SSR tests now cover runtime-only SAPIENT,
stale state, truncation chips, and source-message preservation.

### High: Topology controls are still a footgun

Runtime flow evidence made components product-visible, but that does not create an operator need for topology editing
or component controls. Exposing flow control now would invite accidental framework-building inside SemOps and distract
from feed credibility.

Resolution: keep runtime flow read-only and explicitly defer topology/tier/orchestration UI for MVP.

### Medium: Narrow viewport and keyboard paths needed a hard gate

The first Playwright smoke proved desktop rendering and selection, but not whether the operator loop remained usable
when panels stacked or when map/entity controls were activated by keyboard.

Resolution: add a narrow viewport browser check for source cards, map controls, keyboard entity selection, alert
selection, provenance, and horizontal overflow.

### Low: Component testing can stay lightweight for now

Adding a full DOM component-test library would be premature for the current surface. Server-render tests plus helper
tests catch the current risk without expanding test infrastructure.

Resolution: use Vitest with Svelte server rendering for `SourceCard.svelte` and pure tests for source-health helpers.

## Verification Expectations

- `npm run check`
- `npm run test`
- `npm run test:e2e`
- `npm run build`
- `openspec validate revive-cop-product --strict`

## Follow-Ups

- Add a real accessibility audit tool only when the app has enough UI surface to justify the dependency.
- Revisit topology/tier UI only after an operator workflow cannot be answered by source health, provenance, alert
  state, scenario state, or a focused diagnostic graph lens.
