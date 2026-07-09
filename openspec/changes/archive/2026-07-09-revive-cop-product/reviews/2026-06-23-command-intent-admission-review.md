# Command Intent Admission Review

Date: 2026-06-23

Scope:

- Add a guarded command-intent projection path before CS API ingress, local UI controls, or native transmitters.
- Reject unresolved target assets, expired intents, and duplicate idempotency keys before producing mutation plans.
- Keep durable distributed idempotency, priority arbitration, local override, and native execution out of scope.

## Findings

1. Target resolution must happen before graph writes.
   The command-intent planner writes a strict `cop.task.target` edge. Admission must prove the target asset already
   exists or return no mutation plan; target auto-vivify would recreate the SemStreams must-exist failure mode.

2. Expiry is an admission rule, not just a graph field.
   A stale command with a valid `expires_at` triple is still dangerous if it can pass through after reconnect or replay.
   The guard checks wall clock before planning any mutation.

3. Duplicate idempotency must collapse before writes.
   The in-memory proof reserves idempotency keys and returns no mutation plan for duplicates. A production CS API or
   operator ingress path still needs a durable store or graph-backed idempotency strategy before distributed use.

4. This does not decide authority.
   The guard validates that authority and priority are present, but it does not arbitrate local operator override,
   upstream federation priority, cancellation, supersession, or native safety lockout.

## Decision

Accept the admission guard as a local executable gate for the command-intent impedance model. Do not wire CS API
tasking, UI controls, or native transmit until durable idempotency, target lookup source, authority arbitration,
cancellation/supersession, and async status reconciliation are designed and covered by fixtures.

## Verification

- `go test ./internal/projectors/command`
