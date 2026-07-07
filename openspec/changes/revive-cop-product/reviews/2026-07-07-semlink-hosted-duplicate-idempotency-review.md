# SemLink Hosted Duplicate Idempotency Review

Scope: duplicate SemLink ArduPilot readback requests through the hosted COP API route.

Decision: accept hosted duplicate-collapse evidence for the MVP readback route.

The hosted `/api/cop/semlink/ardupilot/readback` composition now has end-to-end test evidence that a repeated trusted
SemLink request with the same `idempotency_key` is rejected as duplicate admission before a second command-intent graph
write. The response preserves the no-native/no-companion-transmit posture and reports the original native command ID.

Boundary choices:

- This proves duplicate collapse inside the configured hosted process and guarded projector.
- It does not claim durable idempotency across SemOps restarts or multiple API replicas.
- Duplicate handling remains command-intent admission only; it does not add native MAVLink or companion transmit.

Evidence:

- `cmd/semops/main_test.go` posts two trusted SemLink readback requests with the same idempotency key.
- The second response reports duplicate admission, zero mutations, and no transmit authority.
- The graph requester records exactly one create mutation and no update mutation for the duplicate request.

Residual risk:

- Production multi-replica or restart-safe SemLink ingress still needs a durable idempotency strategy before this route
  becomes a distributed write-side surface.
