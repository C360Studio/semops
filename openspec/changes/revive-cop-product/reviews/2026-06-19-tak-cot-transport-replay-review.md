# TAK/CoT Transport Replay Adversarial Review

Date: 2026-06-19

## Decision

Accept the SemOps-owned CoT UDP/TCP fixture replay harness as the TAK/CoT mock transport gate. Do not promote TAK/CoT
to a structural stack feed until projection, graph writes, UI state, and stale-data behavior are proven.

## Evidence Checked

- `pkg/adapters/cot/raw_lane.go` captures raw CoT XML with replay-addressable refs, source, UID, type, callsign, stale
  timestamp, record cap, and byte cap.
- `pkg/adapters/cot/replay.go` appends and loads JSON Lines raw CoT records and exposes deterministic seed events for
  ALPHA/BRAVO operators, North Gate marker, and GeoChat.
- `internal/adapters/cot/adapter.go` decodes raw XML, records health, captures malformed input before rejection, and
  optionally appends replay records without graph dependencies.
- `internal/adapters/cot/udp_listener.go`, `tcp_listener.go`, and `fixture_replay.go` prove loopback UDP datagrams and
  newline-delimited TCP fixture replay into the adapter.
- `go test ./pkg/adapters/cot ./internal/adapters/cot` passes.

## Objections

- This harness proves local UDP/TCP fixture replay only. It is not TAK Server behavior, federation, auth, user/team
  state, mission packages, or full TAK interoperability.
- TCP replay is newline-delimited for deterministic fixtures. Production TAK service work may need a different stream
  framing/session layer.
- Raw events are captured and replayable, but no projector maps operator dots, markers, or GeoChat into governed graph
  entities yet.
- Stale timestamps are preserved but no stale-data downgrade policy exists in runtime or UI.
- GeoChat text is decoded, but content/control split and operator-message entity shape remain projection-gate work.

## Accepted Risks

- Keep this as an in-process harness until the structural stack needs a containerized `semops-adapter-cot` or future
  SemOps TAK service boundary.
- Use SemOps-owned ALPHA/BRAVO seed fixtures now, with SemLink kept as prior art rather than a runtime dependency.
- Allow malformed CoT to enter the raw lane for audit/replay while rejecting it before projection.

## Follow-Up Tasks

- Add TAK projector tests for source assets, operator/track state, marker/control state, GeoChat/content state, and
  native source refs.
- Add born-first graph smoke before TAK enters the structural stack.
- Add COP API/UI feed state and stale-data downgrade behavior for replayed CoT events.
- Keep TAK Server-equivalent behavior on the future SemOps/SemStreams service roadmap, not inside this MVP adapter.
