# CAP Lifecycle Readback Review

Date: 2026-06-19

## Decision

Accept derived CAP lifecycle status in the COP hazard view model.

Do not treat this as authoritative hazard-status ownership, hosted CAP service behavior, NWS/IPAWS integration, CAP
consumer conformance, or replayed lifecycle sequence evidence.

## Objections Raised

- Adding `status` to the hazard view model could be mistaken for CAP owning `cop.hazard.status` in SemStreams.
- CAP update evidence can append multiple `cop.hazard.evidence` triples to one entity, so readback must prefer the
  latest evidence instead of relying on unspecified property selection.
- Expired, cancelled, stale, and non-operational CAP states should be visible to operators without overwriting
  stricter tactical or fusion-owned hazard state.
- The first status mapping should not become a substitute for captured NWS update/cancel/expire fixtures.

## Evidence Checked

- `internal/api/cop` now derives hazard status from CAP evidence `msgType`, CAP `status`, `expires`, and read-time
  freshness.
- CAP hazard readback uses the newest `cop.hazard.evidence`, newest observed time, and newest source reference when
  multiple evidence triples exist.
- The graph projector still does not write `cop.hazard.status`, `cop.hazard.geometry`, or `cop.hazard.severity`.
- The UI type and fixture include hazard status so the existing inspector can display it without new controls.

## Accepted Risks

- The mapping is intentionally small: `active`, `active.update`, `cancelled`, `expired`, `stale`,
  `nonoperational.<status>`, `acknowledged`, and `error`.
- The map symbology does not yet distinguish cancelled or expired hazards; that needs a UX pass.
- CAP status comes from deterministic fixture-shaped evidence, not captured NWS lifecycle sequences.

## Follow-Up Tasks

- Capture NWS update/cancel/expire samples as deterministic fixtures.
- Add scenario-runner playback that shows CAP lifecycle transitions over demo time.
- Add a UX review before styling expired/cancelled/stale hazard polygons differently.
- Decide whether authoritative hazard lifecycle belongs to a CAP-owned control projection or a fusion-owned derived
  projection after fixture evidence exists.
