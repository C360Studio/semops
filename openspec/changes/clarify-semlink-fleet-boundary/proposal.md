# Clarify SemLink Fleet Boundary

## Why

SemLink should stay focused as the vehicle-local ArduPilot companion path. Recent demo planning made it tempting to
push multi-vehicle-per-companion and mesh/multiplex concerns down into SemLink, but the product boundary is cleaner if
SemLink remains one companion boundary per vehicle for MVP and SemOps owns the fleet-level COP.

## What Changes

- Clarify that SemLink MVP evidence represents one vehicle-local companion boundary per vehicle.
- Make SemOps the owner of fleet aggregation, mesh/multiplex handling, cross-companion deduplication, and COP target
  derivation.
- Preserve support for consuming SemLink simple-mesh evidence while treating `vehicle_count > 1` on a SemLink node as
  aggregator/stress evidence unless a later profile explicitly promotes it.
- Keep CS API/SemConnect at the SemOps interoperability edge, not in the SemLink hot path.

## Impact

SemLink can continue its focused one-companion-per-vehicle implementation. SemOps keeps responsibility for displaying
many vehicles, merging many companion sources, preserving provenance, and avoiding accidental claims that one SemLink
runtime must multiplex multiple ArduPilot vehicles.
