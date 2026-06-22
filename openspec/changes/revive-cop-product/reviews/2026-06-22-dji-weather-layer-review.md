# DJI And Weather Layer Review

Date: 2026-06-22

## Decision

Add DJI and weather as first-class feed/layer roadmap entries without changing the current KLV worker ownership.

Weather is split into visual map context, CAP/public alerts, and tactical weather telemetry. DJI is split into
telemetry/control state and media/video references.

## Objections Raised

- DJI video could tempt SemOps to treat every drone media path as KLV/STANAG. Reject that. DJI media may carry vendor
  metadata, subtitles, sidecars, or streams that are not MISB ST 0601 KLV.
- Weather can become enormous if SemOps ingests global models. Reject global model ingestion for the MVP. Query only
  local point/area/route/corridor slices when backend logic needs values.
- Visual weather tiles are operationally useful but do not automatically belong in graph state.
- DJI command/control introduces safety and authority concerns similar to MAVLink command work and must not be
  smuggled in through a telemetry adapter.

## Accepted Risks

- The demo roadmap grows, but the current Phase 1 stack remains MAVLink, TAK/CoT, CAP, and structural UI/API.
- Weather and DJI fixtures still need legal, representative samples.
- Media infrastructure may eventually need a shared SemSource/media sidecar or MediaMTX-style relay, but that is a
  service-promotion decision after fixtures prove the need.

## Follow-Up Tasks

- Select the first deterministic weather fixture and query shape.
- Select the first DJI telemetry/media fixture.
- Decide whether a weather tactical component starts from Open-Meteo-shaped JSON, OGC API EDR, or both.
- Decide whether DJI media needs a generic media-reference vocabulary before implementation.
