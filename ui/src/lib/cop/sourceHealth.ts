import type { DiscoveryDiagnostic, FeedHealth, RuntimeFeed, RuntimeSnapshot, Snapshot } from './types';

export type FeedRow = FeedHealth & { runtime?: RuntimeFeed };

const sourceByFeed: Record<string, string> = {
  'feed.mavlink': 'mavlink',
  'feed.tak': 'tak',
  'feed.command': 'command',
  'feed.cap': 'cap',
  'feed.adsb': 'adsb',
  'feed.klv': 'klv',
  'feed.weather': 'weather',
  'feed.sapient': 'sapient'
};

export function buildFeedRows(snapshot: Snapshot | null, runtime: RuntimeSnapshot | null): FeedRow[] {
  if (!snapshot) return [];
  const runtimeByFeed = new Map((runtime?.feeds ?? []).map((feed) => [feed.id, feed]));
  const rows = snapshot.feeds.map((feed) => ({
    ...feed,
    runtime: runtimeByFeed.get(feed.id)
  }));
  const knownFeeds = new Set(rows.map((feed) => feed.id));
  const generatedAt = runtime?.generated_at ?? snapshot.generated_at;
  for (const runtimeFeed of runtime?.feeds ?? []) {
    if (knownFeeds.has(runtimeFeed.id)) continue;
    rows.push({
      id: runtimeFeed.id,
      name: runtimeFeed.name,
      kind: 'component-flow',
      status: runtimeFeed.status,
      last_event_at: runtimeFeed.last_activity ?? generatedAt,
      message: runtimeFeed.message,
      runtime: runtimeFeed
    });
  }
  return rows;
}

export function discoveryDiagnosticsForFeed(snapshot: Snapshot, feedID: string): DiscoveryDiagnostic[] {
  const source = sourceByFeed[feedID];
  if (!source) return [];
  return (snapshot.diagnostics?.discovery ?? []).filter((item) => item.source === source);
}

export function entityTypeLabel(value: string): string {
  return value.replaceAll('_', ' ');
}

export function runtimeTone(feed: FeedRow): 'flowing' | 'idle' | 'stale' | 'degraded' | undefined {
  const status = feed.runtime?.status;
  if (status === 'flowing' || status === 'idle' || status === 'stale' || status === 'degraded') {
    return status;
  }
  return undefined;
}
