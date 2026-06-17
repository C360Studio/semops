# MAVLink Command Extraction Adversarial Review

Date: 2026-06-17
Scope: `COP-003` active COMMAND_LONG/COMMAND_ACK extraction in `pkg/adapters/mavlink`

## Finding Summary

- Severity: Medium. COMMAND_LONG and COMMAND_ACK are now active codec coverage, but they do not prove live
  command/control against ArduPilot SITL, PX4 SITL, or MAVSDK.
- Severity: Medium. The ignored SITL controller was deliberately rejected. Any new live harness must use explicit
  readiness/state polling and context-aware I/O instead of restoring the old sleep-heavy controller.
- Severity: Medium. Command entities and command lifecycle graph projection are not implemented. Command codec
  coverage only proves native MAVLink command wire shape.
- Severity: Low. ArduCopter custom mode mapping is product-helpful, but PX4 mode semantics may differ and need their
  own smoke evidence before Phase 1 claims PX4 command behavior.

## Role Review

architect:

- Accepts extracting command payload and ACK behavior into the active adapter package.
- Rejects carrying the ignored SITL controller as reference code now that the command codec evidence exists.

go-reviewer:

- Accepts real-frame tests for canonical COMMAND_LONG payload order, COMMAND_ACK parsing, MAV_RESULT strings, and
  ArduCopter mode mapping.
- Requires future live harness tests to avoid arbitrary sleeps and to poll concrete drone state.

technical-writer:

- Requires docs to distinguish command codec coverage from live command/control coverage.
- Requires SITL/PX4 evidence to remain open until a modern harness is added and run.

## Decision

Accept command codec extraction and deletion of the ignored SITL tree. Do not claim live command/control until a
modern simulator harness proves connection, state reads, safe commands, ACK handling, and post-command state polling.
