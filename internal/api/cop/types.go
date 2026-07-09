package cop

import "time"

type Snapshot struct {
	GeneratedAt      time.Time            `json:"generated_at"`
	Scenario         string               `json:"scenario"`
	Summary          Summary              `json:"summary"`
	Diagnostics      SnapshotDiagnostics  `json:"diagnostics"`
	Feeds            []FeedHealth         `json:"feeds"`
	Assets           []Asset              `json:"assets"`
	Tracks           []Track              `json:"tracks"`
	Tasks            []Task               `json:"tasks"`
	Advisories       []Advisory           `json:"advisories"`
	Hazards          []Hazard             `json:"hazards"`
	SensorFootprints []SensorFootprint    `json:"sensor_footprints"`
	Weather          []WeatherObservation `json:"weather_observations"`
	Associations     []Association        `json:"associations"`
	CompanionFleets  []CompanionFleet     `json:"companion_fleets"`
	Alerts           []Alert              `json:"alerts"`
}

type Summary struct {
	ActiveTracks           int `json:"active_tracks"`
	ActiveTasks            int `json:"active_tasks"`
	ActiveAdvisories       int `json:"active_advisories"`
	ActiveSensorFootprints int `json:"active_sensor_footprints"`
	ActiveWeather          int `json:"active_weather_observations"`
	ActiveAssociations     int `json:"active_associations"`
	ActiveCompanionNodes   int `json:"active_companion_nodes"`
	ActiveAlerts           int `json:"active_alerts"`
	StaleFeeds             int `json:"stale_feeds"`
}

type SnapshotDiagnostics struct {
	Discovery []DiscoveryDiagnostic `json:"discovery"`
}

type DiscoveryDiagnostic struct {
	Org        string `json:"org"`
	Platform   string `json:"platform"`
	Source     string `json:"source"`
	Family     string `json:"family"`
	EntityType string `json:"entity_type"`
	Prefix     string `json:"prefix"`
	Count      int    `json:"count"`
	Limit      int    `json:"limit"`
	AtLimit    bool   `json:"at_limit"`
	Error      string `json:"error,omitempty"`
}

type FeedHealth struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Kind        string    `json:"kind"`
	Status      string    `json:"status"`
	LastEventAt time.Time `json:"last_event_at"`
	Message     string    `json:"message"`
}

type Asset struct {
	ID         string     `json:"id"`
	Label      string     `json:"label"`
	Kind       string     `json:"kind"`
	Source     string     `json:"source"`
	Position   *GeoPoint  `json:"position,omitempty"`
	Confidence float64    `json:"confidence"`
	UpdatedAt  time.Time  `json:"updated_at"`
	Provenance Provenance `json:"provenance"`
}

type Track struct {
	ID         string     `json:"id"`
	Label      string     `json:"label"`
	Source     string     `json:"source"`
	Status     string     `json:"status"`
	Position   GeoPoint   `json:"position"`
	Velocity   string     `json:"velocity"`
	Confidence float64    `json:"confidence"`
	UpdatedAt  time.Time  `json:"updated_at"`
	Provenance Provenance `json:"provenance"`
}

type Task struct {
	ID                  string     `json:"id"`
	Label               string     `json:"label"`
	Kind                string     `json:"kind"`
	Source              string     `json:"source"`
	Status              string     `json:"status"`
	Position            *GeoPoint  `json:"position,omitempty"`
	Description         string     `json:"description,omitempty"`
	TargetID            string     `json:"target_id,omitempty"`
	Authority           string     `json:"authority,omitempty"`
	Priority            *int       `json:"priority,omitempty"`
	ExpiresAt           *time.Time `json:"expires_at,omitempty"`
	RequestedBy         string     `json:"requested_by,omitempty"`
	CorrelationID       string     `json:"correlation_id,omitempty"`
	DesiredState        string     `json:"desired_state,omitempty"`
	LocalOverridePolicy string     `json:"local_override_policy,omitempty"`
	ClaimPosture        string     `json:"claim_posture,omitempty"`
	Confidence          float64    `json:"confidence"`
	UpdatedAt           time.Time  `json:"updated_at"`
	Provenance          Provenance `json:"provenance"`
}

type Advisory struct {
	ID         string     `json:"id"`
	Label      string     `json:"label"`
	Kind       string     `json:"kind"`
	Source     string     `json:"source"`
	Status     string     `json:"status"`
	Text       string     `json:"text"`
	Sender     string     `json:"sender,omitempty"`
	Position   *GeoPoint  `json:"position,omitempty"`
	Confidence float64    `json:"confidence"`
	UpdatedAt  time.Time  `json:"updated_at"`
	Provenance Provenance `json:"provenance"`
}

type Hazard struct {
	ID         string     `json:"id"`
	Label      string     `json:"label"`
	Kind       string     `json:"kind"`
	Severity   string     `json:"severity"`
	Status     string     `json:"status"`
	Geometry   []GeoPoint `json:"geometry"`
	Source     string     `json:"source"`
	Confidence float64    `json:"confidence"`
	UpdatedAt  time.Time  `json:"updated_at"`
	Provenance Provenance `json:"provenance"`
}

type SensorFootprint struct {
	ID                         string     `json:"id"`
	Label                      string     `json:"label"`
	Source                     string     `json:"source"`
	Status                     string     `json:"status"`
	SensorPosition             GeoPoint   `json:"sensor_position"`
	FrameCenter                GeoPoint   `json:"frame_center"`
	Ray                        []GeoPoint `json:"ray"`
	Footprint                  []GeoPoint `json:"footprint,omitempty"`
	SensorAltitudeMeters       *float64   `json:"sensor_altitude_meters,omitempty"`
	SensorAzimuthDegrees       *float64   `json:"sensor_azimuth_degrees,omitempty"`
	SensorElevationDegrees     *float64   `json:"sensor_elevation_degrees,omitempty"`
	FrameCenterElevationMeters *float64   `json:"frame_center_elevation_meters,omitempty"`
	MediaRef                   string     `json:"media_ref"`
	PacketRef                  string     `json:"packet_ref"`
	FrameTime                  time.Time  `json:"frame_time"`
	PlatformDesignation        string     `json:"platform_designation,omitempty"`
	ClaimPosture               string     `json:"claim_posture"`
	DecodedFields              []string   `json:"decoded_fields"`
	Warnings                   []string   `json:"warnings"`
	Confidence                 float64    `json:"confidence"`
	UpdatedAt                  time.Time  `json:"updated_at"`
	Provenance                 Provenance `json:"provenance"`
}

type WeatherObservation struct {
	ID               string     `json:"id"`
	Label            string     `json:"label"`
	Source           string     `json:"source"`
	Status           string     `json:"status"`
	Provider         string     `json:"provider"`
	QueryShape       string     `json:"query_shape"`
	QueryGeometryWKT string     `json:"query_geometry_wkt"`
	Position         *GeoPoint  `json:"position,omitempty"`
	ValidTime        time.Time  `json:"valid_time"`
	ModelTime        time.Time  `json:"model_time,omitempty"`
	FreshUntil       time.Time  `json:"fresh_until,omitempty"`
	Variable         string     `json:"variable"`
	Value            float64    `json:"value"`
	Unit             string     `json:"unit,omitempty"`
	ClaimPosture     string     `json:"claim_posture"`
	Confidence       float64    `json:"confidence"`
	UpdatedAt        time.Time  `json:"updated_at"`
	Provenance       Provenance `json:"provenance"`
}

type Association struct {
	ID               string             `json:"id"`
	Label            string             `json:"label"`
	Kind             string             `json:"kind"`
	Source           string             `json:"source"`
	Status           string             `json:"status"`
	PrimaryTrackID   string             `json:"primary_track_id"`
	CandidateTrackID string             `json:"candidate_track_id"`
	Algorithm        string             `json:"algorithm"`
	Reason           string             `json:"reason"`
	DistanceMeters   *float64           `json:"distance_meters,omitempty"`
	TimeDeltaSeconds *float64           `json:"time_delta_seconds,omitempty"`
	ClaimPosture     string             `json:"claim_posture"`
	Confidence       float64            `json:"confidence"`
	UpdatedAt        time.Time          `json:"updated_at"`
	Provenance       Provenance         `json:"provenance"`
	OperatorReview   *AssociationReview `json:"operator_review,omitempty"`
}

type Alert struct {
	ID        string    `json:"id"`
	Label     string    `json:"label"`
	Severity  string    `json:"severity"`
	Status    string    `json:"status"`
	EntityID  string    `json:"entity_id"`
	Reason    string    `json:"reason"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CompanionFleet struct {
	ID                  string                       `json:"id"`
	Label               string                       `json:"label"`
	Source              string                       `json:"source"`
	Status              string                       `json:"status"`
	EvidenceKind        string                       `json:"evidence_kind"`
	VehicleProfile      string                       `json:"vehicle_profile"`
	NodeCount           int                          `json:"node_count"`
	VehicleCount        int                          `json:"vehicle_count"`
	ExpectedSummaries   int                          `json:"expected_summaries,omitempty"`
	AssertionState      string                       `json:"assertion_state"`
	NoTransmitPosture   string                       `json:"no_transmit_posture"`
	RawMAVLinkExcluded  bool                         `json:"raw_mavlink_excluded"`
	RawMAVLinkPolicy    string                       `json:"raw_mavlink_policy,omitempty"`
	DemoEvidenceLabel   string                       `json:"demo_evidence_label"`
	Confidence          float64                      `json:"confidence"`
	UpdatedAt           time.Time                    `json:"updated_at"`
	Nodes               []CompanionNode              `json:"nodes"`
	Readback            CompanionReadback            `json:"readback"`
	CommandPosture      CompanionCommandPosture      `json:"command_posture"`
	RawMAVLinkExclusion CompanionRawMAVLinkExclusion `json:"raw_mavlink_exclusion"`
	Assertions          []CompanionAssertion         `json:"assertions"`
	Provenance          Provenance                   `json:"provenance"`
}

type CompanionNode struct {
	ID                  string `json:"id"`
	VehicleCount        int    `json:"vehicle_count"`
	PeerCount           int    `json:"peer_count"`
	InitialSummaryCount int    `json:"initial_summary_count,omitempty"`
	FinalSummaryCount   int    `json:"final_summary_count,omitempty"`
	WatermarkCount      int    `json:"watermark_count,omitempty"`
	AppliedDiffCount    int    `json:"applied_diff_count,omitempty"`
	DiffItemCount       int    `json:"diff_item_count,omitempty"`
	TTLMergePosture     string `json:"ttl_merge_posture,omitempty"`
}

type CompanionReadback struct {
	AdapterStatus            string    `json:"adapter_status"`
	Accepted                 bool      `json:"accepted"`
	ReceivedRequests         int       `json:"received_requests"`
	CorrelationID            string    `json:"correlation_id,omitempty"`
	NativeExecutionAllowed   bool      `json:"native_execution_allowed"`
	CompanionTransmitAllowed bool      `json:"companion_transmit_allowed"`
	CommandACKStatus         string    `json:"command_ack_status"`
	CommandACKResult         string    `json:"command_ack_result,omitempty"`
	CommandACKObservedAt     time.Time `json:"command_ack_observed_at,omitempty"`
	ResultStatus             string    `json:"result_status"`
	ResultObservedProperty   string    `json:"result_observed_property,omitempty"`
	ResultObservedAt         time.Time `json:"result_observed_at,omitempty"`
}

type CompanionCommandPosture struct {
	Status                     string `json:"status"`
	HardwareBlocked            bool   `json:"hardware_blocked"`
	SimulatorOnly              bool   `json:"simulator_only"`
	PreflightAccepted          bool   `json:"preflight_accepted"`
	ACKAccepted                bool   `json:"ack_accepted"`
	PostStateObserved          bool   `json:"post_state_observed"`
	HardwareTransmitAuthorized bool   `json:"hardware_transmit_authorized"`
}

type CompanionRawMAVLinkExclusion struct {
	RejectedBySummaryIndex bool   `json:"rejected_by_summary_index"`
	Policy                 string `json:"policy,omitempty"`
	Error                  string `json:"error,omitempty"`
}

type CompanionAssertion struct {
	Name   string `json:"name"`
	Passed bool   `json:"passed"`
	Detail string `json:"detail,omitempty"`
}

type GeoPoint struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

type Provenance struct {
	Owner     string    `json:"owner"`
	SourceRef string    `json:"source_ref"`
	Observed  time.Time `json:"observed_at"`
}
