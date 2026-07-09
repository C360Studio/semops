# SemStreams Raw-Lane Current-State Ask Review

Date: 2026-06-23
Scope: upstream SemStreams ask for raw-lane plus current-state projection guidance

## Decision

File a non-blocking SemStreams issue for a documented raw-lane plus current-state projection pattern. The ask should
be guidance and optional helper shape first, not a new mandatory framework layer.

SemOps now has enough repeated feed pressure to justify the ask: MAVLink, TAK/CoT, ADS-B, SAPIENT, and KLV all carry
bounded raw or storage-reference evidence while projecting only governed current state, footprints, detections, or
task readback into the graph.

## Workflow And Evidence

- MAVLink keeps frame bytes on a bounded raw lane and replay store, then writes source asset, track current state,
  and command ACK readback with source references.
- TAK/CoT keeps XML payloads on bounded raw/replay paths while projecting operator tracks, tasks, advisories, and
  marker control state.
- ADS-B keeps OpenSky-shaped snapshots on raw/replay paths while projecting current aircraft tracks.
- SAPIENT keeps JSON/protobuf payloads on bounded raw/replay paths while projecting only reviewed absolute-location
  detections.
- KLV keeps media and packet bytes by reference while projecting bounded sensor/frame-center evidence.

The concrete fixture pressure is not a single failing SemStreams test. It is repeated downstream fixture and component
code that has to answer the same question: which bytes stay off graph, which source reference is graph-visible, which
entity carries current state, which `indexing_profile` applies, and where replay/debug evidence lives.

## Rejected Product-Local Option

SemOps can keep local raw lanes per adapter, but that leaves sibling products to reinvent the same separation between
raw payload retention, replay references, component flow telemetry, governed projection, and index profile posture.
That repetition already shows up across C360 feed products and will get sharper as SemTeams/SemConnect consume the
same SemStreams lifecycle and governance patterns.

## Why This Belongs In SemStreams

SemStreams already owns the surrounding framework vocabulary: component lifecycle, payload registry, ports, graph
ownership, indexing profiles, utilities such as buffer/cache/natsclient, and Prometheus-friendly component health and
flow telemetry. A reusable recipe can tie those pieces together without forcing SemOps product vocabulary upstream.

## Accepted Risks

- This is not a blocker for SemOps; local code can continue.
- The issue should avoid inventing a raw-storage service or mandatory object-store API.
- The pattern should leave product-specific source references, retention policy, and replay fixture tiers to product
  code unless multiple products prove a sharper reusable contract.

## Follow-Up

After filing, mark the SemOps raw-lane/current-state upstream task closed with the SemStreams issue reference and keep
broader indexing/cardinality asks gated until mixed-feed query pressure proves a specific gap.

SemStreams accepted the ask in issue #340 as a docs/composition guide. The upstream response keeps ObjectStore optional,
promotes replay attribution to a first-class rule, preserves per-source current-state plus separate fusion ownership,
and defers source-reference vocabulary/helper contracts until more feeds prove the reusable shape. Draft guide PR #344
is the upstream artifact SemOps should track.
