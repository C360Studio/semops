# MAVLink Command ACK Readback Review

Date: 2026-06-23

Scope:

- Add a SemStreams ownership contract for MAVLink command readback state.
- Project `COMMAND_ACK` packets as `control`-profiled COP task entities.
- Keep outbound command/control, CS API tasking reconciliation, command priority, TTL windows, and safety interlocks
  out of scope.

## Findings

1. ACK projection is useful, but it is not command authority.
   The new `semops.mavlink.command_task.v1` contract records observed MAVLink `COMMAND_ACK` packets as task-state
   readback evidence. It does not prove SemOps can send a command, arbitrate local versus upstream authority, retry
   safely, or verify mission execution.

2. Born-first discipline still applies to command state.
   A command task writes a strict `cop.task.target` foreign edge to the MAVLink source asset. The projector must birth
   the source asset first and must not repeat that strict edge on later task updates. This mirrors the existing
   `cop.track.source` discipline and avoids reintroducing auto-vivify assumptions under a different entity type.

3. The current task identity is an MVP current-state key.
   `system + command + target` is adequate for command readback smoke evidence, but it is not enough for concurrent
   repeated commands, command nonce correlation, upstream CS API task IDs, or local operator overrides. The outbound
   command slice needs an explicit SemOps command-intent identity and reconciliation model before product claims.

4. Index pressure is deliberate.
   ACK readback belongs in `control`, not `signal`, because operators and bridges will inspect command lifecycle state.
   Raw command frames stay in the bounded raw lane through `cop.provenance.source_ref`; SemOps still must not create a
   graph entity per packet.

## Decision

Accept the COMMAND_ACK readback projection as a narrow MAVLink command-lifecycle evidence step. Keep live command
transmit, command priority, TTL expiry, CS API desired/actual reconciliation, and native safety interlocks open for a
separate design review before any SITL command-control demo claim.

## Verification

- `go test ./pkg/cop`
- `go test ./internal/projectors/mavlink`
- `go test ./internal/adapters/mavlink`
- `go test ./internal/components/mavlink ./internal/copownership`

