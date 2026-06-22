<script lang="ts">
  import { Activity } from '@lucide/svelte';
  import { freshnessLabel, formatRate } from './client';
  import { entityTypeLabel, runtimeTone, type FeedRow } from './sourceHealth';
  import type { DiscoveryDiagnostic } from './types';

  let {
    feed,
    diagnostics = []
  }: {
    feed: FeedRow;
    diagnostics?: DiscoveryDiagnostic[];
  } = $props();

  const runtimeFeed = $derived(feed.runtime);
  const tone = $derived(runtimeTone(feed));
</script>

<article
  class="feed-card"
  aria-label={`${feed.name} source state`}
  class:live={feed.status === 'live'}
  class:flowing={tone === 'flowing'}
  class:idle={tone === 'idle'}
  class:stale={tone === 'stale'}
  class:degraded={tone === 'degraded'}
>
  <strong>{feed.name}</strong>
  <span>{runtimeFeed?.status ?? feed.status}</span>
  <small>{feed.message}</small>
  {#if runtimeFeed}
    <div class="flow-metrics" aria-label={`${feed.name} runtime flow`}>
      <span><Activity size={13} /> {formatRate(runtimeFeed.messages_per_second)} msg/s</span>
      <span>{runtimeFeed.healthy_components}/{runtimeFeed.total_components} healthy</span>
      <span>{runtimeFeed.last_activity ? `${freshnessLabel(runtimeFeed.last_activity)} flow` : 'no flow'}</span>
    </div>
  {/if}
  {#if diagnostics.length}
    <div class="index-counts" aria-label={`${feed.name} discovery counts`}>
      {#each diagnostics as item}
        <span class:at-limit={item.at_limit} title={item.prefix}>
          {entityTypeLabel(item.entity_type)} {item.count}{item.at_limit ? '+' : ''}
        </span>
      {/each}
    </div>
  {/if}
</article>
