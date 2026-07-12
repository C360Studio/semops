# Prove SemLink Mesh COP Demo

## Why

SemLink now owns a cleaner mesh proof: many vehicle-local companion nodes, with
one MAVLink vehicle per companion. SemOps has specs for artifact ingest and
fleet-boundary ownership, but it does not yet have a dedicated active spec for
the combined demo claim we want to show: SemOps acting as GCS/COP glass for a
SemLink one-companion-per-vehicle mesh artifact.

Without a named SemOps demo spec, the next implementation slice can drift
between boundary docs, generated artifact ingest, browser fixture tests, and
live/SITL fidelity claims.

## What Changes

- Add a SemOps mesh demo requirement that consumes a SemLink-generated
  `simple-mesh-companion-demo` artifact.
- Pin the default demo claim to at least three SemLink companion nodes with one
  vehicle each and `expected_summaries == node_count`.
- Require COP/GCS glass to expose node count, vehicle profile, per-node vehicle
  count, selected-summary catch-up, raw MAVLink exclusion, source fidelity, and
  no-transmit posture.
- Gate demo language so deterministic mesh evidence cannot be mistaken for live
  BlueOS, Navigator, radio mesh, hardware, SITL, or command authority evidence.

## Impact

This change gives SemOps the implementation contract for the next mesh demo
slice. SemLink stays focused on producing the artifact and companion evidence;
SemOps owns the operator-facing fleet view and claim discipline.
