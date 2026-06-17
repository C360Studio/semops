# MAVLink Replay Store Adversarial Review

Date: 2026-06-17
Scope: `COP-004` durable MAVLink replay fixtures in `pkg/adapters/mavlink`

## Finding Summary

- Severity: Medium. The replay store persists raw-frame records, but it is not yet wired into the scenario runner or
  container stack retention policy.
- Severity: Medium. JSON Lines fixtures are inspectable and deterministic, but they are not a high-volume binary
  storage answer for KLV/video. KLV remains a separate memory-bound proof spike.
- Severity: Medium. Replay fixtures prove native frame bytes can be loaded and parsed. They do not yet prove
  deterministic graph state after a full scenario replay.
- Severity: Low. Missing replay files load as empty fixtures, which is useful for optional fixtures but must not be
  confused with a successful required demo replay.

## Role Review

architect:

- Accepts `pkg/adapters/mavlink.ReplayStore` as a library artifact behind the raw lane.
- Requires scenario-runner integration to treat required fixtures differently from optional fixtures.

go-reviewer:

- Accepts append/load tests that round-trip raw frame bytes and parse loaded frames with the active parser.
- Requires future scenario tests to verify graph state and health counters after replay, not only file readability.

technical-writer:

- Requires docs to call this durable fixture storage, not complete replay behavior.
- Requires KLV and video claims to remain blocked until binary-by-reference handling is proven separately.

## Decision

Accept the replay store as a COP-004 storage increment. Keep scenario-runner playback, graph-state polling, retention
policy, and binary-media proof work open.
