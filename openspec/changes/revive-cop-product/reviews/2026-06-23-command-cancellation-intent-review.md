# Command Cancellation Intent Review

Scope: pure cancellation-intent helper before CS API, UI, or native cancellation handlers

## Decision

Accept a pure cancellation helper that transforms an active current command intent into a `cancel_requested` update on
the same command entity. It preserves the original target and command kind, records the cancellation request authority,
priority, expiry, correlation, idempotency, requester, provenance, and desired cancel state, and relies on the existing
command projector for born-first graph mutation planning.

## Adversarial Findings

- Cancellation must target the existing command entity. Creating a sibling "cancel command" would make native
  executors choose between two tasks and risks losing the relationship to the original desired state.
- Cancellation is a request, not proof of cancellation. `cancel_requested` must not be advertised as native
  `cancelled` until a native driver or status reconciler observes accepted cancellation or a timeout/failure path.
- Identity checks matter. A cancellation request with a different native ID or target asset is rejected before graph
  planning.
- Terminal commands stay terminal. A succeeded, failed, cancelled, rejected, expired, duplicate, superseded, or timed
  out command cannot be reactivated into `cancel_requested`.

## Acceptance

- Active command intents can produce a `cancel_requested` update without repeating the strict target edge.
- Cancellation desired state is deterministic JSON with the original command native ID and optional reason.
- Cancellation request authority, priority, correlation, idempotency, requester, and provenance replace the control
  fields for the update.
- Terminal current commands and mismatched identity are rejected.

## Result

Accepted as a pre-handler cancellation seam. Keep CS API endpoints, UI controls, native cancellation transmit, final
`cancelled` status, timeout, link-loss, and partial execution evidence behind later reviews.
