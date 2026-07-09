# CS API Command Impedance Review

Scope: future CS API tasking/control ingress and status egress

## Decision

Do not implement CS API tasking as a synchronous bridge from HTTP request to native tactical command. Treat CS API as
a standards-facing command interface over SemOps governed command intent, desired state, actual state, and status
evidence.

The CS API boundary should validate and accept or reject quickly, then persist command intent in the SemStreams graph.
Native drivers such as MAVLink, TAK/CoT, SAPIENT, or future protocol workers reconcile that intent into tactical
actions only after SemOps command authority and safety policy allow it.

## Adversarial Findings

- Direct HTTP-to-radio wiring couples a federated standards client to tactical-edge failure modes. Dropped links,
  stale vehicles, or slow acknowledgements must not leave upstream HTTP calls hanging.
- Desired state and actual state need different ownership and freshness rules. A standards command is not proof that
  a drone, sensor, or operator accepted the command.
- TTL/deadline windows are mandatory. Old accepted commands must not become live native actions after reconnect,
  replay, or delayed driver startup.
- Priority and authority arbitration must be explicit before two actors can task the same asset. Local operator
  override, upstream federation authority, emergency priority, and native safety lockout need deterministic outcomes.
- Idempotency, duplicate commands, cancellation, supersession, partial execution, and rejected execution need graph
  status semantics before any live command/control demo.
- CS API Command Status and System Event egress should be a projection over graph-backed command evidence, not a
  side-channel hidden inside the bridge.

## Required Before Implementation

- Command-intent or desired-state graph contracts with indexing profile, owner, and provenance rules.
- Authority and priority model for local SemOps operators versus upstream federated systems.
- TTL/deadline and stale-command rejection policy.
- Idempotency and correlation key handling for repeated upstream requests.
- Cancellation and supersession semantics.
- Native driver reconciliation contract for accepted, executing, succeeded, failed, rejected, timeout, partial, and
  link-loss states.
- Replay fixtures that cover conflict, stale, duplicate, cancellation, timeout, and partial execution cases.

## Open SemStreams Pressure

If multiple feeds need the same command-intent lifecycle, SemOps may need an upstream SemStreams pattern for governed
desired state, command status evidence, or action reconciliation. File that only after MAVLink or TAK tasking proves a
reusable contract rather than a SemOps-specific policy.
