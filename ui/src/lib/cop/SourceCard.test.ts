import { render } from 'svelte/server';
import { describe, expect, it } from 'vitest';
import SourceCard from './SourceCard.svelte';
import type { FeedRow } from './sourceHealth';

describe('SourceCard', () => {
  it('renders source message, runtime flow, and discovery counts together', () => {
    const feed: FeedRow = {
      id: 'feed.adsb',
      name: 'ADS-B',
      kind: 'air-picture',
      status: 'live',
      last_event_at: '2026-06-21T16:20:00Z',
      message: 'OpenSky-compatible fixture via SemStreams component flow',
      runtime: {
        id: 'feed.adsb',
        name: 'ADS-B',
        status: 'flowing',
        message: 'component flow active',
        healthy_components: 3,
        total_components: 3,
        messages_per_second: 4.5,
        last_activity: '2026-06-21T16:20:59Z',
        last_activity_age_seconds: 1
      }
    };

    const { body } = render(SourceCard, {
      props: {
        feed,
        diagnostics: [
          {
            org: 'c360',
            platform: 'edge-compose',
            source: 'adsb',
            family: 'adsb',
            entity_type: 'track',
            prefix: 'c360.edge-compose.cop.adsb.track',
            count: 1,
            limit: 500,
            at_limit: false
          }
        ]
      }
    });

    expect(body).toContain('aria-label="ADS-B source state"');
    expect(body).toContain('flowing');
    expect(body).toContain('OpenSky-compatible fixture via SemStreams component flow');
    expect(body).toContain('4.5 msg/s');
    expect(body).toContain('3/3 healthy');
    expect(body).toContain('track 1');
  });

  it('marks stale runtime state and truncation pressure', () => {
    const feed: FeedRow = {
      id: 'feed.cap',
      name: 'CAP',
      kind: 'advisory',
      status: 'planned',
      last_event_at: '2026-06-21T16:20:00Z',
      message: 'Schema/sample gate pending',
      runtime: {
        id: 'feed.cap',
        name: 'CAP',
        status: 'stale',
        message: 'source reports stale data',
        healthy_components: 2,
        total_components: 3,
        messages_per_second: 0
      }
    };

    const { body } = render(SourceCard, {
      props: {
        feed,
        diagnostics: [
          {
            org: 'c360',
            platform: 'edge-compose',
            source: 'cap',
            family: 'cap',
            entity_type: 'hazard_area',
            prefix: 'c360.edge-compose.cop.cap.hazard_area',
            count: 500,
            limit: 500,
            at_limit: true
          }
        ]
      }
    });

    expect(body).toContain('class="feed-card stale"');
    expect(body).toContain('0 msg/s');
    expect(body).toContain('2/3 healthy');
    expect(body).toContain('no flow');
    expect(body).toContain('hazard area 500+');
    expect(body).toContain('class="at-limit"');
  });
});
