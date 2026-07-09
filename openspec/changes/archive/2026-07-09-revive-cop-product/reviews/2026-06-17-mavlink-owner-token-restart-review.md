# MAVLink OwnerToken And Restart Reconciliation Review

Date: 2026-06-17

Scope:

- `internal/copownership`
- `internal/projectors/mavlink`
- `internal/adapters/mavlink`
- `internal/stack`
- `internal/app`
- `internal/smoke/mavlink`

Evidence:

- SemStreams changed `projection.BindAndHeartbeat` to return typed `ownership.OwnerToken`.
- SemOps migrated `copownership.BindingResult` to carry typed tokens per owner.
- Hosted runtime now passes the token map into the MAVLink stack; projectors call `OwnerToken.Wire()` only when
  building graph mutation requests.
- MAVLink projection now proposes plans without committing birth state. Births are marked only after graph write
  success or after a strict restart reconciliation conflict.
- Adapter reconciliation handles SemStreams `entity_already_exists` on create mutations for the current packet's known
  asset or track ID, marks that entity born, reprojects, and retries.

Verification:

- `go test ./internal/copownership ./internal/projectors/mavlink ./internal/adapters/mavlink ./internal/stack ./internal/app`
- `go test ./... -count=1`
- `go build -o /private/tmp/semops-build ./cmd/semops`
- `bash -n scripts/cop-stack-smoke.sh`
- `docker compose -f compose.cop.yml config`
- `bash scripts/cop-stack-smoke.sh`

## Adversarial Findings

- Architect: Pass. Token composition no longer leaks into SemOps config. The remaining wire string assertions are at
  the graph request boundary, where serialization is expected.
- Go reviewer: Pass with a durability caveat. Projection is now pure until commit, which prevents failed creates from
  poisoning in-memory born state. The reconciliation path is intentionally narrow: only typed create failures with
  `entity_already_exists` for the current packet's expected asset or track are retried.
- Go reviewer: Residual risk. This is not a durable checkpoint system. A restart can recover from already-born asset
  and track creates, but SemOps still lacks a replay cursor, graph read-back seed, or persisted birth cache.
- Technical writer: Pass. Docs must keep saying "typed OwnerToken minted by registry/bind" rather than teaching
  `<owner>#<incarnation>` as an adapter contract.

## Decision

Accept the slice as the current MAVLink graph-write baseline. Treat durable restart evidence as the next hardening
step, not as a blocker for the generated-frame SemStreams breaking-tag gate.
