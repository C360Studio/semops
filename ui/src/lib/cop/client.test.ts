import { describe, expect, it } from 'vitest';
import { loadSnapshot, freshnessLabel } from './client';
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
    expect(result.snapshot.tracks).toHaveLength(2);
    expect(result.snapshot.tasks).toHaveLength(1);
  });
});

describe('freshnessLabel', () => {
  it('formats recent observations', () => {
    expect(freshnessLabel('2026-06-19T11:59:40Z', new Date('2026-06-19T12:00:00Z'))).toBe('20s');
    expect(freshnessLabel('2026-06-19T11:40:00Z', new Date('2026-06-19T12:00:00Z'))).toBe('20m');
  });
});
