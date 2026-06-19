# SAPIENT Feed Evidence

Status: evidence gap before phase commitment.

## Decision

SAPIENT should not enter implementation until authoritative artifacts are found. Do not build around guessed schemas,
rumored compliance suites, or inferred protobuf shapes.

## Local Evidence

- No SemOps SAPIENT adapter exists in the current checkout.
- No authoritative SAPIENT ICD, protobuf schema, validator, sample message, or compliance harness was found locally.
- The feed ladder assigns detections/tracks to `signal`, tasking/collection state to `control`, and native decode
  traces to `trace`.

## External Evidence

- Public web searches in this pass did not verify a SAPIENT compliance suite, authoritative ICD, protobuf
  definition, validator, or fixture corpus.

## Gates

### Artifact Discovery Gate

Target outcome:

- Locate authoritative SAPIENT artifacts before code starts.

Acceptance:

- ICD or schema source is identified.
- Message samples or fixtures are available.
- License and redistribution constraints are understood.
- If a compliance suite exists, its install/run path is documented.

### Parser Gate

Target command after artifacts exist:

```bash
go test ./internal/sapient
```

Acceptance:

- Valid authoritative fixtures parse.
- Malformed messages fail before graph writes.
- Unknown or future fields are handled according to the authoritative compatibility rules.

### Projection Gate

Target command after SemOps graph contracts exist:

```bash
go test ./internal/projectors/sapient
```

Acceptance:

- Detections and tracks use `indexing_profile=signal`.
- Sensor tasking, collection plans, and alert state use `indexing_profile=control`.
- Native decode/replay records use `indexing_profile=trace`.
- SAPIENT does not overwrite stricter source facts without an explicit ownership contract.

## Known Gaps

- No verified public compliance suite.
- No verified authoritative schema.
- No local fixtures.
- No licensing review.

## Adversarial Feed-Entry Questions

- Are we using authoritative artifacts, or just schema-shaped guesses?
- Does any compliance claim name the suite and run command?
- Are tasking/control semantics separated from detection state?
- Are malformed messages rejected before graph writes?
- Are licensing constraints compatible with checked-in fixtures?
