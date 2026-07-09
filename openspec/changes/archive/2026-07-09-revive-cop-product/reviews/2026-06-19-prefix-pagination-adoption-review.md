# Prefix Pagination Adoption Review

Date: 2026-06-19

## Decision

Adopt SemStreams typed prefix-query pagination for graph-backed COP discovery.

SemOps should page `graph.query.prefix` with `graph.PrefixQueryRequest`, follow opaque `NextCursor` values verbatim,
and stop only when a prefix is exhausted or the configured SemOps discovery cap is reached.

## Objections Raised

- Cursor pagination can hide an infinite-loop bug if SemStreams returns an empty page with a continuation cursor.
- Treating `count == limit` as pressure creates false alerts when the prefix is exactly exhausted at the configured
  limit.
- A broad discovery scope can still be expensive because SemStreams currently scans matching keys before slicing a
  page.
- Asking SemStreams for total-count metadata now could be premature if cursor paging already gives enough
  source-health evidence.

## Evidence Checked

- SemStreams `v1.0.0-beta.113` exposes `graph.PrefixQueryRequest`, `graph.PrefixQueryResponse`, `NextCursor`, and
  `MaxPrefixQueryLimit`.
- `internal/api/cop` now marshals the typed request envelope and decodes the typed response envelope while retaining
  compatibility with the older `data.entities` response shape.
- COP snapshot discovery follows continuation cursors until exhausted or capped, clamps page size to the SemStreams
  max prefix limit, and treats empty continuation pages or repeated cursors as prefix-read errors.
- Unit coverage proves SemOps requests a second page with the returned cursor and does not raise `at_limit` when the
  prefix exhausts before the configured cap.
- Unit coverage proves SemOps raises `at_limit` only when it stops at the configured cap while SemStreams still
  reports continuation state.

## Accepted Risks

- The snapshot diagnostic still reports returned count and configured cap, not a true total count.
- Prefix paging is sequential per source/type prefix; high-cardinality demos may need later concurrency or streaming
  review after real evidence.
- The UI copy still uses compact source/type chips, so deeper index-pressure explanation belongs in a future technical
  evidence lens rather than the primary operator map.

## Follow-Up Tasks

- Comment back on SemStreams issue #302 after the SemOps PR lands with the local adoption evidence.
- Revisit total-count metadata only if mixed-feed scenario runs prove returned count plus truncation pressure is not
  enough for source-health decisions.
- Keep discovery scopes narrow before multi-org or multi-platform federation enters the demo.
