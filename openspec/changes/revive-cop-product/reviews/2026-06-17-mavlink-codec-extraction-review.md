# MAVLink Codec Extraction Adversarial Review

Date: 2026-06-17
Scope: `COP-003` parser/generator extraction into `pkg/adapters/mavlink`

## Finding Summary

- Severity: Medium. The old battery parser/generator path was self-consistent but not strong interoperability
  evidence because it used the field declaration order rather than the MAVLink wire order. The active extraction
  corrects this and locks it with `TestGeneratedBatteryStatusUsesCanonicalWireOrder`.
- Severity: Medium. Command coverage is not preserved yet. The retained SITL reference has command helpers, but
  SemOps must not claim command/control coverage until COMMAND_LONG and COMMAND_ACK are active tests.
- Severity: Medium. The codec gate proves binary frame handling only. It does not yet prove raw-lane bounding,
  current-state projection, born-first source/track creation, or SemStreams indexing profile behavior.
- Severity: Low. The parser can register custom message specs, but the first product slice should keep dialect growth
  small until captured frames or simulator evidence prove the next messages.

## Role Review

Architect:

- Approves the active package boundary for parser/generator behavior.
- Requires a separate projector boundary before any graph writes so codec concerns do not own SemStreams mutation
  policy.

go-reviewer:

- Approves real-frame tests for heartbeat, global position, attitude, battery, split buffers, resync, checksum
  rejection, and sequence concurrency.
- Requires SITL command tests and explicit context-aware IO before command/control code enters the active build.

technical-writer:

- Approves deleting extracted ignored references and updating the quarantine doc.
- Requires feed evidence language to keep "codec extracted" separate from "MAVLink Phase 1 complete."

## Decision

Accept the codec extraction as the first `COP-003` implementation slice. Do not mark MAVLink feed entry complete until
SITL/PX4 evidence, projection tests, born-first graph writes, and indexing-profile behavior are proven.
