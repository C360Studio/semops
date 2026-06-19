# CAP Lifecycle Fixture Replay Review

Date: 2026-06-19

## Decision

Accept CAP raw XML lifecycle fixture replay as the feed-local replay artifact for the future scenario runner.

Do not claim hosted CAP polling, NWS/IPAWS integration, captured NWS samples, CAP consumer conformance, or
authoritative hazard lifecycle ownership from this gate.

## Objections Raised

- Synthetic lifecycle fixtures could be mistaken for real NWS sample evidence.
- Replay records could drift into a SemOps-only JSON event format and stop exercising the native CAP XML parser.
- A batch lifecycle projection could accidentally reintroduce auto-vivify behavior if update/cancel records are not
  ordered after an explicit hazard birth.
- CAP cancel/expire fixture evidence should not imply SemOps owns authoritative `cop.hazard.status`.

## Evidence Checked

- `pkg/adapters/cap` now stores replayable raw XML CAP records as JSON Lines.
- `LifecycleFixtureRecords` emits alert, update, cancel, and expired-alert records that parse through the CAP codec.
- `LoadReplay` and `ReplayStore.Append` preserve raw XML bytes and reject incomplete records.
- `internal/projectors/cap` projects the lifecycle fixture as create/update/update/create for the two hazard IDs.
- The CAP projector still writes only append-evidence predicates and does not claim authoritative hazard geometry,
  severity, or status.

## Accepted Risks

- The fixture is synthetic and deterministic, not captured from NWS.
- The fixture pack is feed-local; the full scenario runner still needs a demo clock and multi-feed choreography.
- XML schema and CAP consumer-rule validation remain open.

## Follow-Up Tasks

- Capture real NWS alert/update/cancel examples as deterministic fixtures.
- Add scenario-runner playback that emits the CAP lifecycle fixture on a controllable demo clock.
- Add schema/consumer-rule validation before any CAP conformance claim.
- Decide later whether authoritative hazard lifecycle state belongs to CAP control projection or fusion-derived state.
