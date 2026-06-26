# SemStreams Beta.115 Graph Error Contract Review

Scope: SemOps adoption of SemStreams `v1.0.0-beta.115` ADR-060 graph mutation error returns.

Decision: accept the beta.115 pin and remove legacy graph mutation failure response handling from SemOps feed writers.

## Findings

1. Legacy text-body conflict parsing is a product footgun.
   SemOps should not accept `error: entity already exists` strings or retired `MutationResponse` failure fields as a
   valid graph-write contract. Doing so would let new feed components pass tests while bypassing the SemStreams
   ADR-060 error boundary.

2. Writer reconciliation should stay local and typed.
   Feed graph writers may translate SemStreams `*errs.ClassifiedError` values into local `MutationFailureError`
   values so adapters and components keep a stable package-local reconciliation contract. The source of truth remains
   SemStreams classified Go errors, including `graph.ErrorCodeEntityExists` and `graph.ErrorCodeOwnerLeaseStale`.

3. Success bodies are now success-only.
   `graph.MutationResponse` must be decoded only to confirm a valid success response shape. It no longer carries
   `Success`, `Error`, or `ErrorCode`; degraded success is committed and uses `DegradedReason` if present.

## Required Evidence

- `internal/graphrequest.NATSRequester` uses `RequestWithRetryClassified`.
- MAVLink, CoT, CAP, ADS-B, KLV, weather, SAPIENT, and fusion writers convert classified request errors into local
  typed mutation failures.
- MAVLink restart reconciliation rejects conflicts without an entity ID instead of falling back to plan text.
- Writer tests construct classified errors for conflict paths and no longer fabricate failure response bodies.
- `go test ./...` passes on the beta.115 pin.
- `bash scripts/cop-stack-smoke.sh` passes on the beta.115 pin: scenario runner reached
  `state=succeeded completed=10 failed=0`, hosted COP snapshot and component metrics smokes passed, direct
  MAVLink/CoT/CAP born-first graph smokes passed, and SAPIENT preflight passed.

## Follow-Up

- Keep the restart/read-back durability task open; typed errors solve the contract ambiguity, not durable checkpointing.
- If SemStreams changes classified detail keys beyond `entity`, SemOps should file an upstream issue rather than adding
  parser fallbacks.
- Continue watching the documented CAP append-evidence warning; it is expected for the current no-write-fence CAP
  posture, but should become a SemStreams issue if future append-only bind behavior stops minting usable tokens.
