# Command Native Status Reconciliation Review

Scope: mapping native feed readback evidence into command-intent lifecycle status updates

## Decision

Accept a pure native-status reconciler for command intent. It maps protocol readback evidence, beginning with MAVLink
COMMAND_ACK result strings, into constrained command lifecycle status updates and projects only native ID, status,
description, and provenance fields on the command-intent entity. Desired state, authority, priority, and strict target
edges are not rewritten by native readback.

## Adversarial Findings

- Native readback must not own desired state. Feed evidence can say whether a platform accepted, rejected, started, or
  failed a command, but it should not overwrite the operator/CS API intent fields.
- MAVLink `accepted` is not mission success. It means the command was accepted by the native system; final task
  success requires later protocol- or behavior-specific evidence.
- `in_progress` should not bypass command authority. The reconciler rejects requested-to-executing transitions; the
  command must have passed admission/arbitration and become accepted first.
- Unknown native statuses are rejected instead of converted into invented lifecycle states.

## Acceptance

- MAVLink `accepted` maps to command-intent `accepted`.
- MAVLink `in_progress` maps to `executing` only from an accepted current command.
- MAVLink `denied`, `temporarily_rejected`, and `unsupported` map to `rejected`.
- MAVLink `failed` maps to `failed`; MAVLink `cancelled` maps to `cancelled`.
- Status projection does not emit target, desired-state, authority, or priority triples.

## Result

Accepted as a readback-to-intent reconciliation seam. Keep live transmit, final mission success, cancellation ACK
semantics, timeout timers, link-loss, partial execution, CS API status egress, and UI controls behind later reviews.
