# SemLink Mesh COP Demo

This demo shows SemOps acting as COP/GCS glass for a SemLink-produced
one-vehicle-per-companion mesh artifact.

## Claim

Allowed:

- SemLink produced a deterministic `simple-mesh-companion-demo` artifact.
- The artifact contains three vehicle-local companion nodes.
- Each companion node reports one simulated MAVLink vehicle.
- Selected summaries caught up across the SemLink mesh.
- Raw MAVLink frames remain local-only by default.
- SemOps renders the artifact as curated companion-fleet COP evidence.

Not claimed by this demo:

- live BlueOS, Navigator, radio mesh, hardware, or SITL fidelity;
- native MAVLink transmit, companion hardware transmit, mission upload, mode
  change, arm/disarm, or offboard control authority;
- CS API or SemConnect in the SemLink hot path;
- one SemLink runtime multiplexing multiple vehicles.

## Generate Artifact

From the SemLink checkout:

```sh
SEMOPS_FIXTURE_DIR=/Users/coby/Code/c360/semops/testdata/contracts/semlink-companion-demo-v0
ARTIFACT="$SEMOPS_FIXTURE_DIR/generated-one-to-one-mesh.artifact.json"
SEMLINK_DEMO_REPORT=/tmp/semlink-one-to-one-mesh.report.json \
SEMLINK_DEMO_ARTIFACT="$ARTIFACT" \
./scripts/demo-mesh-companions.sh
```

The committed fixture currently records SemLink commit
`11f7e6dafea06898f1262877b7d0300e311237f1` and generator command
`semlink-demo -mode mesh -nodes 3 -vehicle-profile ardurover`.

## SemOps Smoke

From the SemOps checkout:

```sh
go test ./internal/adapters/semlinkdemo ./internal/api/cop ./internal/smoke/cop -count=1
```

For an externally generated artifact path, run:

```sh
SEMOPS_COP_SMOKE_SEMLINK_ARTIFACT_PATH=/path/to/semlink-artifact.json \
go test ./internal/smoke/cop -run TestSemLinkGeneratedArtifactSmoke -count=1
```

The smoke asserts that the COP snapshot exposes one SemLink companion fleet,
three active companion nodes, one vehicle per node, deterministic source
fidelity, raw MAVLink exclusion, and no native or companion hardware transmit
authority.
