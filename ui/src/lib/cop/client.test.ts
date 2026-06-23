import { describe, expect, it } from 'vitest';
import { loadRuntime, loadSnapshot, freshnessLabel, formatRate } from './client';
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
