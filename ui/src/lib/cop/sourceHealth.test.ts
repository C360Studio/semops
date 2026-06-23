import { describe, expect, it } from 'vitest';
import { fixtureSnapshot } from './fixture';
import { buildFeedRows, discoveryDiagnosticsForFeed, entityTypeLabel, runtimeTone } from './sourceHealth';
import type { RuntimeSnapshot, Snapshot } from './types';

const runtime: RuntimeSnapshot = {
  generated_at: '2026-06-21T16:21:00Z',
  feeds: [
    {
      id: 'feed.mavlink',
      name: 'MAVLink',
      status: 'flowing',
      message: 'component flow active',
      healthy_components: 3,
      total_components: 3,
      messages_per_second: 8.5,
      last_activity: '2026-06-21T16:20:58Z'
    },
    {
      id: 'feed.sapient',
      name: 'SAPIENT',
      status: 'idle',
      message: 'components healthy; no recent flow',
      healthy_components: 2,
      total_components: 2,
      messages_per_second: 0
    }
  ],
  components: []
};

describe('buildFeedRows', () => {
  it('merges runtime data without replacing source messages', () => {
    const rows = buildFeedRows(fixtureSnapshot, runtime);
    const mavlink = rows.find((feed) => feed.id === 'feed.mavlink');

    expect(mavlink?.runtime?.status).toBe('flowing');
    expect(mavlink?.message).toBe('Generated heartbeat and position smoke path');
    expect(runtimeTone(mavlink!)).toBe('flowing');
  });

  it('adds runtime-only preflight feeds as component-flow evidence', () => {
    const rows = buildFeedRows(fixtureSnapshot, runtime);
    const sapient = rows.find((feed) => feed.id === 'feed.sapient');

    expect(sapient).toMatchObject({
      id: 'feed.sapient',
      name: 'SAPIENT',
      kind: 'component-flow',
      status: 'idle',
      message: 'components healthy; no recent flow'
    });
    expect(sapient?.last_event_at).toBe(runtime.generated_at);
    expect(runtimeTone(sapient!)).toBe('idle');
  });

  it('handles missing runtime as a snapshot-only source list', () => {
    const rows = buildFeedRows(fixtureSnapshot, null);

    expect(rows).toHaveLength(fixtureSnapshot.feeds.length);
    expect(rows.some((feed) => feed.runtime)).toBe(false);
  });
});

describe('discoveryDiagnosticsForFeed', () => {
  it('filters diagnostics by feed source and preserves truncation evidence', () => {
    const snapshot: Snapshot = {
      ...fixtureSnapshot,
      diagnostics: {
        discovery: [
          {
            org: 'c360',
            platform: 'edge',
            source: 'adsb',
            family: 'adsb',
            entity_type: 'track',
            prefix: 'c360.edge.cop.adsb.track',
            count: 500,
            limit: 500,
            at_limit: true
          },
          {
            org: 'c360',
            platform: 'edge',
            source: 'weather',
            family: 'weather',
            entity_type: 'weather_observation',
            prefix: 'c360.edge.cop.weather.weather_observation',
            count: 1,
            limit: 500,
            at_limit: false
          },
          {
            org: 'c360',
            platform: 'edge',
            source: 'mavlink',
            family: 'mavlink',
            entity_type: 'asset',
            prefix: 'c360.edge.cop.mavlink.asset',
            count: 1,
            limit: 500,
            at_limit: false
          },
          {
            org: 'c360',
            platform: 'edge',
            source: 'command',
            family: 'command',
            entity_type: 'task',
            prefix: 'c360.edge.cop.command.task',
            count: 1,
            limit: 500,
            at_limit: false
          },
          {
            org: 'c360',
            platform: 'edge',
            source: 'fusion',
            family: 'fusion',
            entity_type: 'association',
            prefix: 'c360.edge.cop.fusion.association',
            count: 1,
            limit: 500,
            at_limit: false
          }
        ]
      }
    };

    expect(discoveryDiagnosticsForFeed(snapshot, 'feed.adsb')).toEqual([snapshot.diagnostics!.discovery[0]]);
    expect(discoveryDiagnosticsForFeed(snapshot, 'feed.weather')).toEqual([snapshot.diagnostics!.discovery[1]]);
    expect(discoveryDiagnosticsForFeed(snapshot, 'feed.command')).toEqual([snapshot.diagnostics!.discovery[3]]);
    expect(discoveryDiagnosticsForFeed(snapshot, 'feed.fusion')).toEqual([snapshot.diagnostics!.discovery[4]]);
    expect(discoveryDiagnosticsForFeed(snapshot, 'feed.sapient')).toEqual([]);
  });

  it('formats entity type labels for compact chips', () => {
    expect(entityTypeLabel('hazard_area')).toBe('hazard area');
  });
});
