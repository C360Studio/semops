import type { Snapshot } from './types';

const now = '2026-06-19T12:00:00Z';
const observed = '2026-06-19T11:59:42Z';
const takObserved = '2026-06-19T11:59:14Z';
const advisoryObserved = '2026-06-19T11:58:00Z';
const klvObserved = '2026-06-19T11:58:45Z';
const weatherObserved = '2026-06-19T11:57:30Z';
const commandObserved = '2026-06-19T11:59:30Z';

export const fixtureSnapshot: Snapshot = {
  generated_at: now,
  scenario: 'phase-1-fixture',
  summary: {
    active_tracks: 2,
    active_tasks: 2,
    active_advisories: 1,
    active_sensor_footprints: 1,
    active_weather_observations: 1,
    active_alerts: 1,
    stale_feeds: 0
  },
  diagnostics: {
    discovery: []
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
      id: 'feed.command',
      name: 'Command',
      kind: 'control',
      status: 'live',
      last_event_at: commandObserved,
      message: 'Synthetic command lifecycle replay state'
    },
    {
      id: 'feed.cap',
      name: 'CAP',
      kind: 'advisory',
      status: 'planned',
      last_event_at: '2026-06-19T11:27:00Z',
      message: 'Schema/sample gate pending'
    },
    {
      id: 'feed.klv',
      name: 'KLV',
      kind: 'sensor-footprint',
      status: 'live',
      last_event_at: klvObserved,
      message: 'Graph-backed KLV sensor/frame-center proof'
    },
    {
      id: 'feed.weather',
      name: 'Weather',
      kind: 'tactical-weather',
      status: 'live',
      last_event_at: weatherObserved,
      message: 'Fixture-backed point weather observation readback'
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
    },
    {
      id: 'c360.edge.cop.command.task.csapi-command-route-42',
      label: 'Route MAVLink system 42 to North Gate',
      kind: 'mavlink.goto',
      source: 'command',
      status: 'cancel_requested',
      description: 'cancel requested: airspace conflict',
      target_id: 'c360.edge.cop.mavlink.asset.system-42',
      authority: 'local.operator',
      priority: 95,
      expires_at: '2026-06-19T12:03:00Z',
      requested_by: 'operator:lead',
      correlation_id: 'ui:cancel-route-42',
      desired_state: '{"command":"cancel","target_native_id":"csapi-command-route-42","reason":"airspace conflict"}',
      confidence: 1,
      updated_at: commandObserved,
      provenance: {
        owner: 'semops.command.intent',
        source_ref: 'command://fixture/hadr-command/0004-route-cancel-requested',
        observed_at: commandObserved
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
  sensor_footprints: [
    {
      id: 'c360.edge.cop.klv.sensor_footprint.object-semops-klv-deterministic-001-ts',
      label: 'TEST-UAS-01 sensor footprint',
      source: 'klv',
      status: 'active.sensor-frame-center',
      sensor_position: { lat: 38.9022, lon: -77.0254 },
      frame_center: { lat: 38.8956, lon: -77.0108 },
      ray: [
        { lat: 38.9022, lon: -77.0254 },
        { lat: 38.8956, lon: -77.0108 }
      ],
      sensor_altitude_meters: 1250.5,
      sensor_azimuth_degrees: 90.25,
      sensor_elevation_degrees: -12.5,
      frame_center_elevation_meters: 932.2,
      media_ref: 'object://semops/klv/deterministic-001.ts',
      packet_ref: 'klv://packet/deterministic/00000001',
      frame_time: klvObserved,
      platform_designation: 'TEST-UAS-01',
      claim_posture: 'sensor-frame-center graph readback; no footprint polygon; no STANAG conformance',
      decoded_fields: [
        'media_ref',
        'packet_ref',
        'observed_at',
        'platform_designation',
        'sensor_position',
        'sensor_altitude_meters',
        'sensor_azimuth_degrees',
        'sensor_elevation_degrees',
        'frame_center',
        'frame_center_elevation_meters'
      ],
      warnings: ['footprint polygon not computed'],
      confidence: 0.82,
      updated_at: klvObserved,
      provenance: {
        owner: 'semops.feed.klv',
        source_ref: 'klv://packet/deterministic/00000001',
        observed_at: klvObserved
      }
    }
  ],
  weather_observations: [
    {
      id: 'c360.edge.cop.weather.weather_observation.open-meteo-position-fixture-temperature-2m',
      label: '29.4 degC temperature_2m',
      source: 'weather',
      status: 'fresh',
      provider: 'open-meteo',
      query_shape: 'position',
      query_geometry_wkt: 'POINT(-77.0400000 38.9000000)',
      position: { lat: 38.9, lon: -77.04 },
      valid_time: weatherObserved,
      model_time: '2026-06-19T11:00:00Z',
      fresh_until: '2026-06-19T12:57:30Z',
      variable: 'temperature_2m',
      value: 29.4,
      unit: 'degC',
      claim_posture: 'fixture-backed point observation; no live provider, weather tile, route-safety, or OGC conformance claim',
      confidence: 0.78,
      updated_at: weatherObserved,
      provenance: {
        owner: 'semops.feed.weather',
        source_ref: 'weather://open-meteo/fixture/position/temperature_2m/2026-06-19T11:57:30Z',
        observed_at: weatherObserved
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
