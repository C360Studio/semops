# ADS-B Structural Runner Review

Date: 2026-06-20
Scope: `COP-007` opt-in ADS-B scenario-runner replay through the hosted adapter seam

Update: the 2026-06-26 product-evidence review reclassifies this as contract-mode structural replay only. Product
ADS-B evidence uses the hosted ADS-B HTTP component path, and product mode rejects
`SEMOPS_SCENARIO_ADSB_FIXTURE=true`.

## Decision

Accept ADS-B as an optional structural scenario replay source for contract mode.
`SEMOPS_SCENARIO_MODE=contract SEMOPS_SCENARIO_ADSB_FIXTURE=true` may append the deterministic OpenSky-shaped fixture
snapshots to the hosted HADR runner, route them through `internal/adapters/adsb`, and write token-backed graph
mutations. Keep live OpenSky, receiver/readsb/dump1090, ASTERIX, product e2e, and aircraft association out of this
acceptance gate.

## Objections Reviewed

- The default MVP stack must not imply live ADS-B coverage. The Compose service passes the flag through but defaults
  it to false.
- The scenario runner should not bypass the hosted adapter. ADS-B replay now depends on the `ADSBSink` interface and
  uses the same adapter seam that live or file-backed sources will use later.
- Runtime ADS-B writes must not repeat the old auto-vivify mistake. The scenario runner appends
  `semops.feed.adsb` ownership only when ADS-B replay is enabled, so structural ADS-B writes receive a
  SemStreams-minted owner token before graph mutation without widening the default MVP owner set.
- OpenSky JSON is only the first sample shape. It does not prove raw ADS-B, ASTERIX, or surveillance-grade product
  reliability.
- Cross-source aircraft association remains fusion-owned. The ADS-B adapter still projects source-partitioned tracks
  and emits no source-asset or cross-source association edges.

## Evidence

- `internal/scenario` replays ADS-B snapshots through an adapter sink instead of direct projector/writer plumbing.
- `cmd/semops-scenario-runner` adds ADS-B fixture records only in contract mode when
  `SEMOPS_SCENARIO_ADSB_FIXTURE=true`.
- `compose.cop.yml` passes `SEMOPS_SCENARIO_ADSB_FIXTURE` through to the scenario runner with a default of `false`,
  while `SEMOPS_SCENARIO_MODE` defaults to `product`.
- Scenario-runner ownership registration appends `OwnerADSB` plus `ADSBTrackContract` only when the ADS-B fixture
  flag is enabled.
- Targeted evidence command: `go test ./internal/scenario ./cmd/semops-scenario-runner ./pkg/cop ./internal/copownership -count=1`.

## Follow-Ups

- Use hosted ADS-B HTTP component smokes for product evidence; reserve ADS-B scenario fixture replay for focused
  contract checks.
- Decide whether live mode starts with optional OpenSky, local receiver/readsb/dump1090 fixtures, or a dedicated
  adapter service.
- Keep statistical aircraft association behind a separate fusion review.
