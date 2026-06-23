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
  alerts: Alert[];
};

export type Summary = {
  active_tracks: number;
  active_tasks: number;
  active_advisories: number;
  active_sensor_footprints: number;
  active_weather_observations: number;
  active_associations: number;
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
  | { kind: 'alert'; id: string };
