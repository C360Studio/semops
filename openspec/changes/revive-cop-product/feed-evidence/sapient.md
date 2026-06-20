# SAPIENT Feed Evidence

Status: JSON preflight parser exists; binary protobuf and harness qualification still required before implementation.

## Decision

SAPIENT has moved from artifact discovery to a parser/harness planning lane. SemOps now has a dependency-light JSON
preflight parser for representative BSI Flex 335 v2 message shapes, but it must not claim SAPIENT product support,
binary protobuf coverage, or compliance until generated bindings and a documented harness run prove the boundary.

## Local Evidence

- No SemOps SAPIENT adapter exists in the current checkout.
- `pkg/adapters/sapient` parses Dstl-harness-shaped JSON preflight fixtures for registration, status report,
  detection report, and task acknowledgement messages.
- `go test ./pkg/adapters/sapient` validates required top-level envelope fields, UUID/ULID identity fields,
  mandatory content fields, location/range-bearing oneof behavior, and malformed fixture rejection before projection.
- No SemOps SAPIENT generated protobuf bindings, binary decoder, hosted adapter, or graph projector exists yet.
- No local SAPIENT test harness run has been performed.
- The feed ladder assigns detections/tracks to `signal`, tasking/collection state to `control`, and native decode
  traces to `trace`.

## External Evidence

- GOV.UK documents SAPIENT as a Dstl/MOD-owned architecture and names BSI Flex 335 as the freely available ICD:
  <https://www.gov.uk/guidance/sapient-autonomous-sensor-system>.
- GOV.UK also links official GitHub assets for protobuf files, a SAPIENT Test Harness, and SAPIENT Middleware.
- BSI identifies `SAPIENT Network of Autonomous Sensors and Effectors - BSI Flex 335 V2:2024` as the current
  structure, format, and content reference for SAPIENT messages:
  <https://www.bsigroup.com/en-US/insights-and-media/insights/brochures/bsi-flex-335-interface-of-the-sapient-sensor-management-specification/>.
- Dstl publishes official protobuf definitions in `dstl/SAPIENT-Proto-Files`, including `bsi_flex_335_v2_0`:
  <https://github.com/dstl/SAPIENT-Proto-Files>.
- Dstl publishes `dstl/BSI-Flex-335-v2-Test-Harness` for BSI Flex 335 v2 component-message compliance testing:
  <https://github.com/dstl/BSI-Flex-335-v2-Test-Harness>.
- Dstl publishes Apex SAPIENT Middleware for routing, optional protobuf validation, archiving, replay, and REST API
  access: <https://github.com/dstl/Apex-SAPIENT-Middleware>.
- The older `dstl/SAPIENT-Middleware-and-Test-Harness` repository now points to the v2 test harness and Apex
  middleware as the current split.

## Gates

### Artifact Discovery Gate

Target outcome:

- Keep the authoritative artifact set explicit before code starts.

Acceptance:

- ICD or schema source is identified. [done]
- Message samples or fixtures are available. [done: Dstl test harness true/false JSON corpora]
- License and redistribution constraints are understood. [done: Apache-2.0 except where noted for Dstl assets]
- If a compliance suite exists, its install/run path is documented. [open: Windows/.NET/PostgreSQL 12 harness path]

### Harness Qualification Gate

Target environment after implementation planning:

- Windows 10/11 VM or workstation with .NET 6 SDK, PostgreSQL 12, and the Dstl BSI Flex 335 v2 Test Harness.

Acceptance:

- Harness version, commit, and configuration are recorded.
- SemOps-generated registration, status, detection, alert, and task-ack messages are exercised where applicable.
- Failures are captured as either SemOps bugs, unsupported SAPIENT subset, or upstream tooling issues.
- No compliance wording appears in demo materials without the harness result and scope.

### Portable Preflight Suite Gate

Future effort:

- Create or contribute a cross-platform SAPIENT preflight suite that can run in Linux CI from official BSI Flex 335 v2
  protobufs and fixture corpora.

Acceptance:

- The suite runs without Windows-only assumptions, PostgreSQL 12 pinning, or manual GUI setup.
- It covers parser validation, mandatory-field failures, representative registration/status/detection/task/alert
  messages, and malformed binary payloads.
- It is described as developer preflight or interoperability evidence until Dstl, BSI, or another accepted authority
  treats it as a compliance substitute.

### Parser Gate

Target command:

```bash
go test ./pkg/adapters/sapient
```

Acceptance:

- Valid Dstl or BSI Flex 335 v2-aligned JSON fixtures parse. [done: representative preflight subset]
- Malformed messages fail before graph writes. [done: representative preflight subset]
- Unknown or future fields are handled according to the authoritative compatibility rules. [open]
- Generated bindings are versioned so BSI Flex 335 v1/v2 drift is visible. [open]

### Binary Protobuf Gate

Target command after generated bindings exist:

```bash
go test ./pkg/adapters/sapient -run Protobuf
```

Acceptance:

- Official BSI Flex 335 v2 `.proto` files are generated or compiled through a reproducible toolchain.
- Binary `SapientMessage` payload fixtures decode into typed SemOps preflight messages.
- JSON preflight and binary decode agree on envelope, content kind, identity fields, and required-field failures.
- BSI Flex 335 v1/v2 drift is visible in package paths, generated-code provenance, or fixture metadata.

### Projection Gate

Target command after SemOps graph contracts exist:

```bash
go test ./internal/projectors/sapient
```

Acceptance:

- Detections and tracks use `indexing_profile=signal`.
- Sensor tasking, collection plans, and alert state use `indexing_profile=control`.
- Native decode/replay records use `indexing_profile=trace`.
- SAPIENT does not overwrite stricter source facts without an explicit ownership contract.

## Known Gaps

- The official test harness has not been run by SemOps.
- The official harness is Windows-focused and requires PostgreSQL 12, so CI automation needs a deliberate plan.
- A portable Linux/CI-friendly preflight suite does not exist yet; creating one would be a meaningful ecosystem
  contribution.
- No local SAPIENT protobuf bindings or binary decoder exists.
- No full official fixture corpus is vendored yet; redistribution and attribution should be checked before committing
  copies beyond trimmed test shapes.
- No SemOps mapping exists for SAPIENT node identity, detection lifecycle, tasking, alert acknowledgements, or Apex
  middleware interop.

## Adversarial Feed-Entry Questions

- Are we using authoritative artifacts, or just schema-shaped guesses?
- Does any compliance claim name the suite and run command?
- Are tasking/control semantics separated from detection state?
- Are malformed messages rejected before graph writes?
- Are licensing constraints compatible with checked-in fixtures?
- Are we treating Apex as an interop/middleware reference rather than outsourcing SemOps product semantics to it?
