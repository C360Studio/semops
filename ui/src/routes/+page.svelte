<script lang="ts">
  import { onMount } from 'svelte';
  import { Activity, AlertTriangle, Database, RefreshCcw } from '@lucide/svelte';
  import { loadRuntime, loadSnapshot, freshnessLabel, formatRate } from '$lib/cop/client';
  import { reconcileSelection, resolveEntity, type SelectableEntity } from '$lib/cop/selection';
  import TacticalMap from '$lib/cop/TacticalMap.svelte';
  import type {
    Advisory,
    Alert,
    Asset,
    DiscoveryDiagnostic,
    EntityRef,
    FeedHealth,
    Hazard,
    RuntimeFeed,
    RuntimeSnapshot,
    Snapshot,
    Task,
    Track
  } from '$lib/cop/types';

  type FeedRow = FeedHealth & { runtime?: RuntimeFeed };

  let snapshot = $state<Snapshot | null>(null);
  let runtime = $state<RuntimeSnapshot | null>(null);
  let source = $state<'api' | 'fixture'>('fixture');
  let error = $state<string | undefined>();
  let selected = $state<EntityRef>({ kind: 'track', id: 'c360.edge.cop.mavlink.track.system-42' });
  let loading = $state(true);

  const selectedEntity = $derived(resolveEntity(snapshot, selected));
  const feedRows = $derived(buildFeedRows(snapshot, runtime));

  async function refresh() {
    loading = true;
    const [snapshotResult, runtimeResult] = await Promise.all([loadSnapshot(), loadRuntime()]);
    snapshot = snapshotResult.snapshot;
    runtime = runtimeResult.runtime;
    source = snapshotResult.source;
    error = [snapshotResult.error, runtimeResult.error].filter(Boolean).join('; ') || undefined;
    selected = reconcileSelection(snapshot, selected);
    loading = false;
  }

  onMount(() => {
    void refresh();
  });

  function selectEntity(kind: EntityRef['kind'], id: string) {
    selected = { kind, id } as EntityRef;
  }

  function entityTitle(entity: SelectableEntity | undefined) {
    if (!entity) return 'No selection';
    return entity.label;
  }

  function discoveryDiagnosticsForFeed(snapshot: Snapshot, feedID: string): DiscoveryDiagnostic[] {
    const sourceByFeed: Record<string, string> = {
      'feed.mavlink': 'mavlink',
      'feed.tak': 'tak',
      'feed.cap': 'cap',
      'feed.adsb': 'adsb',
      'feed.sapient': 'sapient'
    };
    const source = sourceByFeed[feedID];
    if (!source) return [];
    return (snapshot.diagnostics?.discovery ?? []).filter((item) => item.source === source);
  }

  function buildFeedRows(snapshot: Snapshot | null, runtime: RuntimeSnapshot | null): FeedRow[] {
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

  function entityTypeLabel(value: string) {
    return value.replaceAll('_', ' ');
  }

</script>

<svelte:head>
  <title>SemOps COP</title>
</svelte:head>

<main class="app-shell">
  <header class="topbar">
    <section>
      <span class="eyebrow">SemOps COP</span>
      <h1>Common Operating Picture</h1>
    </section>
    {#if snapshot}
      <dl class="summary">
        <div>
          <dt>Tracks</dt>
          <dd>{snapshot.summary.active_tracks}</dd>
        </div>
        <div>
          <dt>Tasks</dt>
          <dd>{snapshot.summary.active_tasks}</dd>
        </div>
        <div>
          <dt>Msgs</dt>
          <dd>{snapshot.summary.active_advisories}</dd>
        </div>
        <div>
          <dt>Feeds</dt>
          <dd>{feedRows.length}</dd>
        </div>
      </dl>
    {/if}
    <button class="icon-button" type="button" onclick={refresh} aria-label="Refresh COP snapshot" title="Refresh COP snapshot">
      <RefreshCcw size={18} />
    </button>
  </header>

  {#if snapshot}
    <section class="workspace" aria-label="COP workspace">
      <section class="map-panel" aria-label="Tactical map">
        <div class="map-toolbar">
          <span class:live={source === 'api'} class="source-badge">
            <Database size={14} />
            {source}
          </span>
          {#if loading}
            <span class="muted">Refreshing</span>
          {:else}
            <span class="muted">{freshnessLabel(snapshot.generated_at)} snapshot</span>
          {/if}
        </div>

        <TacticalMap {snapshot} {selected} onSelect={(next) => selectEntity(next.kind, next.id)} />
      </section>

      <aside class="side-panel" aria-label="Entity inspector">
        <div class="panel-header">
          <span class="eyebrow">Selected</span>
          <h2>{entityTitle(selectedEntity)}</h2>
        </div>
        {#if selectedEntity}
          {@render entityInspector(selectedEntity)}
        {/if}
      </aside>
    </section>

    <section class="lower-grid" aria-label="Feed and alert state">
      <section class="feed-strip">
        <h2>Sources</h2>
        <div class="feed-list">
          {#each feedRows as feed}
            {@const runtimeFeed = feed.runtime}
            <article
              class="feed-card"
              class:live={feed.status === 'live'}
              class:flowing={runtimeFeed?.status === 'flowing'}
              class:idle={runtimeFeed?.status === 'idle'}
              class:stale={runtimeFeed?.status === 'stale'}
              class:degraded={runtimeFeed?.status === 'degraded'}
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
              {#if discoveryDiagnosticsForFeed(snapshot, feed.id).length}
                <div class="index-counts" aria-label={`${feed.name} discovery counts`}>
                  {#each discoveryDiagnosticsForFeed(snapshot, feed.id) as item}
                    <span class:at-limit={item.at_limit} title={item.prefix}>
                      {entityTypeLabel(item.entity_type)} {item.count}{item.at_limit ? '+' : ''}
                    </span>
                  {/each}
                </div>
              {/if}
            </article>
          {/each}
        </div>
      </section>

      <section class="alert-strip">
        <h2>Alerts</h2>
        {#each snapshot.alerts as alert}
          <button
            class:selected={selected.kind === 'alert' && selected.id === alert.id}
            class="alert-row"
            type="button"
            aria-pressed={selected.kind === 'alert' && selected.id === alert.id}
            onclick={() => selectEntity('alert', alert.id)}
          >
            <AlertTriangle size={16} />
            <span>{alert.label}</span>
            <small>{alert.severity}</small>
          </button>
        {/each}
      </section>
    </section>
  {:else}
    <section class="loading-state">
      <Activity size={28} />
    </section>
  {/if}

  {#if error}
    <p class="status-note">{error}</p>
  {/if}
</main>

{#snippet entityInspector(entity: Track | Asset | Task | Advisory | Hazard | Alert)}
  <div class="inspector-grid">
    {#if 'source' in entity}
      <div>
        <span>Source</span>
        <strong>{entity.source}</strong>
      </div>
    {/if}
    {#if 'status' in entity}
      <div>
        <span>Status</span>
        <strong>{entity.status}</strong>
      </div>
    {/if}
    {#if 'confidence' in entity}
      <div>
        <span>Confidence</span>
        <strong>{Math.round(entity.confidence * 100)}%</strong>
      </div>
    {/if}
    {#if 'updated_at' in entity}
      <div>
        <span>Freshness</span>
        <strong>{freshnessLabel(entity.updated_at)}</strong>
      </div>
    {/if}
  </div>

  {#if 'position' in entity && entity.position}
    <dl class="detail-list">
      <div>
        <dt>Latitude</dt>
        <dd>{entity.position.lat.toFixed(5)}</dd>
      </div>
      <div>
        <dt>Longitude</dt>
        <dd>{entity.position.lon.toFixed(5)}</dd>
      </div>
    </dl>
  {/if}

  {#if 'provenance' in entity}
    <section class="provenance">
      <h3>Provenance</h3>
      <dl class="detail-list">
        <div>
          <dt>Owner</dt>
          <dd>{entity.provenance.owner}</dd>
        </div>
        <div>
          <dt>Source ref</dt>
          <dd>{entity.provenance.source_ref}</dd>
        </div>
      </dl>
    </section>
  {/if}

  {#if 'description' in entity && entity.description}
    <p class="reason">{entity.description}</p>
  {/if}

  {#if 'text' in entity}
    <p class="reason">{entity.text}</p>
  {/if}

  {#if 'reason' in entity}
    <p class="reason">{entity.reason}</p>
  {/if}
{/snippet}
