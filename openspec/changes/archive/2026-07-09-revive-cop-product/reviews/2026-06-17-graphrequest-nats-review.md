# SemStreams NATS Graph Requester Adversarial Review

Date: 2026-06-17
Scope: `COP-004` retry-aware NATS requester in `internal/graphrequest`

## Finding Summary

- Severity: Medium. The requester preserves SemStreams mutation retry behavior, but it is still tested with a fake
  requester. Live NATS/SemStreams responder wiring remains open.
- Severity: Medium. Retry is appropriate for convergent graph mutations, not for query paths. Do not reuse this
  requester for read-side polling or UI snapshots.
- Severity: Medium. Retrying create-with-triples can surface `entity_already_exists` after a lost response. Later
  MAVLink work added narrow create-conflict reconciliation for known asset/track births; durable checkpoint/read-back
  remains required before production adapter claims.
- Severity: Low. The requester passes SemStreams' default retry config by default and allows an override for tests or
  deployment tuning.

## Role Review

architect:

- Accepts the requester as a transport adapter, not a new graph abstraction.
- Requires the structural stack to instantiate it with a real `natsclient.Client` before claiming live graph writes.

go-reviewer:

- Accepts tests for timeout, subject, payload, context, retry config, error propagation, and nil-client behavior.
- Requires future live smoke tests to verify graph state after writes, not only successful request calls.

technical-writer:

- Requires docs to state this is mutation request/reply wiring, not query/polling infrastructure.
- Requires durable checkpoint/read-back reconciliation to stay open because retry can expose duplicate birth attempts.

## Decision

Accept the NATS requester boundary. Keep live NATS/SemStreams stack wiring, durable restart reconciliation, and
read-side polling implementation open.
