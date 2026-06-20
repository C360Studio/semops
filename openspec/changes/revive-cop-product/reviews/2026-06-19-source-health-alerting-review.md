# Source Health Alerting Review

Date: 2026-06-19

## Decision

Accept warning alerts for prefix discovery errors and source/type discovery truncation pressure in the COP snapshot.

These alerts are source-health evidence. They do not claim authoritative fusion state, graph repair, or total
cardinality.

## Objections Raised

- Truncation pressure is only a risk signal, not proof that useful entities were omitted.
- Prefix-read errors could flood the alert list if every source/type prefix fails at once.
- Source-health alerts could be confused with tactical alerts unless their reason text stays explicit.
- Cursor pagination still does not expose total-count reporting.

## Evidence Checked

- `internal/api/cop` derives deterministic alert IDs from org, platform, source, entity type, and condition.
- Cap-truncated diagnostics create active warning alerts tied to the affected source feed.
- Prefix-read errors create active warning alerts while partial snapshot state still returns.
- Existing alert selection UI can display the source-health alert reason without adding a new panel.
- Closed by `2026-06-19-prefix-pagination-adoption-review.md`: SemOps pages SemStreams typed prefix-query responses
  before raising truncation pressure.

## Accepted Risks

- Alerts are generated from current snapshot diagnostics, not persisted as graph-backed `alert` entities.
- Repeated prefix-read errors may need grouping if many prefixes fail together.
- The alert severity is fixed at `warning` until SemOps has source-health policy thresholds.

## Follow-Up Tasks

- Ask SemStreams for total-count metadata only if mixed-feed demos prove cursor paging plus truncation alerts are
  insufficient.
- Add grouping or suppression if source-health alerts become noisy during high-cardinality scenarios.
- Keep tactical fusion alerts and technical source-health alerts distinguishable in copy and provenance.
