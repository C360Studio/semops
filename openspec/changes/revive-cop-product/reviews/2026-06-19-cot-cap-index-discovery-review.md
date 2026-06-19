# CoT/CAP Index Discovery Review

Date: 2026-06-19

## Decision

Accept index-backed SemStreams prefix discovery as the normal snapshot discovery path for TAK/CoT and CAP state.

CoT UID and CAP alert-ID seed lists remain useful as compatibility fallback and targeted tests, but they should not be
required in the hosted COP demo path when graph discovery is enabled.

## Objections Raised

- Partial prefix discovery can create a false green if MAVLink discovery works while TAK/CoT or CAP state is missing.
- Optional CoT/CAP seed IDs could make local failures look like empty state instead of a misconfigured graph.
- Prefix discovery without diagnostics can hide cardinality pressure until a large scenario is already running.
- CAP remains append-evidence only, so discovering CAP hazard entities must not imply authoritative hazard ownership.

## Evidence Checked

- `internal/api/cop` now records discovery success per feed family and falls back only for families that produced no
  prefix-discovered state.
- Runtime defaults keep graph discovery enabled and no longer require CoT UID or CAP alert-ID seed lists.
- `compose.cop.yml` no longer configures CoT seed UIDs for the SemOps API service.
- Unit coverage proves MAVLink-only prefix discovery does not suppress CoT/CAP seed fallback, while default config uses
  discovery-only CoT/CAP snapshot state.

## Accepted Risks

- Discovery success is tracked at the feed-family level, not per entity kind; a discovered CoT task can suppress
  CoT seed fallback for a missing CoT advisory.
- Prefix discovery still needs source/type counters and pagination guidance before large mixed-feed demos.
- Seed fallback remains important for targeted tests and degraded deployments, so it should be kept deliberately
  visible rather than deleted.

## Follow-Up Tasks

- Closed by `2026-06-19-source-cardinality-diagnostics-review.md`: add source/type cardinality diagnostics for
  prefix-discovered snapshot state.
- Revisit family-level versus kind-level fallback after the first large scenario replay.
- Keep CAP ownership language append-evidence-only until a separate fusion or control projection owns authoritative
  hazard lifecycle state.
