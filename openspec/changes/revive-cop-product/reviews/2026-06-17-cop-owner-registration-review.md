# COP Owner Registration Review

Date: 2026-06-17

Scope:

- `internal/copownership`
- `internal/smoke/mavlink/live_graph_test.go`
- SemStreams owner-token shape `<owner>#<incarnation>`

Evidence:

- Added `internal/copownership.RegisterFirstPhase`, which groups SemOps COP projection contracts by owner and calls
  SemStreams `projection.BindAndHeartbeat`.
- The binding result exposes the registry incarnation so projectors compose OwnerTokens as `<owner>#<incarnation>`.
- Reran the live MAVLink graph smoke against a clean temporary NATS/SemStreams stack at `nats://127.0.0.1:4222`.
- Result: pass. The smoke registered COP owners, asserted `OwnerOf(track, cop.track.position) == semops.feed.mavlink`,
  asserted the `semops.mavlink.track.v1`/`cop.track.source` foreign-edge claim, and wrote/read the MAVLink track.

## Feedback On `<owner>#<incarnation>`

- Keep it for now. It cleanly separates canonical owner identity from process lifetime and gives SemStreams a stale
  writer fence without forcing every producer to negotiate a session object.
- Do not expose the incarnation as static adapter configuration. SemOps should derive it from the same
  `ownership.Registry` instance that registered the owner claims, then pass it into projectors at composition time.
- Consider a small SemStreams helper type or function for token composition/parsing. The string format is currently
  easy to mistype, and downstream repos should not all re-learn delimiter and empty-token behavior.
- Consider whether append-evidence-only registrations should have a clearer first-class status. SemStreams warns that
  `semops.feed.cap` has no enforceable owning or foreign-edge claim; that is accurate, but product docs need to call
  out that this is governance evidence, not write-fence protection.
- SemStreams response relayed on 2026-06-17: accepted both asks. SemStreams plans to mint typed, opaque owner tokens
  only from the registry/bind path (`Registry.OwnerToken(owner)` and bind-result object) and to split
  append-evidence declarations from enforceable ownership/write-fence claims.

## Adversarial Findings

- Go reviewer: The smoke proves registry-derived tokens work for MAVLink's enforceable replace-owned claims.
- Architect: The token contract is acceptable, but only if the composition root owns registration and token suffix
  propagation. A YAML-provided suffix would create a fresh stale-token footgun.
- Go reviewer: The projector still carries an `OwnerTokenSuffix` field. It is acceptable as a low-level seam, but
  hosted runtime code must wrap it with the registration result.
- Technical writer: Counter evidence still needs before/after deltas. A clean SemStreams stack already exposes a
  `message_type="unknown"` indexing-profile default counter, so a naive "counter equals zero" assertion would be
  misleading.

## Decision

Accept `<owner>#<incarnation>` as an implementation detail for SemOps revival while SemStreams adds typed owner-token
minting. SemOps should migrate to the framework-owned token helper as soon as the breaking tag exposes it.
