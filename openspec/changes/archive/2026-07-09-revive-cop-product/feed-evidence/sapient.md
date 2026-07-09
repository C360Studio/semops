# SAPIENT Feed Evidence

Status: JSON and binary descriptor preflight, raw replay, preflight input/decoder components, opt-in app-runtime
preflight wiring, a narrow absolute-location detection projection contract, and an explicitly graph-gated runtime
projector component exist. Harness qualification, service-mode review, and broader message semantics are still
required before product support or conformance claims.

## Decision

SAPIENT has moved from artifact discovery to a parser/harness planning lane. SemOps now has a dependency-light JSON
preflight parser, a descriptor-based binary protobuf preflight, bounded raw replay, a SemStreams raw HTTP
input -> decoder processor component chain for representative BSI Flex 335 v2 message shapes, a reviewed first
projection contract for absolute-location detection reports only, and a graph-projector processor component that is
off by default. It must not claim SAPIENT product support or compliance until a documented Dstl harness run proves the
boundary.

Runtime graph projection is no longer globally blocked, but it is sharply gated. `cmd/semops` may compose only the
input -> decoder chain behind `SEMOPS_SAPIENT_ENABLED=true`; it registers `OwnerSAPIENT`, subscribes the decoded graph
projector, and writes graph mutations only when `SEMOPS_SAPIENT_GRAPH_ENABLED=true`. SemOps now has
`OwnerSAPIENT`, a source-partitioned `semops.cop.track.sapient-detection-current-state` contract, a pure projector
that plans create/update mutations, a graph writer, a projector component with SemStreams request ports, and COP API
readback for prefix-discovered SAPIENT tracks. Range/bearing, UTM, tasking, alerts, lifecycle, Apex service behavior,
and association semantics remain later gates.
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
- SemOps will keep descriptor-based dynamic protobuf decoding for the current SAPIENT preflight and runtime component
  path. Generated Go bindings are deferred until product service mode, outbound tasking, full protobuf round-trip
  behavior, or performance profiling proves they are needed. See
  `openspec/changes/revive-cop-product/reviews/2026-06-23-sapient-generated-bindings-review.md`.
- `pkg/adapters/sapient` stores JSON and protobuf payload bytes on a bounded raw lane, persists replay records as JSON
  Lines, and decodes replay through the same JSON/protobuf preflight boundary.
- `internal/components/sapient` provides an HTTP raw input component and decoder processor for SAPIENT preflight
  payloads, with `HTTPClientPort`, `TimerPort`, registered raw/decoded `message.BaseMessage` payloads, stream ports,
  replay capture, stale-source health, and no graph request ports.
- `internal/components/sapient` also provides a decoded-message graph projector processor with SemStreams
  `NATSRequestPort` graph mutation ports. It consumes only registered decoded payloads and emits graph writes through
  the reviewed absolute-location projection plan.
- `cmd/semops` can opt into that preflight HTTP input -> decoder chain with `SEMOPS_SAPIENT_ENABLED=true`,
  `SEMOPS_SAPIENT_HTTP_URL`, explicit encoding, stale-source config, raw-lane caps, and optional replay capture.
- App-runtime SAPIENT preflight does not append ownership contracts, register `OwnerSAPIENT`, or subscribe any decoded
  graph projector path unless `SEMOPS_SAPIENT_GRAPH_ENABLED=true`.
- App-runtime SAPIENT graph projection registers `OwnerSAPIENT`, passes SemStreams-minted owner tokens into the
  projector, composes the decoded-message projector, and uses graph request/reply mutation writes only under
  `SEMOPS_SAPIENT_GRAPH_ENABLED=true`.
- `cmd/semops-feed-fixtures` serves `/sapient/messages` for the task-ack preflight smoke and `/sapient/detections`
  for deterministic absolute-location detection projection development.
- `fixtures/sapient/task-ack.json` and `fixtures/sapient/absolute-detection.json` are committed portable fixtures for
  those two runtime fixture-service paths. They are manifest-listed representative JSON, not captured Apex traffic,
  Dstl harness output, or SAPIENT compliance evidence.
- `pkg/cop` now defines `OwnerSAPIENT` and a source-partitioned, signal-profiled track contract for absolute-location
  detection reports only.
- `internal/projectors/sapient` plans create/update graph mutations for `LOCATION_COORDINATE_SYSTEM_LAT_LNG_DEG_M`
  WGS84 detection reports and rejects range/bearing, UTM, unsupported datum, or invalid latitude/longitude inputs.
- `internal/api/cop` can read prefix-discovered SAPIENT track state back into the COP snapshot and source-health
  model.
- No SemOps SAPIENT product service adapter, tasking surface, association model, range/bearing conversion, or UTM
  conversion exists yet. Generated Go bindings are deliberately deferred rather than treated as a current blocker.
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
- SemOps portable runtime fixtures are manifest-listed and checked against the local fixture-service payloads. [done]

### Harness Qualification Gate

Target environment after implementation planning:

- Windows 10/11 VM or workstation with .NET 6 SDK, PostgreSQL 12, and the Dstl BSI Flex 335 v2 Test Harness.

Acceptance:

- Harness version, commit, and configuration are recorded.
- SemOps-generated registration, status, detection, alert, and task-ack messages are exercised where applicable.
- Failures are captured as either SemOps bugs, unsupported SAPIENT subset, or upstream tooling issues.
- No compliance wording appears in demo materials without the harness result and scope.

### Portable Preflight Suite Gate

Decision:

- A portable Linux/CI-friendly SAPIENT preflight suite would be valuable as an ecosystem contribution and developer
  preflight gate.
- It is not required for the SemOps COP MVP because SemOps already has bounded parser, descriptor-binary, replay,
  component-flow, and graph-projection evidence for its narrow SAPIENT subset.
- It must not be described as compliance unless Dstl, BSI, or another accepted authority treats it as a substitute for
  the official harness.
- It should reuse official BSI Flex 335 v2 protobufs and fixture corpora where redistribution allows, and use
  synthetic fixtures only when clearly labeled as developer interoperability evidence.
- See `openspec/changes/revive-cop-product/reviews/2026-06-23-sapient-portable-preflight-suite-review.md`.

Future effort:

- Create or contribute a cross-platform SAPIENT preflight suite that can run in Linux CI from official BSI Flex 335 v2
  protobufs and fixture corpora.

Acceptance:

- The suite runs without Windows-only assumptions, PostgreSQL 12 pinning, or manual GUI setup.
- It covers parser validation, mandatory-field failures, representative registration/status/detection/task/alert
  messages, and malformed binary payloads.
- It is described as developer preflight or interoperability evidence until Dstl, BSI, or another accepted authority
  treats it as a compliance substitute.
- SemOps records this as evaluated and deferred from the MVP critical path. [done]

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
- The HTTP input and decoder components expose no graph request ports and do not register runtime SAPIENT owner
  claims. [done]

### App Runtime Preflight Gate

Target command:

```bash
go test ./internal/app -run SAPIENT
```

Acceptance:

- The hosted app composes SAPIENT as HTTP input -> decoder processor only when `SEMOPS_SAPIENT_ENABLED=true`. [done]
- Runtime ownership does not add SAPIENT owner claims or graph-producing contracts when
  `SEMOPS_SAPIENT_GRAPH_ENABLED=false`. [done]
- `SEMOPS_SAPIENT_REPLAY_PATH`, raw-lane bounds, HTTP URL, poll interval, stale-after, contact policy, and encoding
  are config/env-driven. [done]
- Local provider-shaped HTTP fixtures can drive raw -> decoded preflight streams and append replay without graph
  writes. [done]
- Compose passes SAPIENT runtime env through but defaults `SEMOPS_SAPIENT_ENABLED=false` and requires an explicit URL
  when enabled. [done]

### Generated Binding Gate

Decision:

- Do not add generated SAPIENT Go bindings for the current preflight, raw replay, component-flow, or
  absolute-location graph projection work.
- Keep the embedded Dstl BSI Flex 335 v2 proto source as the authoritative version boundary for dynamic descriptor
  compilation.
- Reopen this gate when SemOps needs product service mode, outbound SAPIENT tasking, exact typed protobuf
  round-trips, broad message coverage, or measured performance improvement that dynamic descriptors cannot satisfy.
- If generated bindings become necessary, prefer a reproducible buf-based generation workflow and record the Dstl
  proto commit plus generator versions.

Future target command if SemOps needs generated Go bindings:

```bash
go test ./pkg/adapters/sapient -run Generated
```

Acceptance:

- Current descriptor-based binary preflight covers representative SAPIENT JSON/protobuf decode without generated
  bindings. [done]
- Generated bindings are not required before SAPIENT graph projection for reviewed absolute-location detections.
  [done]
- Generation uses the same vendored Dstl proto source and records the generator version. [deferred]
- Generated package paths preserve the BSI Flex 335 version boundary. [deferred]
- Generated messages agree with the descriptor-based binary preflight fixtures. [deferred]

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
- Runtime graph writes are allowed only behind `SEMOPS_SAPIENT_GRAPH_ENABLED=true` and only for the reviewed
  absolute-location detection contract. [done]

### App Runtime Graph Gate

Target command:

```bash
go test ./internal/app ./internal/components/sapient ./internal/projectors/sapient ./cmd/semops-feed-fixtures
```

Acceptance:

- `SEMOPS_SAPIENT_ENABLED=true` with graph mode disabled composes only the HTTP input and decoder processor. [done]
- `SEMOPS_SAPIENT_GRAPH_ENABLED=true` requires `SEMOPS_SAPIENT_ENABLED=true` and a positive graph write timeout.
  [done]
- Runtime ownership adds `OwnerSAPIENT` only when graph mode is enabled. [done]
- The projector component consumes registered decoded-message payloads and writes graph mutations through SemStreams
  graph request ports. [done]
- Create-conflict reconciliation marks existing SAPIENT track births and retries as updates without falling back to
  auto-vivify. [done]
- Non-detection messages such as task acknowledgements remain decoded-stream evidence and do not produce graph
  mutations. [done]

### Opt-In Stack Graph Smoke Gate

Target command:

```bash
SEMOPS_COP_SMOKE_SAPIENT_GRAPH_ENABLED=true bash scripts/cop-stack-smoke.sh
```

Acceptance:

- The stack smoke keeps default SAPIENT behavior as decoded-stream preflight against `/sapient/messages`. [done]
- Graph smoke explicitly enables `SEMOPS_SAPIENT_GRAPH_ENABLED=true` and points SAPIENT at `/sapient/detections`.
  [done in harness wiring]
- The Caddy-routed COP snapshot contains a SAPIENT track with `semops.feed.sapient` provenance. [done in smoke
  assertion]
- The Caddy-routed SAPIENT graph smoke accepts stale feed health for deterministic old-source fixtures while still
  requiring SAPIENT source, non-zero position, `semops.feed.sapient` provenance, and non-empty source reference.
  [done]
- Prometheus component metrics and `GET /api/cop/runtime` expect the SAPIENT projector only when graph smoke is
  enabled. [done in smoke assertion]
- The direct SAPIENT decoded-stream smoke validates `detectionReport` in graph mode and `taskAck` in default
  preflight mode. [done]
- `SEMOPS_COP_SMOKE_SAPIENT_GRAPH_ENABLED=true bash scripts/cop-stack-smoke.sh` passed on 2026-06-23 with KLV and
  weather still skipped unless their stack flags are enabled. [done]
- This smoke remains fixture-backed engineering evidence and does not claim SAPIENT product service support,
  compliance, tasking, association, UTM conversion, range/bearing conversion, or Apex middleware behavior. [done]

## Known Gaps

- The official test harness has not been run by SemOps.
- The official harness is Windows-focused and requires PostgreSQL 12, so CI automation needs a deliberate plan.
- A portable Linux/CI-friendly preflight suite does not exist yet; creating one would be a meaningful ecosystem
  contribution.
- Generated SAPIENT Go bindings do not exist by design; descriptor-based decode remains the current product boundary.
- No full official fixture corpus is vendored yet; redistribution and attribution should be checked before committing
  copies beyond trimmed test shapes.
- No SemOps mapping exists for SAPIENT node identity, detection lifecycle, tasking, alert acknowledgements, or Apex
  middleware interop.
- No product-hosted SAPIENT service exists yet; the opt-in app-runtime chain is a SemStreams component flow, not a
  SAPIENT-facing product service.
- Runtime graph projection currently covers only absolute-location detection reports. Tasking, alert acknowledgements,
  association, range/bearing conversion, UTM conversion, detection lifecycle, and Apex middleware interop remain
  intentionally open.

## Adversarial Feed-Entry Questions

- Are we using authoritative artifacts, or just schema-shaped guesses?
- Does any compliance claim name the suite and run command?
- Are tasking/control semantics separated from detection state?
- Are malformed messages rejected before graph writes?
- Are licensing constraints compatible with checked-in fixtures?
- Are we treating Apex as an interop/middleware reference rather than outsourcing SemOps product semantics to it?
