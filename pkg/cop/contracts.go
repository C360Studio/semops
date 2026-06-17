// Package cop defines the SemOps common operating picture model and projection
// contracts.
package cop

import (
	"github.com/c360studio/semstreams/pkg/ownership"
	"github.com/c360studio/semstreams/pkg/projection"
)

const (
	EntityTrack           = "track"
	EntityAsset           = "asset"
	EntityHazardArea      = "hazard_area"
	EntitySensorFootprint = "sensor_footprint"
	EntityAlert           = "alert"
	EntityTask            = "task"
	EntityAdvisory        = "advisory"
)

const (
	OwnerMAVLink = "semops.feed.mavlink"
	OwnerTAK     = "semops.feed.tak"
	OwnerAsset   = "semops.feed.asset"
	OwnerCAP     = "semops.feed.cap"
	OwnerFusion  = "semops.fusion.structural"
)

const (
	TrackPosition   = "cop.track.position"
	TrackVelocity   = "cop.track.velocity"
	TrackStatus     = "cop.track.status"
	TrackObservedAt = "cop.track.observed_at"
	TrackNativeID   = "cop.track.native_id"
	TrackSource     = "cop.track.source"

	AssetName     = "cop.asset.name"
	AssetKind     = "cop.asset.kind"
	AssetSource   = "cop.asset.source"
	AssetNativeID = "cop.asset.native_id"

	HazardGeometry     = "cop.hazard.geometry"
	HazardSeverity     = "cop.hazard.severity"
	HazardStatus       = "cop.hazard.status"
	HazardAdvisoryText = "cop.hazard.advisory_text"
	HazardEvidence     = "cop.hazard.evidence"
	HazardSource       = "cop.hazard.source"

	AlertSeverity       = "cop.alert.severity"
	AlertStatus         = "cop.alert.status"
	AlertReason         = "cop.alert.reason"
	AlertAffectedEntity = "cop.alert.affected_entity"
	AlertSource         = "cop.alert.source"

	ProvenanceSource     = "cop.provenance.source"
	ProvenanceConfidence = "cop.provenance.confidence"
	ProvenanceObservedAt = "cop.provenance.observed_at"
)

var FirstCanonicalEntitySet = []string{
	EntityTrack,
	EntityAsset,
	EntityHazardArea,
	EntitySensorFootprint,
	EntityAlert,
	EntityTask,
	EntityAdvisory,
}

// OwnedContract binds a product projection contract to the SemOps owner that
// registers it with SemStreams.
type OwnedContract struct {
	Owner    string
	Contract projection.Contract
}

// EntityPattern returns the 6-part SemStreams entity glob for a COP entity type.
func EntityPattern(entityType string) string {
	return "c360.*.cop.*." + entityType + ".*"
}

// SourceEntityPattern returns the 6-part entity glob for a source-partitioned
// COP entity owner. Strict feed owners use this to avoid cross-feed predicate
// clobbering on shared entity types.
func SourceEntityPattern(system, entityType string) string {
	return "c360.*.cop." + system + "." + entityType + ".*"
}

// SourceAssetContract owns the source asset identities that strict feed track
// foreign edges target. Feed adapters must birth these entities before writing
// foreign edges; ADR-055 removes triple.add auto-vivify as an ordering crutch.
func SourceAssetContract() projection.Contract {
	return projection.Contract{
		Name:            "semops.cop.asset.source-current-state",
		MessageType:     "semops.source.asset.v1",
		EntityPattern:   EntityPattern(EntityAsset),
		IndexingProfile: "control",
		Groups: []projection.PredicateGroup{{
			Mode: ownership.ModeReplaceOwned,
			Predicates: []string{
				AssetName,
				AssetKind,
				AssetSource,
				AssetNativeID,
				ProvenanceSource,
				ProvenanceConfidence,
				ProvenanceObservedAt,
			},
		}},
	}
}

// MAVLinkTrackContract owns current vehicle state projected from MAVLink.
func MAVLinkTrackContract() projection.Contract {
	return projection.Contract{
		Name:            "semops.cop.track.mavlink-current-state",
		MessageType:     "semops.mavlink.track.v1",
		EntityPattern:   SourceEntityPattern("mavlink", EntityTrack),
		IndexingProfile: "signal",
		Groups: []projection.PredicateGroup{{
			Mode: ownership.ModeReplaceOwned,
			Predicates: []string{
				TrackPosition,
				TrackVelocity,
				TrackStatus,
				TrackObservedAt,
				TrackNativeID,
				ProvenanceSource,
				ProvenanceConfidence,
				ProvenanceObservedAt,
			},
		}},
		ForeignEdges: []projection.ForeignEdge{{
			Predicate:     TrackSource,
			Mode:          ownership.EdgeStrict,
			TargetPattern: EntityPattern(EntityAsset),
		}},
	}
}

// TAKTrackContract owns current track state projected from TAK/CoT markers.
func TAKTrackContract() projection.Contract {
	return projection.Contract{
		Name:            "semops.cop.track.tak-current-state",
		MessageType:     "semops.tak.track.v1",
		EntityPattern:   SourceEntityPattern("tak", EntityTrack),
		IndexingProfile: "signal",
		Groups: []projection.PredicateGroup{{
			Mode: ownership.ModeReplaceOwned,
			Predicates: []string{
				TrackPosition,
				TrackVelocity,
				TrackStatus,
				TrackObservedAt,
				TrackNativeID,
				ProvenanceSource,
				ProvenanceConfidence,
				ProvenanceObservedAt,
			},
		}},
		ForeignEdges: []projection.ForeignEdge{{
			Predicate:     TrackSource,
			Mode:          ownership.EdgeStrict,
			TargetPattern: EntityPattern(EntityAsset),
		}},
	}
}

// CAPHazardEvidenceContract appends loose civil-alert evidence without taking
// ownership of current hazard geometry or status.
func CAPHazardEvidenceContract() projection.Contract {
	return projection.Contract{
		Name:            "semops.cop.hazard.cap-evidence",
		MessageType:     "semops.cap.hazard.v1",
		EntityPattern:   SourceEntityPattern("cap", EntityHazardArea),
		IndexingProfile: "content",
		Groups: []projection.PredicateGroup{{
			Mode: ownership.ModeAppendEvidence,
			Predicates: []string{
				HazardAdvisoryText,
				HazardEvidence,
				HazardSource,
				ProvenanceSource,
				ProvenanceConfidence,
				ProvenanceObservedAt,
			},
		}},
	}
}

// FusionAlertContract owns derived alert state produced by deterministic COP
// fusion, such as hazard/asset intersection or stale-track detection.
func FusionAlertContract() projection.Contract {
	return projection.Contract{
		Name:            "semops.cop.alert.fusion-current-state",
		MessageType:     "semops.fusion.alert.v1",
		EntityPattern:   SourceEntityPattern("fusion", EntityAlert),
		IndexingProfile: "control",
		Groups: []projection.PredicateGroup{{
			Mode: ownership.ModeReplaceOwned,
			Predicates: []string{
				AlertSeverity,
				AlertStatus,
				AlertReason,
				AlertAffectedEntity,
				AlertSource,
				ProvenanceConfidence,
				ProvenanceObservedAt,
			},
		}},
	}
}

// FirstPhaseOwnedContracts returns the initial strict, tolerant, and derived
// contract set used to gate first-phase feed implementation.
func FirstPhaseOwnedContracts() []OwnedContract {
	return []OwnedContract{
		{Owner: OwnerAsset, Contract: SourceAssetContract()},
		{Owner: OwnerMAVLink, Contract: MAVLinkTrackContract()},
		{Owner: OwnerTAK, Contract: TAKTrackContract()},
		{Owner: OwnerCAP, Contract: CAPHazardEvidenceContract()},
		{Owner: OwnerFusion, Contract: FusionAlertContract()},
	}
}
