# TAK/CoT Feed Evidence

Status: native parser gate started, still blocked from structural adapter status by `COP-007` and `COP-008`.

## Decision

TAK/CoT should be the second feed because it adds operator dots, markers, and chat to the COP quickly. SemLink has a
usable clean-room CoT subset and transport bridge that SemOps can port or pattern-match, but no public TAK/CoT
compliance suite was verified. Treat TAK as fixture/replay/interoperability-tested until stronger evidence exists.

## Local Evidence

- `pkg/adapters/cot` now contains a SemOps-local dependency-light XML CoT codec for the first fixture/replay gate.
- `/Users/coby/Code/c360/semlink/internal/cot/cot.go` contains a dependency-light XML CoT codec for air tracks,
  operator positions, markers, GeoChat, and alerts.
- `/Users/coby/Code/c360/semlink/internal/tak/bridge.go` supports outbound multicast/TCP and inbound UDP/TCP paths,
  decodes inbound CoT, translates to COP state, updates the store, and writes graph projections.
- `/Users/coby/Code/c360/semlink/scripts/demo-up.sh` opens inbound TAK UDP by default and seeds ALPHA/BRAVO operator
  dots, a North Gate marker, and GeoChat messages.
- `/Users/coby/Code/c360/semlink/docs/adr/002-tak-cot-bridge.md` records SemLink's limited Tier 0/Phase 1 scope and
  warns against building a full TAK Server inside that demo bridge.
- `docs/feed-product-roadmap.md` records the SemOps correction: future TAK Server-equivalent capability is product
  roadmap scope as a SemStreams-backed SemOps service, but not MVP scope.

## External Evidence

- No public TAK/CoT compliance suite was verified.
- SemLink intentionally avoided importing or studying AGPL `goatak` internals; keep the same legal posture unless a
  separate review changes it.

## Gates

### Parser Gate

Target command in the SemOps port:

```bash
go test ./pkg/adapters/cot
```

Acceptance:

- Operator position, marker, GeoChat, air-track, and malformed XML fixtures are decoded deterministically.
- Missing UID or event type is rejected before projection.
- `time`, `start`, and `stale` values are parsed and preserved for freshness behavior.

Current evidence:

- `go test ./pkg/adapters/cot` passes against SemLink-style ALPHA/BRAVO seed shapes, marker fixtures, GeoChat remarks
  fallback, air-track marshal/unmarshal, classifier checks, and malformed input rejection.

### Mock Transport Gate

Target command in the SemOps stack:

```bash
SEMLINK_TAK_SEED=true ./scripts/demo-up.sh
```

SemOps should replace this with an equivalent scenario-runner fixture rather than depending on SemLink at runtime.

Acceptance:

- UDP seed events can produce ALPHA/BRAVO operator dots, a marker, and GeoChat in the COP.
- TCP inbound fixture support exists before claiming TCP coverage.
- The feed can be run with deterministic local fixtures and no TAK Server.

### Projection Gate

Target command after SemOps graph contracts exist:

```bash
go test ./internal/projectors/tak
```

Acceptance:

- Operator positions project as current-state `signal` entities.
- Markers and task-like state project as `control`.
- GeoChat text projects as `content` or separate content evidence, not hidden in a high-rate position entity.
- Native CoT event references and replay steps are `trace`.
- CoT UIDs are preserved for audit and collision-safe SemOps entity IDs are derived.

### Replay Gate

Target artifact:

- A fixture pack with operator position, marker, GeoChat, alert, stale event, malformed XML, and duplicate UID cases.

Acceptance:

- Replaying the fixture yields deterministic COP state, provenance facts, and stale-data behavior.

## Known Gaps

- No verified public TAK/CoT conformance suite.
- No SemOps-local TAK transport, replay store, projector, graph writer, or UI feed state yet.
- Tasking remains out of scope until a dedicated operator-value and protocol review.
- Cross-source identity resolution between TAK-reported UAVs and MAVLink UAVs is out of scope for Phase 1.

## Adversarial Feed-Entry Questions

- Are we accidentally building TAK Server behavior inside the MVP CoT bridge instead of preserving a later service
  boundary?
- Is "compliance" language avoided unless a real suite or official schema is found?
- Does chat text get the right `content` treatment, or is it buried in a position event?
- Are stale events visible to operators rather than silently retained as fresh state?
- Are CoT UIDs preserved while SemOps entity IDs remain collision safe?
