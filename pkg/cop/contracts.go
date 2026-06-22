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
	OwnerADSB    = "semops.feed.adsb"
	OwnerAsset   = "semops.feed.asset"
	OwnerCAP     = "semops.feed.cap"
	OwnerKLV     = "semops.feed.klv"
	OwnerFusion  = "semops.fusion.structural"
)

const (
	TrackPosition   = "cop.track.position"
	TrackVelocity   = "cop.track.velocity"
	TrackStatus     = "cop.track.status"
	TrackObservedAt = "cop.track.observed_at"
	TrackNativeID   = "cop.track.native_id"
	TrackSource     = "cop.track.source"
	TrackRoll       = "cop.track.roll"
	TrackPitch      = "cop.track.pitch"
	TrackYaw        = "cop.track.yaw"
	TrackBattery    = "cop.track.battery_remaining"

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

	TaskName        = "cop.task.name"
	TaskKind        = "cop.task.kind"
	TaskStatus      = "cop.task.status"
	TaskPosition    = "cop.task.position"
	TaskDescription = "cop.task.description"
	TaskNativeID    = "cop.task.native_id"

	AdvisoryText     = "cop.advisory.text"
	AdvisoryKind     = "cop.advisory.kind"
	AdvisoryStatus   = "cop.advisory.status"
	AdvisorySender   = "cop.advisory.sender"
	AdvisoryPosition = "cop.advisory.position"
	AdvisoryNativeID = "cop.advisory.native_id"

	SensorFootprintNativeID             = "cop.sensor_footprint.native_id"
	SensorFootprintSource               = "cop.sensor_footprint.source"
	SensorFootprintMediaRef             = "cop.sensor_footprint.media_ref"
	SensorFootprintPacketRef            = "cop.sensor_footprint.packet_ref"
	SensorFootprintObservedAt           = "cop.sensor_footprint.observed_at"
	SensorFootprintSensorPosition       = "cop.sensor_footprint.sensor_position"
	SensorFootprintSensorAltitude       = "cop.sensor_footprint.sensor_altitude_meters"
	SensorFootprintSensorAzimuth        = "cop.sensor_footprint.sensor_azimuth_degrees"
	SensorFootprintSensorElevation      = "cop.sensor_footprint.sensor_elevation_degrees"
	SensorFootprintFrameCenter          = "cop.sensor_footprint.frame_center"
	SensorFootprintFrameCenterElevation = "cop.sensor_footprint.frame_center_elevation_meters"
	SensorFootprintPlatformDesignation  = "cop.sensor_footprint.platform_designation"

	ProvenanceSource     = "cop.provenance.source"
	ProvenanceConfidence = "cop.provenance.confidence"
	ProvenanceObservedAt = "cop.provenance.observed_at"
	ProvenanceSourceRef  = "cop.provenance.source_ref"
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
				ProvenanceSourceRef,
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
				TrackRoll,
				TrackPitch,
				TrackYaw,
				TrackBattery,
				ProvenanceSource,
				ProvenanceConfidence,
				ProvenanceObservedAt,
				ProvenanceSourceRef,
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
				ProvenanceSourceRef,
			},
		}},
		ForeignEdges: []projection.ForeignEdge{{
			Predicate:     TrackSource,
			Mode:          ownership.EdgeStrict,
			TargetPattern: EntityPattern(EntityAsset),
		}},
	}
}

// ADSBTrackContract owns current aircraft state projected from ADS-B shaped
// feeds such as OpenSky snapshots. Association with MAVLink, SAPIENT, or fusion
// tracks is deliberately separate fusion evidence.
func ADSBTrackContract() projection.Contract {
	return projection.Contract{
		Name:            "semops.cop.track.adsb-current-state",
		MessageType:     "semops.adsb.track.v1",
		EntityPattern:   SourceEntityPattern("adsb", EntityTrack),
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
				ProvenanceSourceRef,
			},
		}},
	}
}

// KLVSensorFootprintContract owns current video-derived sensor geometry for the
// supported MISB ST 0601 subset. Full footprint polygons remain a later
// contract extension.
func KLVSensorFootprintContract() projection.Contract {
	return projection.Contract{
		Name:            "semops.cop.sensor-footprint.klv-current-state",
		MessageType:     "semops.klv.sensor_footprint.v1",
		EntityPattern:   SourceEntityPattern("klv", EntitySensorFootprint),
		IndexingProfile: "signal",
		Groups: []projection.PredicateGroup{{
			Mode: ownership.ModeReplaceOwned,
			Predicates: []string{
				SensorFootprintNativeID,
				SensorFootprintSource,
				SensorFootprintMediaRef,
				SensorFootprintPacketRef,
				SensorFootprintObservedAt,
				SensorFootprintSensorPosition,
				SensorFootprintSensorAltitude,
				SensorFootprintSensorAzimuth,
				SensorFootprintSensorElevation,
				SensorFootprintFrameCenter,
				SensorFootprintFrameCenterElevation,
				SensorFootprintPlatformDesignation,
				ProvenanceSource,
				ProvenanceConfidence,
				ProvenanceObservedAt,
				ProvenanceSourceRef,
			},
		}},
	}
}

// TAKTaskContract owns durable TAK/CoT control state such as map markers and
// operator intent points.
func TAKTaskContract() projection.Contract {
	return projection.Contract{
		Name:            "semops.cop.task.tak-current-state",
		MessageType:     "semops.tak.task.v1",
		EntityPattern:   SourceEntityPattern("tak", EntityTask),
		IndexingProfile: "control",
		Groups: []projection.PredicateGroup{{
			Mode: ownership.ModeReplaceOwned,
			Predicates: []string{
				TaskName,
				TaskKind,
				TaskStatus,
				TaskPosition,
				TaskDescription,
				TaskNativeID,
				ProvenanceSource,
				ProvenanceConfidence,
				ProvenanceObservedAt,
				ProvenanceSourceRef,
			},
		}},
	}
}

// TAKAdvisoryContract owns TAK/CoT textual content such as GeoChat messages.
func TAKAdvisoryContract() projection.Contract {
	return projection.Contract{
		Name:            "semops.cop.advisory.tak-content",
		MessageType:     "semops.tak.advisory.v1",
		EntityPattern:   SourceEntityPattern("tak", EntityAdvisory),
		IndexingProfile: "content",
		Groups: []projection.PredicateGroup{{
			Mode: ownership.ModeReplaceOwned,
			Predicates: []string{
				AdvisoryText,
				AdvisoryKind,
				AdvisoryStatus,
				AdvisorySender,
				AdvisoryPosition,
				AdvisoryNativeID,
				ProvenanceSource,
				ProvenanceConfidence,
				ProvenanceObservedAt,
				ProvenanceSourceRef,
			},
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
				ProvenanceSourceRef,
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
		{Owner: OwnerTAK, Contract: TAKTaskContract()},
		{Owner: OwnerTAK, Contract: TAKAdvisoryContract()},
		{Owner: OwnerCAP, Contract: CAPHazardEvidenceContract()},
		{Owner: OwnerFusion, Contract: FusionAlertContract()},
	}
}
