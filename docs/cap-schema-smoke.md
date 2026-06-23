# CAP Schema And NWS Sample Smoke

Status: opt-in developer smoke, not default CI and not CAP conformance.

SemOps can validate local CAP XML samples against a local CAP 1.2 XSD with `xmllint` and then parse the same samples
through the SemOps CAP adapter. This is useful for captured NWS/IPAWS/vendor samples, but it does not vendor official
schemas or public alert captures.

## Inputs

- `SEMOPS_CAP_XSD_PATH`: local path to the CAP 1.2 XSD.
- `SEMOPS_CAP_SCHEMA_SAMPLE_PATHS`: path-list of local CAP XML files, directories, or globs.
- `SEMOPS_CAP_SCHEMA_REPLAY_PATH`: optional SemOps CAP replay JSONL file produced by `SEMOPS_CAP_REPLAY_PATH`.

Local schema and sample directories are ignored by git under `fixtures/cap/schema/`, `fixtures/cap/nws-samples/`, and
`fixtures/cap/replay/`.

## Run

Capture a local provider sample only when you are intentionally making a live network call:

```bash
SEMOPS_CAP_CAPTURE_URL="https://api.weather.gov/alerts/active?area=TX" \
SEMOPS_CAP_CAPTURE_USER_AGENT="semops-demo contact@example.invalid" \
bash scripts/cap-capture-nws-sample.sh
```

The script requests `application/cap+xml`, writes the XML to `fixtures/cap/nws-samples/`, and writes a sibling
metadata file with source URL, user-agent, capture time, and SHA-256 when `shasum` is available.

```bash
SEMOPS_CAP_XSD_PATH="fixtures/cap/schema/CAP-v1.2.xsd" \
SEMOPS_CAP_SCHEMA_SAMPLE_PATHS="fixtures/cap/nws-samples" \
go test ./pkg/adapters/cap -run TestCAPSchemaSmokeWithLocalSamples -count=1 -v
```

Replay JSONL captured from the hosted poller can be checked too:

```bash
SEMOPS_CAP_XSD_PATH="fixtures/cap/schema/CAP-v1.2.xsd" \
SEMOPS_CAP_SCHEMA_REPLAY_PATH="fixtures/cap/replay/nws-alerts.jsonl" \
go test ./pkg/adapters/cap -run TestCAPSchemaSmokeWithLocalSamples -count=1 -v
```

## Acceptance

The smoke passes only when:

- `xmllint` is available on `PATH`.
- Every supplied sample validates against the supplied XSD.
- Every supplied sample also parses through `pkg/adapters/cap.Parse`.

Passing this smoke means "local samples satisfy the supplied schema and the SemOps parser." It is not a CAP consumer
conformance claim, NWS service integration claim, IPAWS claim, or proof that default CI has official schemas or public
alert captures.

## Recorded Runs

2026-06-23 local NWS active-alert sample smoke:

- Schema URL: <https://docs.oasis-open.org/emergency/cap/v1.2/CAP-v1.2.xsd>
- Schema SHA-256: `b7798ef25868b068c97b268bda02d067c7d4ba9373adc5638bf37105804ee723`
- Sample URL:
  <https://api.weather.gov/alerts/urn:oid:2.49.0.1.840.0.d3b1c6276b01a6ad0f2ba342dcd0574ff82d2939.001.1>
- User-Agent: `SemOps-COP-demo (https://github.com/C360Studio/semops)`
- Captured at: `2026-06-23T13:28:40Z`
- Sample event: `Flash Flood Warning`
- Sample status/msgType: `Actual` / `Alert`
- Sample sent: `2026-06-23T08:25:00-05:00`
- Sample SHA-256: `c91278df362ac125b08c0152d7fbea1eade60ea84e6dbd21c941185882cd0a97`
- Local files: ignored under `fixtures/cap/schema/` and `fixtures/cap/nws-samples/`

Command:

```bash
SEMOPS_CAP_XSD_PATH="fixtures/cap/schema/CAP-v1.2.xsd" \
SEMOPS_CAP_SCHEMA_SAMPLE_PATHS="fixtures/cap/nws-samples/nws-active-flash-flood-warning-20260623.xml" \
go test ./pkg/adapters/cap -run TestCAPSchemaSmokeWithLocalSamples -count=1 -v
```

Result: passed. This is one active NWS alert sample, not update/cancel/expired lifecycle coverage.
