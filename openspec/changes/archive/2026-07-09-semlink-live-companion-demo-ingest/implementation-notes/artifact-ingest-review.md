# Artifact Ingest Review

## Implemented Handoff

SemOps now accepts a SemLink companion demo artifact through a local file handoff:

- `SEMOPS_COP_SEMLINK_ARTIFACT_PATH` points the COP snapshot provider at the generated artifact.
- `SEMOPS_COP_SEMLINK_ARTIFACT_MAX_AGE` optionally rejects stale generated artifacts.
- `semlink-companion-demo-artifact-v0` wraps the existing SemLink demo report shape with source-fidelity metadata.
- Raw SemLink report JSON can still be parsed as committed fixture evidence for compatibility tests.

The artifact envelope records SemLink version or commit, generator command or profile, simulator family,
no-transmit posture, and per-node source posture. The embedded report remains SemLink's
`single-node-companion-demo` or `simple-mesh-companion-demo` payload.

For shared demos and release evidence, SemLink should populate `semlink_commit` or `semlink_version` with a real
source tag or commit instead of a local/dev placeholder. SemOps accepts either field, but generated artifacts must keep
the envelope `generated_at` equal to the embedded report `generated_at` so freshness checks and provenance refer to one
coherent evidence run.

## Source Fidelity

The COP view model now separates:

- `fixture`: committed SemOps fixture evidence.
- `generated`: SemLink-generated artifact evidence.
- `deterministic`: generated or fixture summary evidence without live vehicle source fidelity.
- `sitl-backed`: node-level ArduPilot SITL source evidence with MAVLink system ID and route metadata.
- `hardware-adjacent`: reserved for later near-hardware evidence that still needs separate acceptance.

The mixed demo fixture at
`testdata/contracts/semlink-companion-demo-v0/generated-mixed-sitl.artifact.json` shows three SemLink nodes with one
ArduPilot SITL-backed node and two deterministic summary nodes.

## Boundary Review

SemLink still owns companion runtime, mesh peer behavior, MAVLink/SITL ingestion, and the producer lane that emits
artifact files. SemOps owns artifact admission, COP/GCS glass, source-fidelity labeling, and operator-facing
no-transmit posture.

This slice does not put CS API or SemConnect into the SemLink-to-SemOps hot path. CS API/SemConnect remains the
interoperability and projection edge at SemOps/SemConnect level.

## N-SITL Rule

One SITL-backed SemLink node is sufficient for the first MAVLink/ArduPilot compatibility proof. A demo only requires
`n` SITL instances when it claims `n` live ArduPilot vehicles, `n` live ArduPilot SITL instances, or per-node live
vehicle fidelity. In that case each claimed live node needs distinct MAVLink identity and route evidence.

## Adversarial Review

This implementation proves SemOps can ingest and display a SemLink-shaped generated artifact. It does not prove:

- BlueOS or Navigator deployment readiness.
- Hardware or radio behavior.
- Mesh reliability under real network loss.
- SemOps control of SemLink/SITL process lifecycle.
- Raw MAVLink forwarding through SemOps.
- Mission upload, mode change, arm/disarm, or offboard control authority.
- `n` live ArduPilot vehicles from an `n`-node mixed fleet unless every node has live vehicle source metadata.

SemLink should review the envelope fields before its producer lane emits this artifact kind as final acceptance.

## Verification

- `go test ./internal/adapters/semlinkdemo ./internal/api/cop ./internal/app ./cmd/semops`
- `go test ./...`
- `go build ./...`
- `npm run check`
- `npm run test`
- `npm run test:e2e`
- `openspec validate semlink-live-companion-demo-ingest --strict`
- `openspec validate --all --strict`
