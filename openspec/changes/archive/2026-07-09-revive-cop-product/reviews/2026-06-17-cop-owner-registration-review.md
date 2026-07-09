# COP Owner Registration Review

Date: 2026-06-17

Scope:

- `internal/copownership`
- `internal/smoke/mavlink/live_graph_test.go`
- SemStreams typed `ownership.OwnerToken` bind-result path

Evidence:

- Added `internal/copownership.RegisterFirstPhase`, which groups SemOps COP projection contracts by owner and calls
  SemStreams `projection.BindAndHeartbeat`.
- The binding result now exposes typed `ownership.OwnerToken` values minted by SemStreams registry/bind results.
- Reran the live MAVLink graph smoke against a clean temporary NATS/SemStreams stack at `nats://127.0.0.1:4222`.
- Result: pass. The smoke registered COP owners, asserted `OwnerOf(track, cop.track.position) == semops.feed.mavlink`,
  asserted the `semops.mavlink.track.v1`/`cop.track.source` foreign-edge claim, and wrote/read the MAVLink track.

## Feedback On OwnerToken Ergonomics

- The original wire shape cleanly separated canonical owner identity from process lifetime, but downstream adapters
  should not assemble it themselves.
- Do not expose incarnation or token pieces as static adapter configuration. SemOps should derive owner tokens from the
  same `ownership.Registry` instance that registered the owner claims, then pass typed tokens into projectors at
  composition time.
- A small SemStreams helper type is the right shape. The token format is easy to mistype, and downstream repos should
  not all re-learn delimiter and empty-token behavior.
- Consider whether append-evidence-only registrations should have a clearer first-class status. SemStreams warns that
  `semops.feed.cap` has no enforceable owning or foreign-edge claim; that is accurate, but product docs need to call
  out that this is governance evidence, not write-fence protection.
- SemStreams response relayed on 2026-06-17: accepted both asks. SemStreams plans to mint typed, opaque owner tokens
  only from the registry/bind path (`Registry.OwnerToken(owner)` and bind-result object) and to split
  append-evidence declarations from enforceable ownership/write-fence claims.
- SemOps migration completed on 2026-06-17: `internal/copownership.BindingResult` carries typed tokens, runtime passes
  the token map through `internal/stack`, and MAVLink projectors call `OwnerToken.Wire()` only at graph mutation
  request construction.

## Adversarial Findings

- Go reviewer: The smoke proves registry-derived tokens work for MAVLink's enforceable replace-owned claims.
- Architect: The token contract is acceptable now that the composition root owns registration and passes typed tokens,
  not a YAML or adapter-provided suffix.
- Go reviewer: The previous `OwnerTokenSuffix` low-level seam has been removed. Remaining tests that mention
  `<owner>#<incarnation>` do so only while asserting serialized graph request payloads.
- Technical writer: Counter evidence now uses before/after deltas for SemOps message types, avoiding naive
  zero-total assertions against unrelated baseline metrics.

## Decision

Accept typed `ownership.OwnerToken` as the SemOps runtime contract. The wire string remains a SemStreams implementation
detail visible only in graph mutation request payloads.
