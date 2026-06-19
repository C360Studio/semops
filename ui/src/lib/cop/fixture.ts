import type { Snapshot } from './types';

const now = '2026-06-19T12:00:00Z';
const observed = '2026-06-19T11:59:42Z';
const takObserved = '2026-06-19T11:59:14Z';
const advisoryObserved = '2026-06-19T11:58:00Z';

export const fixtureSnapshot: Snapshot = {
  generated_at: now,
  scenario: 'phase-1-fixture',
  summary: {
    active_tracks: 2,
    active_tasks: 1,
    active_advisories: 1,
    active_alerts: 1,
    stale_feeds: 0
  },
  feeds: [
    {
      id: 'feed.mavlink',
      name: 'MAVLink',
      kind: 'telemetry',
      status: 'live',
      last_event_at: observed,
      message: 'Generated heartbeat and position smoke path'
    },
    {
      id: 'feed.tak',
      name: 'TAK/CoT',
      kind: 'operators',
      status: 'live',
      last_event_at: takObserved,
      message: 'Seed replay track, task, and GeoChat smoke path'
    },
    {
      id: 'feed.cap',
      name: 'CAP',
      kind: 'advisory',
      status: 'planned',
      last_event_at: '2026-06-19T11:27:00Z',
      message: 'Schema/sample gate pending'
    }
  ],
  assets: [
    {
      id: 'c360.edge.cop.mavlink.asset.system-42',
      label: 'MAVLink system 42',
      kind: 'mavlink-system',
      source: 'mavlink',
      position: { lat: 38.9001, lon: -77.0002 },
      confidence: 1,
      updated_at: observed,
      provenance: {
        owner: 'semops.feed.asset',
        source_ref: 'raw:mavlink:fixture:0001',
        observed_at: observed
      }
    }
  ],
  tracks: [
    {
      id: 'c360.edge.cop.mavlink.track.system-42',
      label: 'UAS 42',
      source: 'mavlink',
      status: 'active.armed',
      position: { lat: 38.9001, lon: -77.0002 },
      velocity: 'NED_CMPS(321 -12 7)',
      confidence: 1,
      updated_at: observed,
      provenance: {
        owner: 'semops.feed.mavlink',
        source_ref: 'raw:mavlink:fixture:0002',
        observed_at: observed
      }
    },
    {
      id: 'c360.edge.cop.tak.track.android-alpha',
      label: 'ANDROID-ALPHA',
      source: 'tak-cot',
      status: 'active.operator',
      position: { lat: 38.892, lon: -77.035 },
      velocity: '',
      confidence: 1,
      updated_at: takObserved,
      provenance: {
        owner: 'semops.feed.tak',
        source_ref: 'cot://fixture/0001',
        observed_at: takObserved
      }
    }
  ],
  tasks: [
    {
      id: 'c360.edge.cop.tak.task.marker-north-gate',
      label: 'North Gate',
      kind: 'marker',
      source: 'tak-cot',
      status: 'active.marker',
      position: { lat: 38.894, lon: -77.038 },
      description: 'checkpoint',
      confidence: 1,
      updated_at: takObserved,
      provenance: {
        owner: 'semops.feed.tak',
        source_ref: 'cot://fixture/0003',
        observed_at: takObserved
      }
    }
  ],
  advisories: [
    {
      id: 'c360.edge.cop.tak.advisory.chat-alpha-1',
      label: 'GeoChat ANDROID-ALPHA',
      kind: 'geochat',
      source: 'tak-cot',
      status: 'active.geochat',
      text: 'hold at checkpoint',
      sender: 'ANDROID-ALPHA',
      position: { lat: 38.892, lon: -77.035 },
      confidence: 1,
      updated_at: takObserved,
      provenance: {
        owner: 'semops.feed.tak',
        source_ref: 'cot://fixture/0004',
        observed_at: takObserved
      }
    }
  ],
  hazards: [
    {
      id: 'c360.edge.cop.cap.hazard_area.flood-watch-1',
      label: 'Flood watch sector',
      kind: 'flood',
      severity: 'watch',
      status: 'active',
      geometry: [
        { lat: 38.895, lon: -77.012 },
        { lat: 38.907, lon: -77.011 },
        { lat: 38.908, lon: -76.992 },
        { lat: 38.896, lon: -76.991 }
      ],
      source: 'cap',
      confidence: 0.74,
      updated_at: advisoryObserved,
      provenance: {
        owner: 'semops.feed.cap',
        source_ref: 'fixture:cap:flood-watch-1',
        observed_at: advisoryObserved
      }
    }
  ],
  alerts: [
    {
      id: 'alert.mavlink.track-freshness',
      label: 'Track freshness nominal',
      severity: 'info',
      status: 'active',
      entity_id: 'c360.edge.cop.mavlink.track.system-42',
      reason: 'MAVLink position observed within freshness window',
      updated_at: observed
    }
  ]
};
