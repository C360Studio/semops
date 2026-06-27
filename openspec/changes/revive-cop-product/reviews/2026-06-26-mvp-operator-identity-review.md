# MVP Operator Identity Review

Date: 2026-06-26

## Scope

Review the first SemOps COP operator identity path for association-review audit. This does not authorize command
execution, identity fusion, upstream CS API status, compliance workflows, or scenario controls.

## Decision

Accept a deliberately low-friction MVP policy:

- No login or external identity provider is required for the demo.
- `X-SemOps-Operator-ID` may stamp the reviewer's local audit label.
- If the header is absent, the API accepts the existing request-body `reviewed_by`; if that is absent, it falls back to
  `operator.local`.
- Every review remains `operator.unverified` and `local.display_only`.
- Role or authority-scope headers that claim anything stronger are rejected.

## Risks

- Header identity is not authentication and can be spoofed by any caller with API access.
- Local audit labels should not be reused as compliance, tasking, command-control, or upstream status authority.
- The policy is intentionally unsuitable for multi-authority operations without a real identity provider and
  arbitration model.

## Required Follow-Up

- Close task 8.31 only when authenticated identity and multi-authority conflict arbitration are implemented.
  [done: `2026-06-27-authenticated-association-review-arbitration.md`]
- Keep `/scenario/controls` fail-closed until control checkpoints, authenticated operator policy, conflict semantics,
  and an executor exist.
- Require another adversarial review before any operator review can drive command execution, identity fusion, CS API
  egress, or compliance claims.
