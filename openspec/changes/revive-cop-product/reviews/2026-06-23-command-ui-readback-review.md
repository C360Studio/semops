# Command UI Readback Review

Date: 2026-06-23

Decision: accept command-intent task discovery and read-only UI inspection as status-visibility evidence.

This slice adds command task prefix discovery to the COP API, maps latest command lifecycle fields into the curated
task view model, separates command feed health from TAK/CoT task health, and renders command tasks as selectable rows
in the Svelte COP. The UI can inspect target, authority, priority, expiry, requested-by, correlation, desired state,
status, owner, and source reference.

Adversarial findings:

- The command task source family is `command` even when the latest provenance source is `cs-api`, `local-ui`, or
  native ACK/readback evidence; otherwise command lifecycle state would be misattributed to TAK, MAVLink, or another
  feed in source health.
- Task readback must use latest lifecycle triples for status, description, source reference, and observed time because
  command entities accumulate requested, cancellation, native readback, and deadline updates over time.
- Command rows are read-only. This is not a command shell, CS API ingress handler, operator execute/cancel workflow,
  hosted scheduler, native command transmitter, or safety-interlock review.
- Command tasks without geometry remain inspectable through the task rail and map selector, but they must not be
  treated as spatial task points until the command has a reviewed geometry or target-state representation.
- The Playwright smoke proves visibility and selection for the mocked API snapshot only. Live command/control remains
  blocked until CS API request handling, stale-command rejection, local override, safety interlocks, native
  acknowledgement, and asynchronous feedback are reviewed.
