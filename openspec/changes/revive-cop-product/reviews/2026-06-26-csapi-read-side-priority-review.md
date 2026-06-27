# CS API Read-Side Priority Review

Scope: first SemOps CS API interop artifact after the MVP read-side priority decision.

## Decision

Accept a read-side CS API egress-first path for the MVP. SemOps can prepare CS API-shaped Systems, Datastreams,
Observations, Deployments, and System Events from the COP snapshot model, while SemConnect remains the conformance
anchor and runtime gateway. Write-side CS API ingress, Command, ControlStream, and native tasking remain stretch goals.

## Findings

1. Bidirectional language was too easy to overread.
   The bridge remains a long-term product goal, but MVP evidence should say read-side egress unless a real ingress
   adapter is in the path.

2. Read-side egress has useful demo value without command authority.
   A standards-facing consumer can inspect SemOps current state without SemOps accepting upstream tasking or driving a
   native transmitter.

3. Command status is still a trap.
   Existing command-intent and ACK/readback work is useful groundwork, but mapping it to CS API Command Status would
   imply a stronger tasking lifecycle than the MVP needs. Keep Command and ControlStream surfaces deferred.

4. SemConnect should stay the conformance anchor.
   SemOps should produce governed state and deterministic read models. SemConnect should carry the CS API gateway and
   harness unless SemOps is explicitly rechartered.

## Accepted Scope

- Internal read-side mapping from COP snapshot state to CS API-shaped read resources.
- Systems, Datastreams, Observations, Deployments, and System Events only.
- Provenance, source refs, observed times, and SemOps entity IDs preserved for later bridge/debug use.
- Deferred-surface evidence naming write-side ingress and command/control as stretch scope.

## Rejected Scope

- CS API HTTP server inside SemOps.
- CS API source ingress adapter.
- Command, ControlStream, Command Status, or native tasking behavior.
- SemConnect conformance claims from SemOps unit tests alone.

## Next Gate

Wire the read model to a deterministic SemConnect fixture or bridge test after the resource family mapping stabilizes.
The SemConnect ETS harness remains the only conformance acceptance gate.
