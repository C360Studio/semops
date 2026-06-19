# TAK/CoT Projection Adversarial Review

Date: 2026-06-19

## Decision

Accept `internal/projectors/cot` as the TAK/CoT projection-planner gate. Do not promote TAK/CoT to a structural stack
feed until graph writes, UI feed state, and stale-data downgrade behavior are proven against replayed CoT events.

## Evidence Checked

- `pkg/cop/contracts.go` now gives `semops.feed.tak` three grouped contracts: track `signal`, task/control `control`,
  and advisory/content `content`.
- `internal/projectors/cot/projector.go` emits SemStreams create/update mutation plans with typed owner tokens
  serialized only at graph request boundaries.
- TAK track projection births the source asset before emitting the strict `cop.track.source` edge.
- Marker events project to `task` entities instead of adding feed-specific marker types to the canonical model.
- GeoChat text projects to `advisory` entities so high-rate track state does not bury operator text.
- `go test ./pkg/cop ./internal/copownership ./internal/projectors/cot` passes.

## Objections

- This is not a TAK graph writer, structural adapter, Caddy/UI-visible feed, TAK Server, federation layer, auth layer,
  or full TAK interoperability claim.
- The projector owns one deterministic UID-to-entity mapping. Cross-source identity resolution between TAK-reported
  vehicles and MAVLink vehicles remains later fusion work.
- Stale timestamps influence only current projection status strings. Runtime downgrade, removal, or operator
  freshness display still needs a separate policy gate.
- Marker-to-task is a reasonable first mapping, but richer TAK tasking and mission-package behavior requires a
  dedicated operator-value and protocol review before expansion.

## Accepted Risks

- Keep TAK contracts grouped under `semops.feed.tak` because SemStreams registration already binds multiple contracts
  per owner and this avoids fake owners for one feed.
- Use source refs to replay-addressable raw CoT records instead of embedding native XML in graph entities.
- Keep alerts as unsupported no-ops for this slice; CAP and fusion alert semantics should not be guessed from the
  first CoT parser gate.

## Follow-Up Tasks

- Wire the CoT adapter to the projector and graph writer with the same born-state reconciliation discipline used by
  MAVLink.
- Add a skipped-by-default TAK live graph smoke before TAK enters the structural stack.
- Add COP API/UI fixture and graph-backed feed state for CoT tracks, tasks, and advisories.
- Add stale-data downgrade behavior and adversarial UX review before showing stale CoT as operator-current state.
