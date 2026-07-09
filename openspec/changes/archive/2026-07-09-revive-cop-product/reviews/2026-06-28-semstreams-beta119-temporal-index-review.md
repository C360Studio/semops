# SemStreams Beta 119 Temporal Index Review

Scope: SemOps adoption of `github.com/c360studio/semstreams v1.0.0-beta.119` after SemStreams issue `#370` landed.

## Verdict

Accept the SemStreams bump.

SemOps track projectors already emit `time.observation.recorded`, so moving from beta.115 to beta.119 lets
SemStreams temporal indexing use source observation time instead of graph-ingest write time for those track entities.

## Verified Upstream Behavior

The downloaded beta.119 module shows the temporal index timestamp precedence in
`processor/graph-index-temporal/component.go`:

- `time.observation.recorded` is the primary event-time key.
- `EntityState.UpdatedAt` remains the processing-time fallback.
- `processor/graph-index-temporal/metrics.go` labels indexed entities as `observed` or `write_fallback`, making
  producer adoption visible.
- SemStreams docs `docs/concepts/30-spatial-temporal-queries.md` and `docs/advanced/05-index-reference.md` describe
  the same precedence.

## SemOps Impact

- `go.mod` now pins `github.com/c360studio/semstreams v1.0.0-beta.119`.
- No SemOps API or graph mutation shape changes were needed.
- Non-track geometry remains out of this slice; normalize those only when concrete KLV, weather, task, or advisory
  query pressure requires it.
