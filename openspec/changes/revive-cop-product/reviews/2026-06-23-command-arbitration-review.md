# Command Arbitration Review

Scope: `internal/projectors/command` authority and priority policy before CS API ingress or native command transmit

## Decision

Accept a pure deterministic arbitration layer for command intent. It ranks already-admitted active intents per target
by local override, authority rank, priority, observation time, and native ID, then returns accepted, superseded, or
ignored decisions. It does not write graph state or transmit native commands.

## Adversarial Findings

- Local override must not be an HTTP-handler special case. If CS API, UI, replay, and future automation each encode
  their own policy, two ingress paths can reach different answers for the same target.
- Priority without authority is unsafe. A remote or automated priority 100 command must not silently beat a local
  operator command because a standards client chose a larger number.
- Arbitration still is not execution authority. Accepted means "eligible for native reconciliation"; it does not mean
  a MAVLink, TAK, SAPIENT, or DJI driver has transmitted or that the platform accepted the action.
- Superseded decisions need graph status follow-up before the product claims complete command lifecycle semantics.
  Returning the decision from a pure policy layer is only the first gate.

## Acceptance

- Local operator authority wins over a higher-priority upstream federated command for the same target.
- Higher authority rank wins before priority for non-local commands.
- Higher priority wins within the same authority.
- Independent targets each get their own accepted candidate.
- Observation time and native ID make tie breaks deterministic.
- Terminal command intents are ignored and not exposed as native execution candidates.

## Result

Accepted as a pre-transmit policy gate. Keep CS API handlers, UI controls, graph status writes, cancellation,
supersession lifecycle, native safety interlocks, and live command transmit behind later reviews.
