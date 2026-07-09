# SAPIENT Parser Preflight Review

Date: 2026-06-20

## Decision

Accept `pkg/adapters/sapient` as a narrow parser-only preflight gate for SAPIENT planning. The package validates
representative BSI Flex 335 v2 JSON message shapes for:

- Registration.
- Status report.
- Detection report.
- Task acknowledgement.

This approves developer preflight evidence only. It does not approve binary protobuf decode, generated protobuf
bindings, graph projection, hosted SAPIENT service work, Dstl harness compliance, or product support claims.

## Objections Raised

- The official standard is protobuf. JSON fixtures from the Dstl harness are useful, but they are not binary
  `SapientMessage` payload coverage.
- The parser validates a representative subset, not the full BSI Flex 335 v2 schema. Alert, alert acknowledgement,
  task, registration acknowledgement, and error messages are still unimplemented.
- Hand-coded mandatory-field checks can drift from the official proto options. Generated bindings, descriptors, or a
  schema-derived validator should replace or backstop this before broad SAPIENT work.
- The current fixture constants are trimmed from public Dstl harness shapes. Full corpus vendoring still needs
  explicit attribution and redistribution review.
- SAPIENT tasking and acknowledgements are control-plane messages; parser success must not imply command authority or
  graph write ownership.

## Evidence Checked

- Official Dstl `sapient_message.proto`, `detection_report.proto`, `registration.proto`, `status_report.proto`,
  `task_ack.proto`, `location.proto`, `range_bearing.proto`, `associated_detection.proto`, and `associated_file.proto`
  field comments and mandatory options.
- Public Dstl BSI Flex 335 v2 Test Harness JSON shapes for valid detection, registration, status, and task-ack
  messages.
- `pkg/adapters/sapient/sapient.go` rejects malformed top-level envelope fields, missing oneof content, multiple
  content fields, invalid UUID/ULID identifiers, missing detection location, wrong registration `icdVersion`, missing
  status report ID, and missing task acknowledgement ID.
- `go test ./pkg/adapters/sapient` passes.

## Accepted Risks

- This package gives SemOps a fast Linux-friendly parser preflight while the official harness remains Windows-focused.
- The code intentionally leaves unsupported SAPIENT message types out rather than pretending to parse the full
  standard.
- Binary protobuf support remains a separate gate because this machine did not have `protoc` or `protoc-gen-go`.

## Follow-Up Tasks

- Add generated BSI Flex 335 v2 protobuf bindings or a reproducible descriptor toolchain.
- Add binary `SapientMessage` payload fixtures and prove JSON/binary agreement for the supported subset.
- Run or qualify the Dstl v2 Test Harness before any compliance language.
- Decide whether a portable Linux/CI preflight suite belongs in SemOps, SemStreams, or a separate ecosystem tool.
