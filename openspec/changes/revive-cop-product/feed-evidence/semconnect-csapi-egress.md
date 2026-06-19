# SemConnect CS API Egress Evidence

Status: standards-facing egress candidate after structural graph state exists.

## Decision

SemConnect CS API is not a Phase 1 input feed. It should be used as a standards-facing egress path once SemOps has
governed systems, sensors, datastreams, observations, deployments, and events worth projecting.

## Local Evidence

- `/Users/coby/Code/c360/semconnect/conformance/run.sh` boots NATS, `semstreams-backend`, `cs-api-server`, and
  Team Engine, seeds fixtures, invokes the ETS, and archives TestNG reports plus service logs.
- `/Users/coby/Code/c360/semconnect/conformance/README.md` states the current Stage 55 pinned suite is green:
  `total=137 passed=137 failed=0 skipped=0`.
- The SemConnect harness exercises real graph reads/writes, observation publish/readback, artifact storage,
  discovery, OpenAPI, content negotiation, and claimed CS API conformance classes.
- SemLink already demonstrates a CS API bridge pattern for current vehicle/COP state.

## External Evidence

- SemConnect uses OGC Team Engine and the Botts CS API ETS as its conformance harness.

## Gates

### Projection Readiness Gate

Target command after SemOps exposes structural graph state:

```bash
go test ./internal/egress/csapi
```

Acceptance:

- SemOps can map a COP asset/platform to a CS API System.
- SemOps can map a sensor and current observation to CS API datastream/observation surfaces.
- SemOps can preserve source provenance and ownership when projecting to egress.
- CS API egress remains a view and does not decide SemOps indexing profiles.

### Harness Gate

Target command in the SemConnect checkout:

```bash
./conformance/run.sh
```

Acceptance:

- The harness runs end to end and archives TestNG XML.
- Any pass/fail count is read from TestNG output, not inferred from `go test`.
- If SemOps-specific egress changes SemConnect mappings, the conformance delta is recorded.

### Replay Gate

Target artifact:

- A SemOps graph fixture with one asset/platform, one hosted sensor, one datastream, one observation, one deployment,
  and one system event.

Acceptance:

- The fixture projects deterministically through SemConnect.
- SemConnect conformance remains a separate acceptance gate.

## Known Gaps

- SemOps does not yet expose the canonical graph state needed for meaningful CS API egress.
- CS API egress should not block Phase 1 structural COP.
- Egress can be fully conformant while SemOps feed ingestion is still incomplete; keep those claims separate.

## Adversarial Feed-Entry Questions

- Are we using SemConnect as egress rather than moving COP product ownership into SemConnect?
- Does the SemOps graph own indexing/provenance before CS API projection?
- Does the conformance result come from the actual harness output?
- Are SemOps workflows driving egress requirements, or are we chasing standards coverage too early?

## Source Links

- OGC Team Engine: <https://github.com/opengeospatial/teamengine>
