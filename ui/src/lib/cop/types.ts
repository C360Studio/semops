export type Snapshot = {
  generated_at: string;
  scenario: string;
  summary: Summary;
  diagnostics?: SnapshotDiagnostics;
  feeds: FeedHealth[];
  assets: Asset[];
  tracks: Track[];
  tasks: Task[];
  advisories: Advisory[];
  hazards: Hazard[];
  sensor_footprints: SensorFootprint[];
  weather_observations: WeatherObservation[];
  associations: Association[];
  companion_fleets: CompanionFleet[];
  alerts: Alert[];
};

export type Summary = {
  active_tracks: number;
  active_tasks: number;
  active_advisories: number;
  active_sensor_footprints: number;
  active_weather_observations: number;
  active_associations: number;
  active_companion_nodes: number;
  active_alerts: number;
  stale_feeds: number;
};

export type SnapshotDiagnostics = {
  discovery: DiscoveryDiagnostic[];
};

export type DiscoveryDiagnostic = {
  org: string;
  platform: string;
  source: string;
  family: string;
  entity_type: string;
  prefix: string;
  count: number;
  limit: number;
  at_limit: boolean;
  error?: string;
};

export type FeedHealth = {
  id: string;
  name: string;
  kind: string;
  status: 'live' | 'planned' | 'stale' | 'down' | string;
  last_event_at: string;
  message: string;
};

export type RuntimeSnapshot = {
  generated_at: string;
  feeds: RuntimeFeed[];
  components: RuntimeComponent[];
};

export type RuntimeFeed = {
  id: string;
  name: string;
  status: 'flowing' | 'idle' | 'stale' | 'degraded' | string;
  message: string;
  healthy_components: number;
  total_components: number;
  messages_per_second: number;
  last_activity?: string;
  last_activity_age_seconds?: number;
};

export type RuntimeComponent = {
  name: string;
  feed: string;
  role: string;
  type: string;
  status: string;
  healthy: boolean;
  messages_per_second: number;
  bytes_per_second: number;
  error_rate: number;
  error_count: number;
  last_activity?: string;
  last_check?: string;
  uptime_seconds: number;
};

export type ScenarioStatus = {
  scenario_id: string;
  state: 'idle' | 'running' | 'succeeded' | 'failed' | string;
  ingress_mode?: 'feed-boundary' | 'direct-graph-contract' | string;
  current_step?: string;
  started_at?: string;
  updated_at?: string;
  finished_at?: string;
  completed_steps: number;
  failed_steps: number;
  last_error?: string;
  summary: ScenarioSummary;
};

export type ScenarioControls = {
  enabled: boolean;
  state: 'blocked' | 'enabled' | string;
  reason: string;
  supported_actions: ScenarioControlAction[];
  required_claim_scope: string;
  scenario_id?: string;
  checkpoint_id?: string;
  checkpoint_state?: string;
};

export type ScenarioControlAction = 'start' | 'reset' | 'pause' | 'resume' | string;

export type ScenarioSummary = {
  mavlink_frames: number;
  cot_events: number;
  cap_alerts: number;
  adsb_snapshots: number;
  feed_boundary_deliveries: number;
  contract_graph_mutation_attempts: number;
  mutations: number;
  errors: number;
};

export type Asset = {
  id: string;
  label: string;
  kind: string;
  source: string;
  position?: GeoPoint;
  confidence: number;
  updated_at: string;
  provenance: Provenance;
};

export type Track = {
  id: string;
  label: string;
  source: string;
  status: string;
  position: GeoPoint;
  velocity: string;
  confidence: number;
  updated_at: string;
  provenance: Provenance;
};

export type Task = {
  id: string;
  label: string;
  kind: string;
  source: string;
  status: string;
  position?: GeoPoint;
  description?: string;
  target_id?: string;
  authority?: string;
  priority?: number;
  expires_at?: string;
  requested_by?: string;
  correlation_id?: string;
  desired_state?: string;
  local_override_policy?: string;
  claim_posture?: string;
  confidence: number;
  updated_at: string;
  provenance: Provenance;
};

export type Advisory = {
  id: string;
  label: string;
  kind: string;
  source: string;
  status: string;
  text: string;
  sender?: string;
  position?: GeoPoint;
  confidence: number;
  updated_at: string;
  provenance: Provenance;
};

export type Hazard = {
  id: string;
  label: string;
  kind: string;
  severity: string;
  status: string;
  geometry: GeoPoint[];
  source: string;
  confidence: number;
  updated_at: string;
  provenance: Provenance;
};

export type SensorFootprint = {
  id: string;
  label: string;
  source: string;
  status: string;
  sensor_position: GeoPoint;
  frame_center: GeoPoint;
  ray: GeoPoint[];
  footprint?: GeoPoint[];
  sensor_altitude_meters?: number;
  sensor_azimuth_degrees?: number;
  sensor_elevation_degrees?: number;
  frame_center_elevation_meters?: number;
  media_ref: string;
  packet_ref: string;
  frame_time: string;
  platform_designation?: string;
  claim_posture: string;
  decoded_fields: string[];
  warnings: string[];
  confidence: number;
  updated_at: string;
  provenance: Provenance;
};

export type WeatherObservation = {
  id: string;
  label: string;
  source: string;
  status: string;
  provider: string;
  query_shape: string;
  query_geometry_wkt: string;
  position?: GeoPoint;
  valid_time: string;
  model_time?: string;
  fresh_until?: string;
  variable: string;
  value: number;
  unit?: string;
  claim_posture: string;
  confidence: number;
  updated_at: string;
  provenance: Provenance;
};

export type Association = {
  id: string;
  label: string;
  kind: string;
  source: string;
  status: string;
  primary_track_id: string;
  candidate_track_id: string;
  algorithm: string;
  reason: string;
  distance_meters?: number;
  time_delta_seconds?: number;
  claim_posture: string;
  confidence: number;
  updated_at: string;
  provenance: Provenance;
  operator_review?: AssociationReview;
};

export type AssociationReviewDecision = 'acknowledged' | 'challenged';

export type AssociationReview = {
  association_id: string;
  decision: AssociationReviewDecision;
  reviewed_by: string;
  reviewed_at: string;
  reviewer_role: string;
  authority_scope: string;
  conflict_policy: string;
  comment?: string;
};

export type CompanionFleet = {
  id: string;
  label: string;
  source: string;
  status: string;
  evidence_kind: string;
  vehicle_profile: string;
  node_count: number;
  vehicle_count: number;
  expected_summaries?: number;
  assertion_state: string;
  no_transmit_posture: string;
  raw_mavlink_excluded: boolean;
  raw_mavlink_policy?: string;
  demo_evidence_label: string;
  confidence: number;
  updated_at: string;
  nodes: CompanionNode[];
  readback: CompanionReadback;
  command_posture: CompanionCommandPosture;
  raw_mavlink_exclusion: CompanionRawMAVLinkExclusion;
  assertions: CompanionAssertion[];
  provenance: Provenance;
};

export type CompanionNode = {
  id: string;
  vehicle_count: number;
  peer_count: number;
  initial_summary_count?: number;
  final_summary_count?: number;
  watermark_count?: number;
  applied_diff_count?: number;
  diff_item_count?: number;
  ttl_merge_posture?: string;
};

export type CompanionReadback = {
  adapter_status: string;
  accepted: boolean;
  received_requests: number;
  correlation_id?: string;
  native_execution_allowed: boolean;
  companion_transmit_allowed: boolean;
  command_ack_status: string;
  command_ack_result?: string;
  command_ack_observed_at?: string;
  result_status: string;
  result_observed_property?: string;
  result_observed_at?: string;
};

export type CompanionCommandPosture = {
  status: string;
  hardware_blocked: boolean;
  simulator_only: boolean;
  preflight_accepted: boolean;
  ack_accepted: boolean;
  post_state_observed: boolean;
  hardware_transmit_authorized: boolean;
};

export type CompanionRawMAVLinkExclusion = {
  rejected_by_summary_index: boolean;
  policy?: string;
  error?: string;
};

export type CompanionAssertion = {
  name: string;
  passed: boolean;
  detail?: string;
};

export type Alert = {
  id: string;
  label: string;
  severity: string;
  status: string;
  entity_id: string;
  reason: string;
  updated_at: string;
};

export type GeoPoint = {
  lat: number;
  lon: number;
};

export type Provenance = {
  owner: string;
  source_ref: string;
  observed_at: string;
};

export type EntityRef =
  | { kind: 'track'; id: string }
  | { kind: 'asset'; id: string }
  | { kind: 'task'; id: string }
  | { kind: 'advisory'; id: string }
  | { kind: 'hazard'; id: string }
  | { kind: 'sensor-footprint'; id: string }
  | { kind: 'weather-observation'; id: string }
  | { kind: 'association'; id: string }
  | { kind: 'companion-fleet'; id: string }
  | { kind: 'alert'; id: string };
