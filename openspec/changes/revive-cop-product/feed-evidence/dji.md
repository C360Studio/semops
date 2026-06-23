# DJI Feed Evidence

Status: critical HADR drone/vendor layer with first synthetic parser fixture evidence; not yet implemented as a
SemStreams component or live DJI bridge.

## Decision

DJI should be a first-class SemOps feed family, but it should not be folded into MAVLink or KLV.

Treat DJI as two related lanes:

1. Sensor/telemetry/control state: aircraft pose, gimbal/camera state, media/session status, battery, mission/session
   state, and command authority.
2. Media/video references: recorded media, live streams, keyframes, subtitles, vendor metadata, or later generic
   media-track extraction.

## Impact On Current KLV Work

DJI video reinforces the need for generic media references and optional shared media infrastructure, but it does not
change the KLV worker boundary.

- SemOps should not force DJI media or subtitles through the KLV/MISB decoder unless the source actually emits KLV.
- SemSource may help with generic storage, hashing, byte ranges, keyframes, and media references.
- A future media sidecar may expose generic track extraction for multiple products.
- SemOps still owns DJI semantics, command authority, support language, and any vendor-specific projection.
- SemOps still owns KLV/MISB/STANAG demux and decode claims for the KLV worker.

## First Acceptance Gates

- Parse a deterministic DJI-shaped telemetry fixture without graph writes. [done]
- Preserve aircraft position, altitude, heading, gimbal/camera state, battery/source freshness, source identity, and
  media references. [done]
- Preserve command-authority posture as data, with remote command execution disabled in the first fixture. [done]
- Publish bounded raw/vendor payload references rather than raw video bytes. [done]
- Project DJI current aircraft/sensor state as `signal`, session/control state as `control`, annotations as
  `content`, and vendor replay/extraction records as `trace`.
- Review command authority, local override, credentials, and safety policy before any live driver or command path.

## Local Evidence

- `fixtures/dji/telemetry-media.json` is a SemOps-owned synthetic DJI-shaped telemetry/media-reference fixture.
- The fixture is not captured DJI SDK, Cloud API, flight-log, subtitle, or media metadata evidence.
- `pkg/adapters/dji` parses aircraft state, battery, gimbal, camera, media references, source identity, and
  command-authority posture without graph writes.
- Media references are URI and metadata records only; this slice does not embed video bytes or decode media.
- Command authority is represented as posture data only. The fixture sets `remote_commands_enabled=false` and
  `local_override_required=true`.

## Known Gaps

- No legal representative DJI telemetry/media fixture has been selected.
- No DJI replay store exists beyond the committed synthetic JSON fixture.
- No DJI SDK/cloud integration strategy has been chosen.
- No live DJI bridge, media relay, or command authority path exists.
- No DJI product support, compatibility, or certification claim is allowed yet.

## Verification

- `go test ./pkg/adapters/dji`
- `go test ./...`

## Source Links

- DJI Onboard SDK overview: <https://developer.dji.com/onboard-sdk/documentation/introduction/homepage.html>
- DJI Mobile SDK introduction: <https://developer.dji.com/mobile-sdk/documentation/introduction/index.html>
- DJI Payload SDK introduction: <https://developer.dji.com/payload-sdk/documentation/introduction/index.html>
