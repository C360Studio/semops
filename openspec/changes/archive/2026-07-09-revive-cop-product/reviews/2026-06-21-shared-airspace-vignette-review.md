# Shared-Airspace Vignette Review

Date: 2026-06-21
Scope: Product-visible HADR flood/evacuation plus ADS-B shared-airspace smoke evidence

## Decision

Accept the combined COP snapshot smoke as the first shared-airspace vignette gate.

The value is not another feed-specific assertion. The value is proving that the operator surface can show the HADR
scenario context and an air-picture feed in one curated COP snapshot after the SemStreams graph has accepted governed
writes from independent component flows.

## Boundaries

- ADS-B evidence comes from the hosted ADS-B HTTP input -> decoder -> graph-projector component chain against the local
  fixture provider.
- `semops-scenario-runner` remains responsible for the HADR MAVLink, TAK/CoT, and CAP replay path and does not claim
  a second `semops.feed.adsb` writer in the default Compose smoke.
- This is fixture-backed shared-airspace evidence, not live OpenSky reliability, ASTERIX support, receiver protocol
  support, deconfliction, or statistical track association.
- The adversarial Phase 1 signoff review remains open because broader demo credibility, graph/index cardinality, and
  monitoring behavior still need a full-stage review.

## Risks

- The combined test can be misread as service maturity. The test and docs must keep "local fixture" and component-flow
  boundaries visible.
- One snapshot can prove coexistence but not temporal reasoning. Track association, stale windows, and conflict
  handling belong in later tier-escalation work.
- Using the same ADS-B owner from both the scenario runner and hosted app would create ownership ambiguity. Keep the
  smoke path on the hosted ADS-B component flow unless the owner model is deliberately changed.

## Evidence

- `internal/smoke/cop` asserts one Caddy-routed snapshot contains scenario MAVLink/TAK/CAP state and an ADS-B aircraft
  track when the local ADS-B HTTP component is enabled.
- `scripts/cop-stack-smoke.sh` includes the combined shared-airspace smoke in the one-command stack run.
- `openspec/changes/revive-cop-product/specs/containerized-demo-infra/spec.md` records the ownership and fixture
  boundaries.
