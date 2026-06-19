# Source Cardinality Diagnostics Review

Date: 2026-06-19

## Decision

Accept source/type prefix-discovery diagnostics in the COP snapshot contract and compact source-card UI readout.

Do not promote this into a graph console, topology UI, or orchestration surface. The first value is early evidence of
index pressure and partial prefix-read failure while SemOps adds feeds one at a time.

## Objections Raised

- Showing prefix counts could distract operators from tactical state if the UI treats diagnostics as primary content.
- Count-at-limit is an imperfect pressure signal because SemStreams prefix query does not yet expose total count or
  pagination metadata.
- Prefix-read errors should not make a partially useful snapshot vanish when other sources are readable.
- Diagnostics could leak too much graph substrate detail if raw triples, subjects, or native payloads appear in the
  operator view.

## Evidence Checked

- `internal/api/cop` records one diagnostic per configured source/type prefix with org, platform, source, family,
  entity type, prefix, returned count, query limit, at-limit pressure, and prefix-read error text.
- Unit coverage proves normal prefix discovery emits seven diagnostics for the first MAVLink/TAK/CAP source set.
- Unit coverage proves limit pressure and a partial TAK prefix-read error are surfaced while a useful MAVLink snapshot
  still returns.
- The Svelte source cards show only compact source/type count chips and at-limit emphasis; raw graph triples stay
  behind the API.

## Accepted Risks

- `at_limit` means the returned count reached the configured prefix query limit, not that SemStreams knows the true
  total cardinality.
- Errors are exposed as technical evidence text in the snapshot; future operator-facing wording may need a cleaner
  source-health alert abstraction.
- Source-card chips are a temporary diagnostic surface until the product has a deliberate technical evidence panel.

## Follow-Up Tasks

- Ask SemStreams for typed pagination or total-count metadata only after mixed-feed demos prove the need.
- Convert repeated at-limit pressure into source-health alerting before high-cardinality feeds such as ADS-B or KLV
  enter the main demo.
- Keep raw graph inspection behind a deliberate diagnostic lens, ideally informed by SemConnect/SemLink graph-view
  prior art.
