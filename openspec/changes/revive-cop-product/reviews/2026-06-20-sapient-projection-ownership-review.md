# SAPIENT Projection Ownership Review

Date: 2026-06-20

## Decision

Keep SAPIENT graph writes blocked.

The JSON preflight, descriptor-based protobuf preflight, bounded raw lane, and replay store are useful parser evidence.
They do not yet justify a SAPIENT owner constant, projection contract, hosted component package, or compliance claim.
SemOps should not add `OwnerSAPIENT` until the first graph projection has a reviewed entity model, indexing profile,
source identity policy, and test-harness or explicit non-compliance-demo scope.

The first acceptable projection should be narrow:

- Absolute-location detection reports may become source-partitioned `track`, `observation`, or future detection
  entities with `signal` indexing after the canonical entity choice is made.
- Range/bearing detections must not be projected as coordinates until the source sensor pose, reference frame, and
  uncertainty are available.
- Sensor registration, status, tasking, collection plans, and alert acknowledgements are `control` candidates, not
  detection-state shortcuts.
- Native decode and replay records remain `trace` or raw-lane artifacts, not graph entities by default.
- Associated detections, derived links, and multi-sensor correlation belong to fusion or evidence claims, not to the
  SAPIENT adapter's source-owner contract.

## Objections Raised

- SAPIENT's message richness makes it tempting to create graph state before SemOps understands the source semantics.
- A generated protobuf binding would not solve ownership. It would only make invalid projection easier to write.
- Range/bearing messages are source-relative. Turning them into global tracks without sensor pose would manufacture
  false precision.
- Associated detections can look like cross-source truth. They must not become authoritative association edges without
  a fusion contract.
- Apex middleware is useful service-shape evidence, but outsourcing product semantics to it would weaken SemOps'
  ownership, freshness, command-authority, and indexing rules.
- The Dstl harness is real evidence, but Windows-only, .NET 6, and PostgreSQL 12 requirements mean it is not a
  drop-in Linux CI gate.

## Evidence Checked

- `pkg/adapters/sapient` JSON preflight parser for representative BSI Flex 335 v2 message shapes.
- Dynamic protobuf decode through vendored Dstl v2 `.proto` sources and `github.com/bufbuild/protocompile`.
- Bounded JSON/protobuf raw lane and JSON Lines replay support.
- `pkg/cop/contracts.go`, which intentionally has no SAPIENT owner constant or projection contract today.
- GOV.UK SAPIENT guidance, BSI Flex 335 v2 references, Dstl protobuf assets, Dstl BSI Flex 335 v2 Test Harness, and
  Apex SAPIENT Middleware anchors recorded in the feed evidence docs.

## Accepted Risks

- SemOps can claim SAPIENT parser preflight and replay evidence, but not SAPIENT product support or compliance.
- SAPIENT cannot enter the structural graph stack until a narrow projection contract is reviewed and tested.
- Hosted SAPIENT ingress remains blocked until service mode is chosen: file replay, TCP/UDP, Apex REST/middleware,
  NATS, or external HTTP polling/client behavior.

## Follow-Up Tasks

- Run or qualify the Dstl BSI Flex 335 v2 Test Harness before compliance language appears.
- Decide whether generated Go bindings add value beyond dynamic descriptor decode.
- Define the first SAPIENT source identity and projection contract before adding `OwnerSAPIENT`.
- Use SemStreams input and processor components for hosted SAPIENT ingress once service mode is chosen.
- Revisit SemStreams issue #310 if SAPIENT/Apex integration needs reusable HTTP polling/client metadata.
