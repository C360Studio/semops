import { describe, expect, it } from 'vitest';
import { loadRuntime, loadScenarioStatus, loadSnapshot, freshnessLabel, formatRate, reviewAssociation } from './client';
import { fixtureSnapshot } from './fixture';

describe('loadSnapshot', () => {
  it('uses API data when available', async () => {
    const result = await loadSnapshot(async () => {
      return new Response(JSON.stringify({ ...fixtureSnapshot, scenario: 'api' }), {
        status: 200,
        headers: { 'content-type': 'application/json' }
      });
    });

    expect(result.source).toBe('api');
    expect(result.snapshot.scenario).toBe('api');
  });

  it('falls back to fixtures when API is unavailable', async () => {
    const result = await loadSnapshot(async () => {
      throw new Error('offline');
    });

    expect(result.source).toBe('fixture');
    expect(result.error).toContain('offline');
    expect(result.snapshot.tracks).toHaveLength(3);
    expect(result.snapshot.tasks).toHaveLength(2);
    expect(result.snapshot.tasks[1].source).toBe('command');
    expect(result.snapshot.associations).toHaveLength(1);
    expect(result.snapshot.summary.active_associations).toBe(1);
  });
});

describe('loadRuntime', () => {
  it('uses API runtime data when available', async () => {
    const result = await loadRuntime(async () => {
      return new Response(
        JSON.stringify({
          generated_at: '2026-06-21T17:30:00Z',
          feeds: [{ id: 'feed.mavlink', name: 'MAVLink', status: 'flowing' }],
          components: []
        }),
        {
          status: 200,
          headers: { 'content-type': 'application/json' }
        }
      );
    });

    expect(result.error).toBeUndefined();
    expect(result.runtime?.feeds[0].id).toBe('feed.mavlink');
  });

  it('does not replace the COP snapshot when runtime is unavailable', async () => {
    const result = await loadRuntime(async () => {
      throw new Error('offline');
    });

    expect(result.runtime).toBeNull();
    expect(result.error).toContain('offline');
  });
});

describe('loadScenarioStatus', () => {
  it('uses same-origin scenario status when available', async () => {
    let requestedURL = '';
    const result = await loadScenarioStatus(async (url) => {
      requestedURL = url.toString();
      return new Response(
        JSON.stringify({
          scenario_id: 'phase-1-hadr-flood-evacuation',
          state: 'succeeded',
          ingress_mode: 'feed-boundary',
          completed_steps: 6,
          failed_steps: 0,
          summary: {
            mavlink_frames: 2,
            cot_events: 4,
            cap_alerts: 0,
            adsb_snapshots: 0,
            feed_boundary_deliveries: 6,
            contract_graph_mutation_attempts: 0,
            mutations: 0,
            errors: 0
          }
        }),
        {
          status: 200,
          headers: { 'content-type': 'application/json' }
        }
      );
    });

    expect(requestedURL).toBe('/scenario/status');
    expect(result.error).toBeUndefined();
    expect(result.status?.ingress_mode).toBe('feed-boundary');
    expect(result.status?.summary.feed_boundary_deliveries).toBe(6);
  });

  it('does not replace COP state when scenario status is unavailable', async () => {
    const result = await loadScenarioStatus(async () => {
      throw new Error('offline');
    });

    expect(result.status).toBeNull();
    expect(result.error).toContain('offline');
  });
});

describe('reviewAssociation', () => {
  it('posts an operator association review', async () => {
    let requestedURL = '';
    let requestedBody = '';
    let requestedHeaders = new Headers();
    const result = await reviewAssociation(
      'c360.edge.cop.fusion.association.mavlink-to-tak',
      { decision: 'acknowledged', reviewed_by: 'operator.local' },
      {
        fetcher: async (url, init) => {
          requestedURL = url.toString();
          requestedBody = init?.body?.toString() ?? '';
          requestedHeaders = new Headers(init?.headers);
          return new Response(
            JSON.stringify({
              association_id: 'c360.edge.cop.fusion.association.mavlink-to-tak',
              decision: 'acknowledged',
              reviewed_by: 'operator.local',
              reviewed_at: '2026-06-24T01:20:00Z',
              reviewer_role: 'operator.unverified',
              authority_scope: 'local.display_only',
              conflict_policy: 'latest_review_wins_display_only'
            }),
            {
              status: 200,
              headers: { 'content-type': 'application/json' }
            }
          );
        }
      }
    );

    expect(requestedURL).toContain('/api/cop/associations/c360.edge.cop.fusion.association.mavlink-to-tak/review');
    expect(JSON.parse(requestedBody)).toEqual({ decision: 'acknowledged', reviewed_by: 'operator.local' });
    expect(requestedHeaders.get('X-SemOps-Operator-ID')).toBeNull();
    expect(result.decision).toBe('acknowledged');
    expect(result.authority_scope).toBe('local.display_only');
  });

  it('sends the local operator audit header when provided', async () => {
    let requestedHeaders = new Headers();
    await reviewAssociation(
      'association-1',
      { decision: 'challenged', reviewed_by: 'operator:field-lead' },
      {
        operatorID: 'operator:field-lead',
        fetcher: async (_url, init) => {
          requestedHeaders = new Headers(init?.headers);
          return new Response(
            JSON.stringify({
              association_id: 'association-1',
              decision: 'challenged',
              reviewed_by: 'operator:field-lead',
              reviewed_at: '2026-06-24T01:20:00Z',
              reviewer_role: 'operator.unverified',
              authority_scope: 'local.display_only',
              conflict_policy: 'latest_review_wins_display_only'
            }),
            {
              status: 200,
              headers: { 'content-type': 'application/json' }
            }
          );
        }
      }
    );

    expect(requestedHeaders.get('X-SemOps-Operator-ID')).toBe('operator:field-lead');
  });
});

describe('freshnessLabel', () => {
  it('formats recent observations', () => {
    expect(freshnessLabel('2026-06-19T11:59:40Z', new Date('2026-06-19T12:00:00Z'))).toBe('20s');
    expect(freshnessLabel('2026-06-19T11:40:00Z', new Date('2026-06-19T12:00:00Z'))).toBe('20m');
  });
});

describe('formatRate', () => {
  it('keeps compact message-rate labels', () => {
    expect(formatRate(0)).toBe('0');
    expect(formatRate(2.25)).toBe('2.3');
    expect(formatRate(42.7)).toBe('43');
  });
});
