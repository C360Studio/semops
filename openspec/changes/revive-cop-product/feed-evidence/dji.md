# DJI Feed Evidence

Status: critical HADR drone/vendor layer, not yet implemented.

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

- Parse a deterministic DJI-shaped telemetry fixture without graph writes.
- Preserve aircraft position, altitude, heading, gimbal/camera state, battery/source freshness, source identity, and
  media references.
- Publish bounded raw/vendor payload references rather than raw video bytes.
- Project DJI current aircraft/sensor state as `signal`, session/control state as `control`, annotations as
  `content`, and vendor replay/extraction records as `trace`.
- Review command authority, local override, credentials, and safety policy before any live driver or command path.

## Known Gaps

- No legal representative DJI telemetry/media fixture has been selected.
- No DJI SDK/cloud integration strategy has been chosen.
- No live DJI bridge, media relay, or command authority path exists.
- No DJI product support, compatibility, or certification claim is allowed yet.

## Source Links

- DJI Onboard SDK overview: <https://developer.dji.com/onboard-sdk/documentation/introduction/homepage.html>
