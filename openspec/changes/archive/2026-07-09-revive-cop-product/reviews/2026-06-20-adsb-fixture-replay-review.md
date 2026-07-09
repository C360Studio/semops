# ADS-B Fixture Replay Review

Date: 2026-06-20
Scope: `COP-007` deterministic ADS-B replay through scenario-runner input

## Decision

Accept deterministic OpenSky-shaped ADS-B snapshot replay as an executable feed gate. This approves fixture records,
JSONL replay load/store, graph-plan writing, and optional scenario-runner replay through ADS-B projector/writer
interfaces. It does not approve hosted polling, live OpenSky use, local receiver protocols, ASTERIX, or aircraft
association.

## Objections Reviewed

- A replay fixture can accidentally become a product claim. The docs and OpenSpec now say this is deterministic
  snapshot replay only.
- Live OpenSky behavior depends on credentials, rate limits, and network availability. Replay remains the default CI
  path and live mode stays optional future work.
- Multiple snapshots must exercise birth/update discipline. The fixture replays the same ICAO24 in the second
  snapshot so the projector emits an update after the first birth is marked.
- Missing-position and MLAT evidence must stay honest. The fixture includes a missing-position aircraft and a
  non-ADS-B position source without fake coordinates or association edges.
- Scenario replay should not require Phase 1 to carry ADS-B. ADS-B snapshots are optional fixture family inputs; the
  default Phase 1 HADR fixture remains showable without ADS-B.

## Evidence

- `pkg/adapters/adsb.OpenSkyFixtureRecords` emits deterministic OpenSky-shaped snapshot records.
- `pkg/adapters/adsb.ReplayStore` and `LoadReplay` provide JSONL replay persistence for ADS-B raw snapshots.
- `internal/projectors/adsb.GraphWriter` sends create/update plans through SemStreams mutation subjects.
- `internal/scenario.Runner` can replay ADS-B snapshots through parse, projection, graph-plan writing, and
  born-state marking when supplied with ADS-B projector and writer dependencies.
- Targeted evidence command: `go test ./pkg/adapters/adsb ./internal/projectors/adsb ./internal/scenario -count=1`.

## Follow-Ups

- Add hosted ADS-B adapter health counters before wiring ADS-B into Compose.
- Decide whether the first hosted path uses fixture replay only, optional OpenSky live mode, or local receiver files.
- Keep live OpenSky, readsb/dump1090, ASTERIX, and statistical association behind separate reviews.
