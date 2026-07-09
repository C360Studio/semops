# DJI Synthetic Fixture Review

Date: 2026-06-22

Scope: first DJI-shaped telemetry/media-reference parser fixture.

## Decision

Accept a SemOps-owned synthetic DJI-shaped fixture as the first local evidence gate for DJI telemetry, media
references, and command-authority posture.

This is useful because DJI is a critical HADR drone/vendor layer and should appear in the product ladder early.
It is also dangerous because synthetic DJI-shaped data can be mistaken for DJI product compatibility. The fixture
must stay labeled as SemOps contract evidence only until legal representative captured samples or SDK/cloud API
integration tests exist.

## Adversarial Notes

### High: Synthetic DJI data can become accidental support theater

The parser proves SemOps can preserve the fields it currently cares about. It does not prove compatibility with DJI
Cloud API, Mobile SDK, Onboard SDK, Payload SDK, flight logs, subtitle tracks, media metadata, or live stream behavior.

Mitigation:

- Keep `fixtures/dji/telemetry-media.json` labeled as synthetic.
- Do not use this fixture in demo copy as DJI product support.
- Require a captured/legal representative fixture or SDK/cloud bridge before product compatibility claims.

### High: Command authority is data, not actuation

The fixture records command-authority posture, local override, and remote-command disabled state. It does not create a
command path and must not be treated as driver actuation evidence.

Mitigation:

- Keep command authority separate from telemetry parsing.
- Review credential handling, local override, operator confirmation, safety policy, and priority handling before any
  live command bridge.

### Medium: DJI media is not KLV by default

DJI video can arrive as live streams, recordings, subtitles, sidecars, vendor metadata, or SDK/cloud media references.
It is not automatically MISB ST 0601 or STANAG 4609 evidence.

Mitigation:

- Route DJI media as generic media references first.
- Send only actual KLV/MISB packets into the KLV worker.
- Keep SemOps responsible for DJI semantics and SemSource responsible only for generic storage/reference substrate.

## Verification

- `go test ./pkg/adapters/dji`
- `go test ./...`

## Follow-Ups

- Identify a legal representative captured DJI telemetry/media fixture.
- Add a replay store only after first captured or fixture-format contract stabilizes.
- Design SemStreams component ports and ownership only after ingress surface is chosen.
