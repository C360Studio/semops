# SAPIENT Feed Evidence

Status: JSON and binary descriptor preflight, raw replay, preflight input/decoder components, opt-in app-runtime
preflight wiring, and a narrow absolute-location detection projection contract exist. Harness qualification and
runtime graph-production review are still required before product support or conformance claims.

## Decision

SAPIENT has moved from artifact discovery to a parser/harness planning lane. SemOps now has a dependency-light JSON
preflight parser, a descriptor-based binary protobuf preflight, bounded raw replay, a SemStreams raw HTTP
input -> decoder processor component chain for representative BSI Flex 335 v2 message shapes, and a reviewed first
projection contract for absolute-location detection reports only. It must not claim SAPIENT product support or
compliance until a documented Dstl harness run proves the boundary. `cmd/semops` may now compose the preflight
input -> decoder chain behind `SEMOPS_SAPIENT_ENABLED=true`, but it still produces raw/decoded streams only.

Runtime graph projection remains blocked. SemOps now has `OwnerSAPIENT`, a source-partitioned
`semops.cop.track.sapient-detection-current-state` contract, a pure projector that plans create/update mutations, and
COP API readback for prefix-discovered SAPIENT tracks. It has no graph-producing hosted SAPIENT component or graph
writer by design. Range/bearing, UTM, tasking, alerts, lifecycle, Apex service behavior, and association semantics
remain later gates.
See `openspec/changes/revive-cop-product/reviews/2026-06-21-sapient-runtime-preflight-review.md` for the runtime
preflight boundary review.

## Local Evidence

- `pkg/adapters/sapient` parses Dstl-harness-shaped JSON preflight fixtures for registration, status report,
  detection report, and task acknowledgement messages.
- `go test ./pkg/adapters/sapient` validates required top-level envelope fields, UUID/ULID identity fields,
  mandatory content fields, location/range-bearing oneof behavior, and malformed fixture rejection before projection.
- `pkg/adapters/sapient` embeds the official Dstl BSI Flex 335 v2 `.proto` sources and license under
  `pkg/adapters/sapient/protos/sapient_msg`.
- `ParseBinaryMessage` compiles those sources through `github.com/bufbuild/protocompile`, decodes binary
  `SapientMessage` payloads with dynamic protobuf descriptors, and validates the result through the same preflight
  model.
- `pkg/adapters/sapient` stores JSON and protobuf payload bytes on a bounded raw lane, persists replay records as JSON
  Lines, and decodes replay through the same JSON/protobuf preflight boundary.
- `internal/components/sapient` provides an HTTP raw input component and decoder processor for SAPIENT preflight
  payloads, with `HTTPClientPort`, `TimerPort`, registered raw/decoded `message.BaseMessage` payloads, stream ports,
  replay capture, stale-source health, and no graph request ports.
- `cmd/semops` can opt into that preflight HTTP input -> decoder chain with `SEMOPS_SAPIENT_ENABLED=true`,
  `SEMOPS_SAPIENT_HTTP_URL`, explicit encoding, stale-source config, raw-lane caps, and optional replay capture.
- App-runtime SAPIENT preflight does not append ownership contracts, register `OwnerSAPIENT`, or subscribe any decoded
  graph projector path.
- `pkg/cop` now defines `OwnerSAPIENT` and a source-partitioned, signal-profiled track contract for absolute-location
  detection reports only.
- `internal/projectors/sapient` plans create/update graph mutations for `LOCATION_COORDINATE_SYSTEM_LAT_LNG_DEG_M`
  WGS84 detection reports and rejects range/bearing, UTM, unsupported datum, or invalid latitude/longitude inputs.
- `internal/api/cop` can read prefix-discovered SAPIENT track state back into the COP snapshot and source-health
  model.
- No SemOps SAPIENT generated Go bindings, product service adapter, graph-producing component, or graph writer exists
  yet.
- No local SAPIENT test harness run has been performed.
- The feed ladder assigns detections/tracks to `signal`, tasking/collection state to `control`, and native decode
  traces to `trace`.

## External Evidence

- GOV.UK documents SAPIENT as a Dstl/MOD-owned architecture and names BSI Flex 335 as the freely available ICD:
  <https://www.gov.uk/guidance/sapient-autonomous-sensor-system>.
- GOV.UK also links official GitHub assets for protobuf files, a SAPIENT Test Harness, and SAPIENT Middleware.
- BSI identifies `SAPIENT Network of Autonomous Sensors and Effectors - BSI Flex 335 V2:2024` as the current
  structure, format, and content reference for SAPIENT messages on bsigroup.com.
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

### Binary Protobuf Descriptor Gate

Target command:

```bash
go test ./pkg/adapters/sapient -run Protobuf
```

Acceptance:

- Official BSI Flex 335 v2 `.proto` files are generated or compiled through a reproducible toolchain. [done:
  descriptor compile]
- Binary `SapientMessage` payload fixtures decode into typed SemOps preflight messages. [done: representative subset]
- JSON preflight and binary decode agree on envelope, content kind, identity fields, and required-field failures.
  [done: representative subset]
- BSI Flex 335 v1/v2 drift is visible in package paths, generated-code provenance, or fixture metadata. [done:
  vendored v2 import path]

### Raw Replay Gate

Target command:

```bash
go test ./pkg/adapters/sapient -run 'RawLane|Replay'
```

Acceptance:

- Raw JSON and protobuf payload bytes are captured with source, receive time, encoding, content kind, and native
  identity metadata when parsing succeeds. [done: representative subset]
- Parser-failing bytes can still be retained when the native encoding is known. [done]
- Replay records reload from JSON Lines and decode back through the JSON/protobuf preflight boundary. [done]
- Raw payloads remain out of graph entities until a projection ownership/indexing review approves references or
  derived state. [open]

### Component Preflight Gate

Target command:

```bash
go test ./internal/components/sapient ./internal/contracts -run SAPIENT
```

Acceptance:

- SAPIENT HTTP raw ingress is represented as a SemStreams input component with `HTTPClientPort`, `TimerPort`, config
  schema, health, flow metrics, and explicit encoding handling. [done]
- Raw and decoded SAPIENT preflight messages use registered `message.BaseMessage` payloads and tappable stream
  subjects. [done]
- The decoder captures JSON/protobuf bytes into the SAPIENT replay store before publishing decoded preflight
  messages. [done]
- Malformed messages are captured and replayed before parse failure, without graph writes. [done]
- The component package exposes no graph request ports and does not register runtime SAPIENT owner claims. [done]

### App Runtime Preflight Gate

Target command:

```bash
go test ./internal/app -run SAPIENT
```

Acceptance:

- The hosted app composes SAPIENT as HTTP input -> decoder processor only when `SEMOPS_SAPIENT_ENABLED=true`. [done]
- Runtime ownership does not add SAPIENT owner claims or graph-producing contracts. [done]
- `SEMOPS_SAPIENT_REPLAY_PATH`, raw-lane bounds, HTTP URL, poll interval, stale-after, contact policy, and encoding
  are config/env-driven. [done]
- Local provider-shaped HTTP fixtures can drive raw -> decoded preflight streams and append replay without graph
  writes. [done]
- Compose passes SAPIENT runtime env through but defaults `SEMOPS_SAPIENT_ENABLED=false` and requires an explicit URL
  when enabled. [done]

### Generated Binding Gate

Target command if SemOps needs generated Go bindings rather than dynamic descriptors:

```bash
go test ./pkg/adapters/sapient -run Generated
```

Acceptance:

- Generation uses the same vendored Dstl proto source and records the generator version.
- Generated package paths preserve the BSI Flex 335 version boundary.
- Generated messages agree with the descriptor-based binary preflight fixtures.

### Projection Gate

Target command:

```bash
go test ./pkg/cop ./internal/projectors/sapient ./internal/api/cop
```

Acceptance:

- Detections and tracks use `indexing_profile=signal`. [done for absolute-location detection reports]
- Sensor tasking, collection plans, and alert state use `indexing_profile=control`.
- Native decode/replay records use `indexing_profile=trace`.
- SAPIENT does not overwrite stricter source facts without an explicit ownership contract. [done: source-partitioned
  track pattern and no foreign edges]
- First projection starts with absolute-location reports only, unless source sensor pose, reference frame, and
  uncertainty are available for range/bearing detections. [done: range/bearing rejected]
- UTM or other coordinate systems are rejected until a deliberate conversion and datum policy exists. [done]
- Associated detections, derived links, and cross-source correlation use fusion or evidence contracts rather than the
  SAPIENT adapter's source-owner contract. [done: no association edges]
- Runtime graph writes remain blocked until a graph-producing component/writer boundary is reviewed. [open]

## Known Gaps

- The official test harness has not been run by SemOps.
- The official harness is Windows-focused and requires PostgreSQL 12, so CI automation needs a deliberate plan.
- A portable Linux/CI-friendly preflight suite does not exist yet; creating one would be a meaningful ecosystem
  contribution.
- No generated SAPIENT Go bindings exist.
- No full official fixture corpus is vendored yet; redistribution and attribution should be checked before committing
  copies beyond trimmed test shapes.
- No SemOps mapping exists for SAPIENT node identity, detection lifecycle, tasking, alert acknowledgements, or Apex
  middleware interop.
- No product-hosted SAPIENT service or runtime graph projection path exists yet; the opt-in app-runtime chain is
  preflight only.
- No graph-producing SAPIENT component or SAPIENT graph writer exists yet; that is intentional until runtime
  projection ownership, backpressure, service mode, and harness scope are reviewed.

## Adversarial Feed-Entry Questions

- Are we using authoritative artifacts, or just schema-shaped guesses?
- Does any compliance claim name the suite and run command?
- Are tasking/control semantics separated from detection state?
- Are malformed messages rejected before graph writes?
- Are licensing constraints compatible with checked-in fixtures?
- Are we treating Apex as an interop/middleware reference rather than outsourcing SemOps product semantics to it?
