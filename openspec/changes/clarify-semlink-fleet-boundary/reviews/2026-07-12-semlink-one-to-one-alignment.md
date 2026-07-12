# SemLink One-To-One Alignment Review

Date: 2026-07-12

SemLink updated its side in commit `d7ce086c7cdca83ac7588e975272e0db1bf2b502`
(`chore(companion): align mesh proof and onboarding`) to align the mesh proof
with the SemOps fleet boundary.

## SemLink Disposition

- The default mesh proof is `3` companion nodes with `1` simulated MAVLink
  vehicle per node.
- The public mesh demo no longer exposes `-vehicles-per-node`.
- `internal/companion.NewHarness` creates one simulator vehicle per companion
  node, so tests fail early if the deterministic mesh proof drifts back toward
  multi-autopilot companion ownership.
- The SemLink OpenSpec change `prove-one-vehicle-companion-mesh` records
  multi-vehicle-behind-one-SemLink-runtime as a non-goal for MVP.

## SemOps Acceptance

This satisfies the SemOps boundary decision in this change: SemLink owns the
vehicle-local companion proof, while SemOps owns fleet aggregation,
mesh/multiplex handling, cross-companion target derivation, and COP display.

SemOps should keep accepting SemLink simple-mesh artifacts as multi-companion
evidence. It should not require or infer a SemLink runtime that multiplexes more
than one ArduPilot vehicle unless a later reviewed profile explicitly declares
that aggregator mode.

## Verification

Run in `/Users/coby/Code/c360/semlink` against SemLink commit
`d7ce086c7cdca83ac7588e975272e0db1bf2b502`:

```sh
openspec validate --all --strict
go test ./internal/e2e ./cmd/semlink-demo ./internal/companion -count=1
```

Results:

- OpenSpec: `8 passed, 0 failed`
- Focused Go tests: passed
