# SemLink Companion Readback Contract v0

Status: draft

Owner: SemOps

Reviewers: SemLink, SemConnect

## Purpose

This contract defines the MVP SemLink-to-SemOps companion path for one safe ArduPilot readback request:
`MAV_CMD_REQUEST_MESSAGE` for `AUTOPILOT_VERSION`.

The hot path is native and low friction. SemLink does not need to expose or consume CS API to submit this MVP request.
SemOps still proves standards interop at the SemConnect/CS API edge.

## Directionality

The MVP has two separate paths:

- Intent admission: a SemLink companion-side adapter posts readback intent to SemOps.
- Evidence readback: SemOps, semstreams-ui, or operator tooling may pull SemLink `/api/evidence` for local companion
  readiness and posture.

Those paths have different trust models. This contract covers intent admission only. SemLink `/api/evidence` remains
read-only companion evidence and is not the command/request API.

## Endpoint

```http
POST /api/cop/semlink/ardupilot/readback
Content-Type: application/json
```

Current implementation note: the live handler still uses the earlier `mesh_node_id` and `vehicle_system_id` shape.
The contract below is the v0 target shape for the next handler migration.

## Authentication Boundary

SemLink identifies itself as a companion node. SemLink does not mint trusted SemOps operator headers.

A SemOps deployment gateway, sidecar, or test harness owns trusted-header translation before the request reaches the
SemOps API. Direct SemLink-to-SemOps local development may use test headers, but that is not a production trust model.

The trusted boundary must provide:

| Header | Meaning | Owner |
| --- | --- | --- |
| `X-SemOps-Operator-Authenticated` | Trusted marker that the upstream boundary authenticated the caller | SemOps deployment boundary |
| `X-SemOps-Operator-ID` | Authenticated caller or service principal | SemOps deployment boundary |
| `X-SemOps-Operator-Role` | SemOps role allowed for readback admission | SemOps deployment boundary |
| `X-SemOps-Authority-Scope` | Must be `semlink.readback.intent` | SemOps deployment boundary |
| `X-SemOps-Authority-Domain` | Authority domain used for audit | SemOps deployment boundary |
| `X-SemOps-SemLink-Companion-Node-ID` | Authenticated companion node ID | SemOps deployment boundary |

`X-SemOps-SemLink-Mesh-Node-ID` is a compatibility alias while the current route is migrated. The body field
`companion_node_id` is canonical in v0.

## Request

```json
{
  "contract": "c360.semops.semlink.ardupilot.readback.v0",
  "companion_node_id": "blue-boat-01",
  "target_system_id": 42,
  "target_component_id": 1,
  "command_id": 512,
  "requested_message_id": 148,
  "correlation_id": "corr-blue-boat-01-autopilot-version-001",
  "idempotency_key": "idem-blue-boat-01-autopilot-version-001",
  "requested_at": "2026-07-07T18:30:00Z",
  "ttl_seconds": 30,
  "source_ref": "semlink://blue-boat-01/ardupilot/system-42/request-autopilot-version"
}
```

### Request Rules

- `companion_node_id` is required and must match the trusted companion-node header after gateway translation.
- `target_system_id` is required and must be in the MAVLink system ID range `1..255`.
- `target_component_id` is required for v0. Use `1` for the autopilot component when no narrower component is known.
- `command_id` is required and must be `512` (`MAV_CMD_REQUEST_MESSAGE`).
- `requested_message_id` is required and must be `148` (`AUTOPILOT_VERSION`).
- `correlation_id` is required for tracing.
- `idempotency_key` is required. Duplicate keys collapse before a second graph write.
- `requested_at` is required. SemOps derives `expires_at = requested_at + ttl_seconds`.
- `ttl_seconds` is required and must be positive. Expired requests are rejected after reconnect or replay unless a
  later policy explicitly renews them.
- `source_ref` is optional. If omitted, SemOps derives a `semlink://` source reference from the companion node and
  target system.

SemLink does not send `target_asset_id`. SemOps derives the canonical COP target asset from the MAVLink target system
and configured MAVLink org/platform, then proves the target is born before command-intent graph writes.

SemLink does not send `action`. For v0, the action is fully determined by `command_id=512` and
`requested_message_id=148`.

SemLink does not need to send a request `id`. SemOps can derive a native command-intent ID from the companion node,
correlation ID, and requested message.

## Accepted Response

HTTP status: `202 Accepted`

```json
{
  "contract": "c360.semops.semlink.ardupilot.readback.v0",
  "accepted": true,
  "status": "accepted",
  "duplicate": false,
  "correlation_id": "corr-blue-boat-01-autopilot-version-001",
  "idempotency_key": "idem-blue-boat-01-autopilot-version-001",
  "companion_node_id": "blue-boat-01",
  "authorized_companion_node_id": "blue-boat-01",
  "authority_scope": "semlink.readback.intent",
  "target_asset_id": "c360.edge.cop.mavlink.asset.system-42",
  "entity_id": "c360.edge.cop.command.task.semlink-blue-boat-01-autopilot-version",
  "native_id": "semlink-blue-boat-01-autopilot-version",
  "claim_scope": "semlink-companion-command-intent-only",
  "source_ref": "semlink://blue-boat-01/ardupilot/system-42/request-autopilot-version",
  "requested_at": "2026-07-07T18:30:00Z",
  "expires_at": "2026-07-07T18:30:30Z",
  "native_execution_allowed": false,
  "companion_transmit_allowed": false,
  "mutations": 1
}
```

## Duplicate Response

HTTP status: `202 Accepted`

```json
{
  "contract": "c360.semops.semlink.ardupilot.readback.v0",
  "accepted": false,
  "status": "duplicate",
  "duplicate": true,
  "correlation_id": "corr-blue-boat-01-autopilot-version-001",
  "idempotency_key": "idem-blue-boat-01-autopilot-version-001",
  "companion_node_id": "blue-boat-01",
  "existing_native_id": "semlink-blue-boat-01-autopilot-version",
  "native_execution_allowed": false,
  "companion_transmit_allowed": false,
  "mutations": 0
}
```

## Rejected Response

HTTP status depends on the rejection boundary:

- `400` for malformed JSON, unsupported MAVLink command/message, invalid target system/component, or invalid TTL.
- `401` or `403` for missing, mismatched, or insufficient trusted gateway headers.
- `202` with `accepted=false` for semantic admission rejections such as duplicate, stale, superseded, or unresolved
  target after a syntactically valid request.

```json
{
  "contract": "c360.semops.semlink.ardupilot.readback.v0",
  "accepted": false,
  "status": "rejected",
  "duplicate": false,
  "correlation_id": "corr-blue-boat-01-autopilot-version-001",
  "idempotency_key": "idem-blue-boat-01-autopilot-version-001",
  "companion_node_id": "blue-boat-01",
  "rejected_reason": "expired request",
  "native_execution_allowed": false,
  "companion_transmit_allowed": false,
  "mutations": 0
}
```

## Status And Result Evidence

The admission response records command intent only. It does not prove that SemLink transmitted a MAVLink command or
that ArduPilot returned `AUTOPILOT_VERSION`.

For v0, SemOps can expose ACK/status readback through its command-task read model when native status evidence exists.
Structured `AUTOPILOT_VERSION` payload decoding is deferred to a follow-up SemLink/SemOps slice. When added, result
evidence should use separate timestamps:

- `ack_observed_at` for `COMMAND_ACK`.
- `result_observed_at` for decoded `AUTOPILOT_VERSION` evidence.

`AUTOPILOT_VERSION` result payloads map to observation/readback evidence, not to command acceptance itself.

## Standards Mapping

| Field | Contract role | Standards mapping | Disposition |
| --- | --- | --- | --- |
| `contract` | Contract version discriminator | None | C360 governance: versioned compatibility guard |
| `companion_node_id` | SemLink companion identity | CS API `System` identifier at SemOps edge | Domain field with CS API projection |
| `target_system_id` | MAVLink target system | MAVLink system ID | Standards-owned |
| `target_component_id` | MAVLink target component | MAVLink component ID | Standards-owned |
| `command_id` | MAVLink command | MAVLink `MAV_CMD_REQUEST_MESSAGE` (`512`) | Standards-owned |
| `requested_message_id` | Requested MAVLink message | MAVLink `AUTOPILOT_VERSION` (`148`) | Standards-owned |
| `correlation_id` | End-to-end trace key | CS API/SensorThings extension or command metadata | Governance/audit field |
| `idempotency_key` | Duplicate-collapse key | No direct MAVLink/CS API equivalent | C360 governance: replay and duplicate safety |
| `requested_at` | Request issuance time | CS API Command `issueTime`; SensorThings task/request time | Standards-aligned |
| `ttl_seconds` | Admission lifetime | Derives command/task `expires_at`; no MAVLink equivalent | C360 governance: replay safety |
| `source_ref` | Native provenance URI | SemOps provenance; CS API link or extension at edge | Governance/audit field |
| `target_asset_id` | SemOps COP target | CS API `System` at SemConnect edge | SemOps-derived; not sent by SemLink |
| `accepted` | SemOps admission decision | CS API Command status projection | Governance result |
| `status` | Admission/readback status | CS API Command status projection | Standards-aligned at edge |
| `duplicate` | Duplicate admission flag | No direct MAVLink/CS API equivalent | C360 governance: duplicate safety |
| `existing_native_id` | Existing command-intent native ID | SemOps command intent identity | C360 governance/readback |
| `claim_scope` | Authority claim shape | No direct MAVLink/CS API equivalent | C360 governance: no-transmit posture |
| `native_execution_allowed` | Native transmit posture | No direct MAVLink/CS API equivalent | C360 governance: safety |
| `companion_transmit_allowed` | Companion transmit posture | No direct MAVLink/CS API equivalent | C360 governance: safety |
| `ack_observed_at` | ACK observation time | MAVLink `COMMAND_ACK`; CS API SystemEvent/Observation time | Deferred v0.1 |
| `result_observed_at` | Result observation time | MAVLink `AUTOPILOT_VERSION`; CS API Observation time | Deferred v0.1 |

## CS API / SemConnect Edge

SemOps must prove that accepted readback intent and status can project through SemConnect without semantic loss. The
expected CS API shape is:

- SemLink companion and vehicle target as CS API `System` resources.
- Readback channel as a CS API `ControlStream`.
- Admission record as a CS API `Command`.
- ACK/status as CS API command status or `SystemEvent`.
- Decoded `AUTOPILOT_VERSION` as observation/readback evidence when that payload exists.

CS API score for MVP is Amber: it is the right interop and projection edge, but the native companion contract remains
the hot path.

## Implementation Follow-Ups

- Migrate the SemOps handler request body from `mesh_node_id` / `vehicle_system_id` / `action` to the v0 contract
  shape, preserving compatibility aliases where needed.
- Move trusted-header companion node binding from mesh wording to companion-node wording.
- Keep `target_asset_id` as a SemOps response/readback field, not a SemLink request requirement.
- Add structured `AUTOPILOT_VERSION` payload decoding and observation evidence in a later slice.
