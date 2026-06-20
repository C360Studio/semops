package cop

import (
	"errors"
	"reflect"
	"testing"

	"github.com/c360studio/semstreams/pkg/ownership"
	"github.com/c360studio/semstreams/pkg/projection"
)

func TestFirstCanonicalEntitySet(t *testing.T) {
	want := []string{
		"track",
		"asset",
		"hazard_area",
		"sensor_footprint",
		"alert",
		"task",
		"advisory",
	}
	if len(FirstCanonicalEntitySet) != len(want) {
		t.Fatalf("entity count = %d, want %d", len(FirstCanonicalEntitySet), len(want))
	}
	for i := range want {
		if FirstCanonicalEntitySet[i] != want[i] {
			t.Fatalf("entity[%d] = %q, want %q", i, FirstCanonicalEntitySet[i], want[i])
		}
	}
}

func TestFirstPhaseContractsValidateAndDerive(t *testing.T) {
	grouped := make(map[string][]projection.Contract)
	for _, owned := range FirstPhaseOwnedContracts() {
		if err := owned.Contract.Validate(); err != nil {
			t.Fatalf("%s should validate: %v", owned.Contract.Name, err)
		}
		grouped[owned.Owner] = append(grouped[owned.Owner], owned.Contract)
	}

	for owner, contracts := range grouped {
		registration, err := projection.Derive(owner, contracts...)
		if err != nil {
			t.Fatalf("%s should derive grouped ownership: %v", owner, err)
		}
		if registration.Owner != owner {
			t.Fatalf("registration owner = %q, want %q", registration.Owner, owner)
		}
		if len(registration.Claims) == 0 && len(registration.ForeignEdges) == 0 {
			t.Fatalf("%s derived no claims", owner)
		}
	}
}

func TestStrictTolerantAndFusionOwnershipModes(t *testing.T) {
	assets := SourceAssetContract()
	if got := assets.Groups[0].Mode; got != ownership.ModeReplaceOwned {
		t.Fatalf("asset mode = %q, want replace-owned", got)
	}
	if assets.IndexingProfile != "control" {
		t.Fatalf("asset indexing profile = %q, want control", assets.IndexingProfile)
	}

	mavlink := MAVLinkTrackContract()
	if got := mavlink.Groups[0].Mode; got != ownership.ModeReplaceOwned {
		t.Fatalf("MAVLink mode = %q, want replace-owned", got)
	}
	if mavlink.IndexingProfile != "signal" {
		t.Fatalf("MAVLink indexing profile = %q, want signal", mavlink.IndexingProfile)
	}

	tak := TAKTrackContract()
	if got := tak.Groups[0].Mode; got != ownership.ModeReplaceOwned {
		t.Fatalf("TAK mode = %q, want replace-owned", got)
	}
	if tak.IndexingProfile != "signal" {
		t.Fatalf("TAK indexing profile = %q, want signal", tak.IndexingProfile)
	}
	if tak.EntityPattern == mavlink.EntityPattern {
		t.Fatalf("TAK and MAVLink strict track contracts must be source-partitioned")
	}

	adsb := ADSBTrackContract()
	if err := adsb.Validate(); err != nil {
		t.Fatalf("ADS-B contract should validate: %v", err)
	}
	if got := adsb.Groups[0].Mode; got != ownership.ModeReplaceOwned {
		t.Fatalf("ADS-B mode = %q, want replace-owned", got)
	}
	if adsb.IndexingProfile != "signal" {
		t.Fatalf("ADS-B indexing profile = %q, want signal", adsb.IndexingProfile)
	}
	if adsb.EntityPattern == mavlink.EntityPattern || adsb.EntityPattern == tak.EntityPattern {
		t.Fatalf("ADS-B strict track contract must be source-partitioned")
	}
	if len(adsb.ForeignEdges) != 0 {
		t.Fatalf("ADS-B feed must not claim association foreign edges: %+v", adsb.ForeignEdges)
	}
	adsbRegistration, err := projection.Derive(OwnerADSB, adsb)
	if err != nil {
		t.Fatalf("derive ADS-B ownership: %v", err)
	}
	if len(adsbRegistration.ForeignEdges) != 0 {
		t.Fatalf("ADS-B derived foreign edges = %+v, want none", adsbRegistration.ForeignEdges)
	}

	takTask := TAKTaskContract()
	if got := takTask.Groups[0].Mode; got != ownership.ModeReplaceOwned {
		t.Fatalf("TAK task mode = %q, want replace-owned", got)
	}
	if takTask.IndexingProfile != "control" {
		t.Fatalf("TAK task indexing profile = %q, want control", takTask.IndexingProfile)
	}

	takAdvisory := TAKAdvisoryContract()
	if got := takAdvisory.Groups[0].Mode; got != ownership.ModeReplaceOwned {
		t.Fatalf("TAK advisory mode = %q, want replace-owned", got)
	}
	if takAdvisory.IndexingProfile != "content" {
		t.Fatalf("TAK advisory indexing profile = %q, want content", takAdvisory.IndexingProfile)
	}

	cap := CAPHazardEvidenceContract()
	for _, group := range cap.Groups {
		if group.Mode != ownership.ModeAppendEvidence {
			t.Fatalf("CAP mode = %q, want append-evidence", group.Mode)
		}
	}
	if cap.IndexingProfile != "content" {
		t.Fatalf("CAP indexing profile = %q, want content", cap.IndexingProfile)
	}

	fusion := FusionAlertContract()
	if got := fusion.Groups[0].Mode; got != ownership.ModeReplaceOwned {
		t.Fatalf("fusion mode = %q, want replace-owned", got)
	}
	if fusion.IndexingProfile != "control" {
		t.Fatalf("fusion indexing profile = %q, want control", fusion.IndexingProfile)
	}
}

func TestTAKOwnerBindsTrackControlAndContentContracts(t *testing.T) {
	owners := make([]string, 0)
	for _, owned := range FirstPhaseOwnedContracts() {
		if owned.Owner == OwnerTAK {
			owners = append(owners, owned.Contract.MessageType)
		}
	}

	want := []string{
		TAKTrackContract().MessageType,
		TAKTaskContract().MessageType,
		TAKAdvisoryContract().MessageType,
	}
	if !reflect.DeepEqual(owners, want) {
		t.Fatalf("TAK contracts = %#v, want %#v", owners, want)
	}

	registration, err := projection.Derive(
		OwnerTAK,
		TAKTrackContract(),
		TAKTaskContract(),
		TAKAdvisoryContract(),
	)
	if err != nil {
		t.Fatalf("derive TAK grouped contracts: %v", err)
	}
	if len(registration.ForeignEdges) != 1 {
		t.Fatalf("TAK foreign edges = %d, want strict track source only", len(registration.ForeignEdges))
	}
}

func TestForeignEdgesDeclareADR056BornFirstShape(t *testing.T) {
	for _, owned := range []OwnedContract{
		{Owner: OwnerMAVLink, Contract: MAVLinkTrackContract()},
		{Owner: OwnerTAK, Contract: TAKTrackContract()},
	} {
		contract := owned.Contract
		registration, err := projection.Derive(owned.Owner, contract)
		if err != nil {
			t.Fatalf("%s should derive: %v", contract.Name, err)
		}
		if len(registration.ForeignEdges) != 1 {
			t.Fatalf("%s foreign edges = %d, want 1", contract.Name, len(registration.ForeignEdges))
		}

		edge := registration.ForeignEdges[0]
		if edge.Predicate != TrackSource {
			t.Fatalf("%s foreign edge predicate = %q, want %q", contract.Name, edge.Predicate, TrackSource)
		}
		if edge.Producer != contract.MessageType {
			t.Fatalf("%s producer = %q, want %q", contract.Name, edge.Producer, contract.MessageType)
		}
		if edge.TargetPattern != EntityPattern(EntityAsset) {
			t.Fatalf("%s target pattern = %q, want %q", contract.Name, edge.TargetPattern, EntityPattern(EntityAsset))
		}
		if edge.Mode != ownership.EdgeStrict {
			t.Fatalf("%s edge mode = %q, want strict born-first edge", contract.Name, edge.Mode)
		}
	}
}

func TestOverlappingReplaceOwnedPredicatesAreRejected(t *testing.T) {
	dupe := MAVLinkTrackContract()
	dupe.Name = "semops.cop.track.conflicting-current-state"
	dupe.Groups = []projection.PredicateGroup{{
		Mode:       ownership.ModeReplaceOwned,
		Predicates: []string{TrackPosition},
	}}

	_, err := projection.Derive(OwnerMAVLink, MAVLinkTrackContract(), dupe)
	if !errors.Is(err, ownership.ErrOwnershipOverlap) {
		t.Fatalf("overlap error = %v, want ErrOwnershipOverlap", err)
	}
}

func TestTolerantCAPContractDoesNotClaimAuthoritativeHazardState(t *testing.T) {
	for _, group := range CAPHazardEvidenceContract().Groups {
		if group.Mode != ownership.ModeAppendEvidence {
			t.Fatalf("CAP group mode = %q, want append-evidence", group.Mode)
		}
		for _, predicate := range group.Predicates {
			switch predicate {
			case HazardGeometry, HazardSeverity, HazardStatus:
				t.Fatalf("CAP evidence contract must not own authoritative predicate %q", predicate)
			}
		}
	}
}
