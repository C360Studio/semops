# SemLink Readback Auth Gate Review

Scope: caller authorization for the hosted SemLink ArduPilot readback route.

Decision: accept a narrow trusted-header gate for SemLink readback intent.

The hosted route now requires `SEMOPS_COP_OPERATOR_IDENTITY_MODE=trusted_headers` before
`SEMOPS_COP_SEMLINK_READBACK_ENABLED=true` can expose `/api/cop/semlink/ardupilot/readback`. The request must carry
trusted caller headers, including `X-SemOps-Operator-Authenticated: true`, an operator ID, an authority domain, and
`X-SemOps-Authority-Scope: semlink.readback.intent`.

Boundary choices:

- The SemLink scope is separate from `association.review`; association-review authority does not imply command-intent
  ingress authority.
- Missing trusted headers fail before request decoding, ingress admission, target lookup, or graph writes.
- Successful responses include authenticated caller posture for audit readback.
- The authorization gate still only permits readback-intent persistence; it does not enable native MAVLink or
  companion transmit.

Evidence:

- `internal/api/cop/semlink_readback_auth.go` adds the SemLink readback trusted-header authorizer.
- `internal/api/cop/semlink_readback.go` applies the optional authorizer before admission and annotates accepted or
  rejected admission responses with authenticated caller posture.
- `cmd/semops/main.go` installs the authorizer only for the hosted SemLink readback route and rejects enabled config
  unless COP operator identity mode is `trusted_headers`.
- Handler, app config, and hosted wiring tests cover missing headers, caller posture, and trusted-mode validation.

Residual risk:

- This trusts an upstream authentication boundary; SemOps still does not implement request signing, mTLS identity, or
  BlueOS extension trust establishment in-process.
- A live SemLink node, Navigator, and ArduPilot path remain outside this slice.
