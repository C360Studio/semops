# Command Intent Graph Writer Review

Scope: SemStreams graph writer for governed command-intent projection plans.

Verdict: accept.

The slice adds the missing command-intent apply path under `internal/projectors/command`. SemLink and CS API ingress
already produce guarded command plans, but a hosted route would have had to choose between dry-run responses and ad hoc
mutation code. The new writer reuses the existing SemStreams `create_with_triples` / `update_with_triples` request
shape and the repository's typed classified-error handling pattern.

Boundary choices:

- The writer applies only command projector `Plan` mutations; it does not create an HTTP route, transmitter, retry loop,
  CS API status publisher, or companion transmit path.
- Create and update requests keep the `semops.command.intent` owner token, trace id, command indexing profile, and
  existing command task triples from the projector.
- When constructed with the originating projector, the writer marks successfully-applied create plans as born so future
  projections become updates instead of duplicate creates.
- Failed applies do not mark born state, which keeps retry behavior honest.

Evidence:

- `go test ./internal/projectors/command`

Residual risk:

- This is a local writer proof. The hosted API route still needs explicit target resolution, idempotency persistence,
  graph requester wiring, request authentication posture, and response-shape review before SemLink or CS API write-side
  ingress is exposed through the SemOps API.
