# SemOps OpenSpec

SemOps uses OpenSpec change sets as the planning contract for large product moves. The documents here are
human-readable, but they are written so implementation tickets, tests, demos, and upstream SemStreams asks can
trace back to explicit requirements.

## Layout

- `project.md`: standing project context and conventions.
- `changes/<change-id>/proposal.md`: why the change exists, what changes, and expected impact.
- `changes/<change-id>/design.md`: design decisions, trade-offs, rollout, and open questions.
- `changes/<change-id>/tasks.md`: implementation checklist in dependency order.
- `changes/<change-id>/specs/*/spec.md`: capability requirements and scenarios.

When a change is accepted and implemented, its spec deltas can be promoted into long-lived baseline specs under
`openspec/specs`.
