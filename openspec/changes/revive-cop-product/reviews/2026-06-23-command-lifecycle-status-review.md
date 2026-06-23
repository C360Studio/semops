# Command Lifecycle Status Review

Scope: command-intent status vocabulary and transition guard before CS API, UI, or native-driver status handlers

## Decision

Accept a small command lifecycle vocabulary and pure transition validator. The command planner now rejects unknown
status strings, arbitration uses the shared terminal-status classifier, and native-execution eligibility is limited to
accepted intents.

## Adversarial Findings

- Open status strings are a future interop trap. A CS API bridge could emit `approved`, a UI could emit `ready`, and a
  native driver could emit `done`, leaving downstream consumers unable to reconcile command state.
- Terminal statuses must be sticky. A command that is succeeded, failed, cancelled, rejected, expired, duplicate,
  superseded, or timed out must not become active again after reconnect or replay.
- `executing` is not a valid initial native candidate. A command must pass admission/arbitration and become accepted
  before a native driver can reconcile execution.
- This is status grammar, not full lifecycle execution. Cancellation requests, native ACK reconciliation, timeout
  timers, link loss, and partial execution still need graph-backed fixtures and handler reviews.

## Acceptance

- Unknown status strings are rejected by the command-intent planner.
- Empty status normalizes to `requested`.
- Terminal-status detection is centralized for arbitration and future handlers.
- Transition validation allows requested-to-accepted, accepted-to-executing, cancel-requested-to-cancelled, and
  terminal idempotency.
- Transition validation rejects requested-to-executing, executing-to-accepted, terminal-to-active, and unknown target
  status values.

## Result

Accepted as a pre-handler lifecycle guard. Keep CS API request handling, UI controls, native status reconciliation,
cancellation endpoints, safety interlocks, and live transmit behind later reviews.
