import { fixtureSnapshot } from './fixture';
import type { Snapshot } from './types';

export type SnapshotLoadResult = {
  snapshot: Snapshot;
  source: 'api' | 'fixture';
  error?: string;
};

export async function loadSnapshot(fetcher: typeof fetch = fetch): Promise<SnapshotLoadResult> {
  try {
    const response = await fetcher('/api/cop/snapshot', {
      headers: { accept: 'application/json' }
    });
    if (!response.ok) {
      throw new Error(`snapshot request failed: ${response.status}`);
    }
    const snapshot = (await response.json()) as Snapshot;
    return { snapshot, source: 'api' };
  } catch (error) {
    return {
      snapshot: fixtureSnapshot,
      source: 'fixture',
      error: error instanceof Error ? error.message : 'snapshot unavailable'
    };
  }
}

export function freshnessLabel(isoTime: string, now = new Date()): string {
  const time = Date.parse(isoTime);
  if (Number.isNaN(time)) {
    return 'unknown';
  }
  const ageSeconds = Math.max(0, Math.round((now.getTime() - time) / 1000));
  if (ageSeconds < 60) {
    return `${ageSeconds}s`;
  }
  const ageMinutes = Math.round(ageSeconds / 60);
  if (ageMinutes < 60) {
    return `${ageMinutes}m`;
  }
  return `${Math.round(ageMinutes / 60)}h`;
}
