# Live Snapshot Provider Adversarial Review

Date: 2026-06-19

## Decision

Accept the first graph-backed COP snapshot provider as a Phase 1 bridge from governed MAVLink graph state into the
browser contract. Do not treat it as the final query/index strategy.

## Evidence Checked

- `internal/api/cop/graph_provider.go` queries SemStreams `graph.query.entity` for configured MAVLink source asset and
  track IDs, prefers classified query responses when available, and maps triples into COP snapshot tracks, assets,
  feed health, freshness, confidence, and provenance.
- `cmd/semops` wires the provider to the connected SemStreams client and retains the fixture provider as a cold-start
  fallback.
- `compose.cop.yml` enables the hosted MAVLink UDP listener and exposes it for local smoke input.
- `internal/smoke/cop/live_snapshot_test.go` sends generated MAVLink over UDP and waits for the Caddy-routed snapshot
  to show graph-backed track state.

## Objections

- The provider watches configured MAVLink system IDs rather than discovering all relevant COP entities. This is
  acceptable for the first demo slice but will not scale to mixed feeds or operator-selected areas of interest.
- Track source asset position is hydrated from the current track because the source asset contract does not own a
  position predicate yet. That is a UI convenience, not a semantic claim about the asset entity.
- The snapshot does not yet expose graph revision, owner-claim status, or indexing profile metadata to the inspector.
- The CAP append-evidence contract currently triggers a SemStreams warning because it is not an enforceable write
  fence. That is consistent with the framework direction, but SemOps should watch the tagged API for zero-token or
  evidence-declaration changes.

## Accepted Risks

- Keep configured MAVLink system IDs for Phase 1 while feed discovery and scenario catalogs are still forming.
- Keep fixture fallback for cold-start and frontend development, but do not count fixture snapshots as graph-compliance
  evidence.
- Use `graph.query.entity` now rather than direct KV reads so SemOps exercises the deployed request/reply surface.

## Verification

- `go test ./internal/api/cop ./internal/app ./internal/smoke/cop -count=1`
- `go test ./... -count=1`
- `bash scripts/cop-stack-smoke.sh`

Post-tag verification after SemStreams published `v1.0.0-beta.112`:

- Removed the local Go module replace and pinned `github.com/c360studio/semstreams v1.0.0-beta.112`.
- `go test ./internal/api/cop ./internal/app ./internal/smoke/cop -count=1`
- `go test ./... -count=1`
- `go build -o /private/tmp/semops-build ./cmd/semops`
- `bash scripts/cop-stack-smoke.sh`

Prefix-discovery contract verification after SemStreams published `v1.0.0-beta.113`:

- Bumped the SemOps module pin to `github.com/c360studio/semstreams v1.0.0-beta.113`.
- Confirmed the sibling SemStreams checkout used by `scripts/cop-stack-smoke.sh` is also at `v1.0.0-beta.113`.
- `go test ./internal/contracts ./internal/api/cop ./internal/smoke/cap ./internal/smoke/cot ./internal/smoke/mavlink`
- `go test ./...`
- `go build -o /tmp/semops ./cmd/semops`
- `bash scripts/cop-stack-smoke.sh`

## Follow-Up Tasks

- Add graph revision/readback metadata to the selected-entity inspector.
- Replace configured MAVLink ID polling with a bounded graph query or scenario catalog once SemStreams/COP discovery
  requirements are clearer.
- Keep the CAP append-evidence-only warning on the SemStreams watch list and record any future OwnerToken or
  evidence-declaration API drift as a SemStreams issue candidate.
