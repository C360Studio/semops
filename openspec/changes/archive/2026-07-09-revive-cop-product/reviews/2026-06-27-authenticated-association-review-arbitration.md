# Authenticated Association Review Arbitration

Date: 2026-06-27
Scope: authenticated operator identity and multi-authority conflict arbitration for fusion association review

## Decision

Accept the first authenticated association-review path as an opt-in authority gate. The default COP remains
display-only and unauthenticated. Hosted SemOps may switch to `SEMOPS_COP_OPERATOR_IDENTITY_MODE=trusted_headers`
when an upstream authentication boundary owns the trusted identity headers.

Authenticated reviews carry `operator.authenticated`, `association.review`, authority domain, authenticated flag,
and `multi_authority_blocks_conflicts`. Matching decisions across authority domains converge into a consensus current
review. Conflicting decisions across authority domains produce `decision=blocked_conflict` and
`conflict_state=blocked_conflict` instead of allowing a latest-writer override.

This is enough to close task `8.31` as the review-decision authority gate. It does not authorize command execution,
identity fusion, upstream CS API status publication, compliance workflow decisions, or automatic association.

## Red-Team Findings

1. Trusted headers are not authentication by themselves.

   The trusted-header resolver is opt-in and should be used only behind an auth proxy or test harness that strips and
   owns those headers. The default resolver still records `operator.unverified` and `local.display_only`.

2. A conflict is a stop sign, not a tie-breaker.

   Peer authority-domain disagreement resolves to `blocked_conflict`. SemOps does not pick a winner by timestamp,
   priority, request order, or local display label.

3. Current state is not full audit history.

   The graph contract remains a fusion-owned current-state `association_review` entity. It records the effective
   arbitrated state and enough metadata for readback, but it is not a durable per-authority append log.

4. Display-only review cannot overwrite authenticated authority state.

   Once an authenticated authority-domain vote exists for an association, later local display-only reviews are
   rejected by the local arbitration store.

## Accepted Evidence

- `internal/api/cop` resolves trusted-header identity only when explicitly configured and validates operator ID,
  authenticated marker, role, scope, and authority domain.
- `MemoryAssociationReviewStore` preserves authority-domain votes, derives consensus, and emits blocked conflicts
  when authenticated peer domains disagree.
- `GraphAssociationReviewStore` previews and writes the effective arbitrated review state before updating local
  overlay state.
- `internal/projectors/fusion` projects authenticated, authority-domain, conflict-policy, and conflict-state
  predicates into the fusion-owned `association_review` graph contract.
- `internal/api/cop.GraphProvider` hydrates the new review fields back into COP snapshot association readback.
- `cmd/semops` enables trusted-header identity only through `SEMOPS_COP_OPERATOR_IDENTITY_MODE=trusted_headers`.

## Follow-Ups

- Add durable per-authority review history before relying on historical dispute analysis.
- Keep command execution, identity merge/split, upstream CS API status, and compliance workflows blocked until each
  consumes `blocked_conflict` as a hard stop and has its own end-to-end authority tests.
- Add UI language for authenticated consensus and blocked conflicts before exposing trusted-header mode in demos.
