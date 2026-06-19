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
  alerts: Alert[];
};

export type Summary = {
  active_tracks: number;
  active_tasks: number;
  active_advisories: number;
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
  | { kind: 'alert'; id: string };
