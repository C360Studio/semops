# Prefix Discovery Readback Review

Date: 2026-06-19

## Decision

Accept SemStreams `graph.query.prefix` as the preferred COP snapshot readback path for source-partitioned MAVLink,
TAK/CoT, and CAP entities.

Keep configured seed IDs as compatibility fallback while the stack hardens, but do not treat seed IDs as the
product-grade discovery model.

## Objections Raised

- Prefix discovery can hide partial failures if one source prefix works and another does not.
- A broad prefix scan can become a cardinality footgun if SemOps points it at too wide a scope.
- The current prefix response is an untyped `{"entities":[...]}` envelope, not a strongly named SemStreams client API.
- Discovery improves readback, but it does not replace feed-specific compliance, lifecycle, or replay evidence.
- Future graph visualization should reuse SemConnect/SemLink graph-lens patterns before SemOps creates a new graph UI.

## Evidence Checked

- SemStreams graph-query registers public `graph.query.prefix` and routes it to graph-ingest prefix lookup.
- SemStreams graph-ingest uses server-side prefix filtering and returns full `EntityState` values in an entities
  envelope.
- SemOps unit coverage proves `GET /api/cop/snapshot` can hydrate MAVLink, TAK/CoT, and CAP state from prefix results
  without using seed entity IDs.
- Seeded `graph.query.entity` reads remain available when prefix discovery is disabled, unavailable, or empty.

## Accepted Risks

- The first implementation queries a fixed set of COP source/type prefixes per configured org/platform scope.
- Prefix discovery limit defaults to a bounded value; future pagination or streaming may be needed for large demos.
- SemOps still needs live stack evidence after the next smoke run to prove the deployed SemStreams tag behaves exactly
  like the inspected local source.

## Follow-Up Tasks

- SemStreams issue #302 tracks typed prefix-query request/response contracts, a client helper, and pagination
  guidance.
- Add source/type counters to COP diagnostics once the API has a deliberate operator or technical evidence panel.
- Revisit discovery scopes before multi-org or multi-platform federation enters the demo.
