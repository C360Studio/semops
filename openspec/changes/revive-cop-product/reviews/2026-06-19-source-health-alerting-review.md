# Source Health Alerting Review

Date: 2026-06-19

## Decision

Accept warning alerts for prefix discovery errors and at-limit pressure in the COP snapshot.

These alerts are source-health evidence. They do not claim authoritative fusion state, graph repair, or pagination
support.

## Objections Raised

- At-limit pressure is only a truncation risk signal, not proof that useful entities were omitted.
- Prefix-read errors could flood the alert list if every source/type prefix fails at once.
- Source-health alerts could be confused with tactical alerts unless their reason text stays explicit.
- This still does not solve large-demo pagination or total-count reporting.

## Evidence Checked

- `internal/api/cop` derives deterministic alert IDs from org, platform, source, entity type, and condition.
- At-limit diagnostics create active warning alerts tied to the affected source feed.
- Prefix-read errors create active warning alerts while partial snapshot state still returns.
- Existing alert selection UI can display the source-health alert reason without adding a new panel.

## Accepted Risks

- Alerts are generated from current snapshot diagnostics, not persisted as graph-backed `alert` entities.
- Repeated prefix-read errors may need grouping if many prefixes fail together.
- The alert severity is fixed at `warning` until SemOps has source-health policy thresholds.

## Follow-Up Tasks

- Ask SemStreams for pagination or total-count metadata once mixed-feed demos show routine at-limit pressure.
- Add grouping or suppression if source-health alerts become noisy during high-cardinality scenarios.
- Keep tactical fusion alerts and technical source-health alerts distinguishable in copy and provenance.
