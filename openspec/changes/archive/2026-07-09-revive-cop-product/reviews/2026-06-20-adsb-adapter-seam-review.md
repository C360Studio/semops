# ADS-B Adapter Seam Review

Date: 2026-06-20
Scope: `COP-007` hosted ADS-B snapshot-ingest seam

## Decision

Accept the ADS-B adapter seam as hosted snapshot ingest for OpenSky-shaped JSON. This approves bounded raw capture,
JSONL replay append, projection writes, born-first reconciliation, stack composition with SemStreams request/reply,
and pollable health counters. It does not approve live OpenSky polling, receiver protocols, ASTERIX, airspace
filtering, or aircraft association.

## Objections Reviewed

- A hosted adapter name can sound like live service support. The adapter currently ingests supplied snapshots; it
  does not fetch from OpenSky or read a receiver.
- Raw snapshots could flood the graph. Raw JSON stays on the bounded raw lane and replay store; only current aircraft
  state projects to graph tracks.
- Malformed snapshots must remain auditable. The adapter captures and appends replay records before returning parse
  errors.
- ADR-055 conflicts must not regress to auto-vivify assumptions. `entity_already_exists` create failures mark the
  matching ADS-B track as born and reproject to update-only state.
- Association remains a fusion problem. The adapter still emits no `cop.track.source` edge and no cross-source
  aircraft association.
- Health must be operationally useful before Compose promotion. The seam reports received, captured, decoded, state,
  mutation, parse, replay, projection, and write counters plus last raw ref and ICAO24.

## Evidence

- `pkg/adapters/adsb.RawLane` captures bounded OpenSky-shaped snapshots with stable `adsb://raw/...` refs.
- `internal/adapters/adsb.Adapter` parses, captures, appends replay, projects, writes, reconciles born-first
  conflicts, and reports health.
- `internal/stack.NewADSBAdapter` wires the adapter to SemStreams NATS request/reply or injected writers.
- Targeted evidence command: `go test ./pkg/adapters/adsb ./internal/adapters/adsb ./internal/stack -count=1`.

## Follow-Ups

- Decide the first structural demo source: fixture replay only, optional live OpenSky, or local receiver files.
- Add Compose/runtime wiring only after the source choice is explicit.
- Keep live OpenSky auth/rate limits, readsb/dump1090, ASTERIX, and statistical association behind separate reviews.
