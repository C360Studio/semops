# Alert Target Geometry Review

Date: 2026-06-22
Scope: COP UI alert selection and map geometry behavior.

## Decision

Accept alert-to-target map highlighting for the first alert geometry slice.

Alerts remain control/status view-model objects. When an alert's `entity_id` references a rendered spatial entity,
the UI keeps the alert selected in the inspector while passing the referenced track, asset, task, advisory, hazard, or
sensor footprint as the effective map selection. Source-health alerts that reference feed IDs do not project onto the
map.

This closes the Phase 1 "alert geometry" UX need without inventing a separate alert geometry entity or ownership
contract.

## Adversarial Notes

### Medium: Alert geometry must not imply alert-owned spatial truth

The highlighted geometry belongs to the target entity, not the alert. For example, a MAVLink freshness alert can
highlight the MAVLink track, and a future CAP/fusion alert can highlight a hazard area, but the alert does not own the
track position or hazard polygon.

Resolution: keep alert map behavior as a target-link projection until a dedicated fusion-alert geometry contract
exists.

### Medium: Source-health alerts are deliberately non-spatial

Prefix truncation and discovery-error alerts reference source feeds such as `feed.mavlink`. Highlighting an arbitrary
map object for those technical alerts would be misleading.

Resolution: unresolved alert targets produce no map highlight.

## Verification

- `npm --prefix ui run test`
- `npm --prefix ui run check`
- `npm --prefix ui run test:e2e`

## Follow-Ups

- Revisit alert-owned geometry only after fusion alerts have graph contracts, provenance, and operator semantics.
- Keep source-health alerts visually distinct from tactical/fusion alerts if alert volume grows.
