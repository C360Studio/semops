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
		"weather_observation",
		"association",
		"association_review",
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

	mavlinkCommandTask := MAVLinkCommandTaskContract()
	if err := mavlinkCommandTask.Validate(); err != nil {
		t.Fatalf("MAVLink command task contract should validate: %v", err)
	}
	if got := mavlinkCommandTask.Groups[0].Mode; got != ownership.ModeReplaceOwned {
		t.Fatalf("MAVLink command task mode = %q, want replace-owned", got)
	}
	if mavlinkCommandTask.IndexingProfile != "control" {
		t.Fatalf("MAVLink command task indexing profile = %q, want control", mavlinkCommandTask.IndexingProfile)
	}
	if mavlinkCommandTask.EntityPattern == mavlink.EntityPattern {
		t.Fatalf("MAVLink command tasks must use a task entity pattern, not track")
	}
	mavlinkRegistration, err := projection.Derive(OwnerMAVLink, mavlink, mavlinkCommandTask)
	if err != nil {
		t.Fatalf("derive MAVLink track + command task ownership: %v", err)
	}
	if len(mavlinkRegistration.ForeignEdges) != 2 {
		t.Fatalf("MAVLink foreign edges = %d, want track source + task target", len(mavlinkRegistration.ForeignEdges))
	}

	commandIntent := CommandIntentContract()
	if err := commandIntent.Validate(); err != nil {
		t.Fatalf("command intent contract should validate: %v", err)
	}
	if got := commandIntent.Groups[0].Mode; got != ownership.ModeReplaceOwned {
		t.Fatalf("command intent mode = %q, want replace-owned", got)
	}
	if commandIntent.IndexingProfile != "control" {
		t.Fatalf("command intent indexing profile = %q, want control", commandIntent.IndexingProfile)
	}
	if commandIntent.EntityPattern == mavlinkCommandTask.EntityPattern {
		t.Fatalf("command intent must be source-partitioned away from MAVLink ACK readback tasks")
	}
	commandRegistration, err := projection.Derive(OwnerCommand, commandIntent)
	if err != nil {
		t.Fatalf("derive command intent ownership: %v", err)
	}
	if len(commandRegistration.ForeignEdges) != 1 {
		t.Fatalf("command intent foreign edges = %d, want task target", len(commandRegistration.ForeignEdges))
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

	sapient := SAPIENTTrackContract()
	if err := sapient.Validate(); err != nil {
		t.Fatalf("SAPIENT track contract should validate: %v", err)
	}
	if got := sapient.Groups[0].Mode; got != ownership.ModeReplaceOwned {
		t.Fatalf("SAPIENT mode = %q, want replace-owned", got)
	}
	if sapient.IndexingProfile != "signal" {
		t.Fatalf("SAPIENT indexing profile = %q, want signal", sapient.IndexingProfile)
	}
	if sapient.EntityPattern == mavlink.EntityPattern || sapient.EntityPattern == tak.EntityPattern || sapient.EntityPattern == adsb.EntityPattern {
		t.Fatalf("SAPIENT strict track contract must be source-partitioned")
	}
	if len(sapient.ForeignEdges) != 0 {
		t.Fatalf("SAPIENT feed must not claim association foreign edges: %+v", sapient.ForeignEdges)
	}
	sapientRegistration, err := projection.Derive(OwnerSAPIENT, sapient)
	if err != nil {
		t.Fatalf("derive SAPIENT ownership: %v", err)
	}
	if len(sapientRegistration.ForeignEdges) != 0 {
		t.Fatalf("SAPIENT derived foreign edges = %+v, want none before fusion association claims", sapientRegistration.ForeignEdges)
	}

	klv := KLVSensorFootprintContract()
	if err := klv.Validate(); err != nil {
		t.Fatalf("KLV sensor-footprint contract should validate: %v", err)
	}
	if got := klv.Groups[0].Mode; got != ownership.ModeReplaceOwned {
		t.Fatalf("KLV sensor-footprint mode = %q, want replace-owned", got)
	}
	if klv.IndexingProfile != "signal" {
		t.Fatalf("KLV sensor-footprint indexing profile = %q, want signal", klv.IndexingProfile)
	}
	if klv.EntityPattern == mavlink.EntityPattern || klv.EntityPattern == tak.EntityPattern || klv.EntityPattern == adsb.EntityPattern || klv.EntityPattern == sapient.EntityPattern {
		t.Fatalf("KLV sensor-footprint contract must be source-partitioned")
	}
	klvRegistration, err := projection.Derive(OwnerKLV, klv)
	if err != nil {
		t.Fatalf("derive KLV ownership: %v", err)
	}
	if len(klvRegistration.ForeignEdges) != 0 {
		t.Fatalf("KLV derived foreign edges = %+v, want none before footprint association claims", klvRegistration.ForeignEdges)
	}

	weather := WeatherObservationContract()
	if err := weather.Validate(); err != nil {
		t.Fatalf("weather observation contract should validate: %v", err)
	}
	if got := weather.Groups[0].Mode; got != ownership.ModeReplaceOwned {
		t.Fatalf("weather observation mode = %q, want replace-owned", got)
	}
	if weather.IndexingProfile != "signal" {
		t.Fatalf("weather observation indexing profile = %q, want signal", weather.IndexingProfile)
	}
	if weather.EntityPattern == mavlink.EntityPattern ||
		weather.EntityPattern == tak.EntityPattern ||
		weather.EntityPattern == adsb.EntityPattern ||
		weather.EntityPattern == sapient.EntityPattern ||
		weather.EntityPattern == klv.EntityPattern {
		t.Fatalf("weather observation contract must be source-partitioned")
	}
	weatherRegistration, err := projection.Derive(OwnerWeather, weather)
	if err != nil {
		t.Fatalf("derive weather ownership: %v", err)
	}
	if len(weatherRegistration.ForeignEdges) != 0 {
		t.Fatalf("weather derived foreign edges = %+v, want none before route/hazard association claims", weatherRegistration.ForeignEdges)
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

	association := FusionTrackAssociationContract()
	if err := association.Validate(); err != nil {
		t.Fatalf("fusion track association contract should validate: %v", err)
	}
	if got := association.Groups[0].Mode; got != ownership.ModeReplaceOwned {
		t.Fatalf("association mode = %q, want replace-owned", got)
	}
	if association.IndexingProfile != "control" {
		t.Fatalf("association indexing profile = %q, want control", association.IndexingProfile)
	}
	if association.EntityPattern == mavlink.EntityPattern || association.EntityPattern == adsb.EntityPattern {
		t.Fatalf("fusion association contract must be source-partitioned away from feed tracks")
	}

	review := FusionAssociationReviewContract()
	if err := review.Validate(); err != nil {
		t.Fatalf("fusion association review contract should validate: %v", err)
	}
	if got := review.Groups[0].Mode; got != ownership.ModeReplaceOwned {
		t.Fatalf("association review mode = %q, want replace-owned", got)
	}
	if review.IndexingProfile != "control" {
		t.Fatalf("association review indexing profile = %q, want control", review.IndexingProfile)
	}
	if review.EntityPattern == association.EntityPattern || review.EntityPattern == mavlink.EntityPattern {
		t.Fatalf("association review contract must be separated from associations and feed tracks")
	}
}

func TestFusionOwnerBindsAlertAssociationAndReviewContracts(t *testing.T) {
	owners := make([]string, 0)
	for _, owned := range FirstPhaseOwnedContracts() {
		if owned.Owner == OwnerFusion {
			owners = append(owners, owned.Contract.MessageType)
		}
	}

	want := []string{
		FusionAlertContract().MessageType,
		FusionTrackAssociationContract().MessageType,
		FusionAssociationReviewContract().MessageType,
	}
	if !reflect.DeepEqual(owners, want) {
		t.Fatalf("fusion contracts = %#v, want %#v", owners, want)
	}

	registration, err := projection.Derive(
		OwnerFusion,
		FusionAlertContract(),
		FusionTrackAssociationContract(),
		FusionAssociationReviewContract(),
	)
	if err != nil {
		t.Fatalf("derive fusion grouped contracts: %v", err)
	}
	if len(registration.ForeignEdges) != 3 {
		t.Fatalf("fusion foreign edges = %d, want primary + candidate track + review association", len(registration.ForeignEdges))
	}
	var trackEdges, associationReviewEdges int
	for _, edge := range registration.ForeignEdges {
		if edge.Mode != ownership.EdgeStrict {
			t.Fatalf("fusion edge mode = %q, want strict", edge.Mode)
		}
		switch edge.TargetPattern {
		case EntityPattern(EntityTrack):
			trackEdges++
		case SourceEntityPattern("fusion", EntityAssociation):
			associationReviewEdges++
		default:
			t.Fatalf("fusion target pattern = %q", edge.TargetPattern)
		}
	}
	if trackEdges != 2 || associationReviewEdges != 1 {
		t.Fatalf("fusion edge targets track=%d review-association=%d, want 2/1", trackEdges, associationReviewEdges)
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
	for _, tc := range []struct {
		owned         OwnedContract
		wantPredicate string
	}{
		{owned: OwnedContract{Owner: OwnerCommand, Contract: CommandIntentContract()}, wantPredicate: TaskTarget},
		{owned: OwnedContract{Owner: OwnerMAVLink, Contract: MAVLinkTrackContract()}, wantPredicate: TrackSource},
		{owned: OwnedContract{Owner: OwnerMAVLink, Contract: MAVLinkCommandTaskContract()}, wantPredicate: TaskTarget},
		{owned: OwnedContract{Owner: OwnerTAK, Contract: TAKTrackContract()}, wantPredicate: TrackSource},
	} {
		contract := tc.owned.Contract
		registration, err := projection.Derive(tc.owned.Owner, contract)
		if err != nil {
			t.Fatalf("%s should derive: %v", contract.Name, err)
		}
		if len(registration.ForeignEdges) != 1 {
			t.Fatalf("%s foreign edges = %d, want 1", contract.Name, len(registration.ForeignEdges))
		}

		edge := registration.ForeignEdges[0]
		if edge.Predicate != tc.wantPredicate {
			t.Fatalf("%s foreign edge predicate = %q, want %q", contract.Name, edge.Predicate, tc.wantPredicate)
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

func TestFusionAssociationForeignEdgesDeclareTrackTargets(t *testing.T) {
	contract := FusionTrackAssociationContract()
	registration, err := projection.Derive(OwnerFusion, contract)
	if err != nil {
		t.Fatalf("%s should derive: %v", contract.Name, err)
	}
	if len(registration.ForeignEdges) != 2 {
		t.Fatalf("fusion association foreign edges = %d, want 2", len(registration.ForeignEdges))
	}

	wantPredicates := map[string]bool{
		AssociationPrimaryTrack:   false,
		AssociationCandidateTrack: false,
	}
	for _, edge := range registration.ForeignEdges {
		if _, ok := wantPredicates[edge.Predicate]; !ok {
			t.Fatalf("unexpected fusion association edge predicate %q", edge.Predicate)
		}
		wantPredicates[edge.Predicate] = true
		if edge.Producer != contract.MessageType {
			t.Fatalf("producer = %q, want %q", edge.Producer, contract.MessageType)
		}
		if edge.TargetPattern != EntityPattern(EntityTrack) {
			t.Fatalf("target pattern = %q, want %q", edge.TargetPattern, EntityPattern(EntityTrack))
		}
		if edge.Mode != ownership.EdgeStrict {
			t.Fatalf("edge mode = %q, want strict born-first edge", edge.Mode)
		}
	}
	for predicate, seen := range wantPredicates {
		if !seen {
			t.Fatalf("missing fusion association edge predicate %q", predicate)
		}
	}
}

func TestFusionAssociationReviewForeignEdgeDeclaresAssociationTarget(t *testing.T) {
	contract := FusionAssociationReviewContract()
	registration, err := projection.Derive(OwnerFusion, contract)
	if err != nil {
		t.Fatalf("%s should derive: %v", contract.Name, err)
	}
	if len(registration.ForeignEdges) != 1 {
		t.Fatalf("fusion association review foreign edges = %d, want 1", len(registration.ForeignEdges))
	}
	edge := registration.ForeignEdges[0]
	if edge.Predicate != AssociationReviewAssociation {
		t.Fatalf("edge predicate = %q, want %q", edge.Predicate, AssociationReviewAssociation)
	}
	if edge.Producer != contract.MessageType {
		t.Fatalf("producer = %q, want %q", edge.Producer, contract.MessageType)
	}
	if edge.TargetPattern != SourceEntityPattern("fusion", EntityAssociation) {
		t.Fatalf("target pattern = %q, want fusion association pattern", edge.TargetPattern)
	}
	if edge.Mode != ownership.EdgeStrict {
		t.Fatalf("edge mode = %q, want strict born-first edge", edge.Mode)
	}
}

func TestFusionAssociationReviewContractOwnsNonAuthoritativeReviewSemantics(t *testing.T) {
	contract := FusionAssociationReviewContract()
	seen := make(map[string]bool)
	for _, group := range contract.Groups {
		if group.Mode != ownership.ModeReplaceOwned {
			t.Fatalf("association review group mode = %q, want replace-owned", group.Mode)
		}
		for _, predicate := range group.Predicates {
			seen[predicate] = true
		}
	}

	for _, predicate := range []string{
		AssociationReviewReviewerRole,
		AssociationReviewAuthorityScope,
		AssociationReviewConflictPolicy,
	} {
		if !seen[predicate] {
			t.Fatalf("association review contract missing semantic predicate %q", predicate)
		}
	}
}

func TestCommandIntentContractCarriesImpedanceFieldsWithoutNativeStatusAuthority(t *testing.T) {
	contract := CommandIntentContract()
	seen := make(map[string]bool)
	for _, group := range contract.Groups {
		if group.Mode != ownership.ModeReplaceOwned {
			t.Fatalf("command intent group mode = %q, want replace-owned", group.Mode)
		}
		for _, predicate := range group.Predicates {
			seen[predicate] = true
			switch predicate {
			case TrackPosition,
				TrackVelocity,
				TrackStatus,
				TrackObservedAt,
				TrackNativeID,
				TrackSource:
				t.Fatalf("command intent contract must not own native track predicate %q", predicate)
			}
		}
	}

	for _, predicate := range []string{
		TaskDesired,
		TaskAuthority,
		TaskPriority,
		TaskExpiresAt,
		TaskCorrelation,
		TaskIdempotency,
		TaskRequestedBy,
		TaskStatus,
	} {
		if !seen[predicate] {
			t.Fatalf("command intent contract missing impedance predicate %q", predicate)
		}
	}
	if len(contract.ForeignEdges) != 1 || contract.ForeignEdges[0].Predicate != TaskTarget {
		t.Fatalf("command intent foreign edges = %+v, want strict task target", contract.ForeignEdges)
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

func TestSAPIENTTrackContractDoesNotClaimTaskAlertOrAssociationAuthority(t *testing.T) {
	contract := SAPIENTTrackContract()
	for _, group := range contract.Groups {
		if group.Mode != ownership.ModeReplaceOwned {
			t.Fatalf("SAPIENT group mode = %q, want replace-owned signal state", group.Mode)
		}
		for _, predicate := range group.Predicates {
			switch predicate {
			case TrackSource,
				TaskName,
				TaskStatus,
				TaskPosition,
				TaskDescription,
				AlertSeverity,
				AlertStatus,
				AlertReason,
				AlertAffectedEntity,
				AlertSource:
				t.Fatalf("SAPIENT track contract must not own authority predicate %q", predicate)
			}
		}
	}
}

func TestWeatherObservationContractDoesNotClaimHazardOrDecisionAuthority(t *testing.T) {
	contract := WeatherObservationContract()
	for _, group := range contract.Groups {
		if group.Mode != ownership.ModeReplaceOwned {
			t.Fatalf("weather group mode = %q, want replace-owned signal state", group.Mode)
		}
		for _, predicate := range group.Predicates {
			switch predicate {
			case HazardGeometry,
				HazardSeverity,
				HazardStatus,
				HazardAdvisoryText,
				HazardEvidence,
				HazardSource,
				AlertSeverity,
				AlertStatus,
				AlertReason,
				AlertAffectedEntity,
				AlertSource,
				TaskName,
				TaskStatus,
				TaskPosition:
				t.Fatalf("weather observation contract must not own authority predicate %q", predicate)
			}
		}
	}
}
