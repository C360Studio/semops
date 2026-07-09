# Full Stack Smoke Evidence

Date: 2026-06-21
Command: `bash scripts/cop-stack-smoke.sh`

## Result

Pass.

The full one-command Compose smoke built and started NATS, SemStreams, SemOps API/runtime, SemOps UI, Caddy, the
scenario runner, and the local ADS-B/SAPIENT fixture provider. It then verified active scenario status, Caddy-routed
COP snapshot readback, direct live graph smokes, and SAPIENT decoded-stream preflight.

## Evidence Observed

- Scenario runner status reached `state=succeeded`, `completed=10`, `failed=0`.
- `TestHostedCOPSnapshotReflectsMAVLinkUDP` passed.
- `TestHostedCOPSnapshotReflectsCoTUDP` passed.
- `TestHostedCOPSnapshotReflectsScenarioRunner` passed.
- `TestHostedCOPSnapshotReflectsADSBHTTPProvider` passed.
- `TestHostedCOPSnapshotReflectsHADRSharedAirspace` passed.
- `TestLiveGraphMAVLinkBornFirstSmoke` passed.
- `TestLiveGraphCoTBornFirstSmoke` passed.
- `TestLiveGraphCAPBornFirstSmoke` passed.
- `TestLiveSAPIENTPreflightDecodedSmoke` passed.
- The smoke tore the Compose stack down and removed the temporary NATS volume.

## Caveats

- The run used local fixtures for ADS-B and SAPIENT and deterministic scenario playback for HADR. It is not live
  OpenSky, live NWS/IPAWS, SAPIENT conformance, TAK Server, CS API bridge, KLV, or production-readiness evidence.
- SemStreams logged the known append-evidence warning for `semops.feed.cap`: the registration has no enforceable
  ownership or foreign-edge claim because CAP is intentionally evidence-only. This remains expected until the
  framework has first-class evidence-contribution registration semantics separate from ownership/write fences.
- The UI build still emits Vite large chunk warnings for the MapLibre/deck.gl bundle. This is not a smoke failure, but
  it remains a future performance/code-splitting consideration.
