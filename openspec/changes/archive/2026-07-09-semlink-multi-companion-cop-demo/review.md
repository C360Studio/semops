# Boundary Review

Date: 2026-07-09

## SemConnect / CS API Edge

The companion-fleet view model keeps SemLink runtime reports native in the SemLink-to-SemOps hot path. SemConnect and
CS API remain the interoperability/projection edge for SemOps-level consumers, not a required SemLink dependency.

Future projection fit:

- Companion nodes and vehicles can project as SemOps/SemConnect systems or deployments when SemConnect needs a common
  read API.
- MAVLink-native readback fields remain MAVLink vocabulary: `target_system_id`, `target_component_id`,
  `command_id=512`, `requested_message_id=148`, `COMMAND_ACK`, and `AUTOPILOT_VERSION`.
- SemOps readback adapter status is command/admission evidence. `COMMAND_ACK` status and decoded
  `AUTOPILOT_VERSION` payloads are separate observation/readback evidence.

## Standards And Governance Disposition

MAVLink owns command, message, target, ACK, and result vocabulary. SemLink mesh watermarks, bounded diffs, TTL merge
posture, demo assertion state, and raw-MAVLink exclusion are SemLink/C360 evidence fields, not CS API or MAVLink
domain facts.

C360 governance concepts remain labelled as local posture:

- no native transmit authority
- no companion hardware transmit authority
- deterministic demo evidence label
- raw MAVLink exclusion policy
- assertion state

## Adversarial Review

This slice proves SemOps can consume deterministic SemLink single-node and simple-mesh reports, normalize them into a
curated COP companion-fleet view model, and render/inspect at least three companion nodes in browser smoke coverage.

It does not prove live BlueOS, Navigator, ArduPilot hardware, RF/radio mesh reliability, peer management, mesh topology
control, companion command-rule execution, raw MAVLink forwarding, or CS API-first SemLink runtime compatibility. Peer
URLs from the SemLink report are parsed but are not exposed in the COP view model or browser surface.
