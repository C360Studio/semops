<script lang="ts">
  import { onMount } from 'svelte';
  import { Activity, AlertTriangle, Database, RefreshCcw } from '@lucide/svelte';
  import { loadSnapshot, freshnessLabel } from '$lib/cop/client';
  import TacticalMap from '$lib/cop/TacticalMap.svelte';
  import type { Alert, Asset, EntityRef, Hazard, Snapshot, Track } from '$lib/cop/types';

  let snapshot = $state<Snapshot | null>(null);
  let source = $state<'api' | 'fixture'>('fixture');
  let error = $state<string | undefined>();
  let selected = $state<EntityRef>({ kind: 'track', id: 'c360.edge.cop.mavlink.track.system-42' });
  let loading = $state(true);

  const selectedEntity = $derived(resolveSelected(snapshot, selected));
  async function refresh() {
    loading = true;
    const result = await loadSnapshot();
    snapshot = result.snapshot;
    source = result.source;
    error = result.error;
    if (!resolveSelected(snapshot, selected) && snapshot.tracks[0]) {
      selected = { kind: 'track', id: snapshot.tracks[0].id };
    }
    loading = false;
  }

  onMount(() => {
    void refresh();
  });

  function selectEntity(kind: EntityRef['kind'], id: string) {
    selected = { kind, id } as EntityRef;
  }

  function entityTitle(entity: Track | Asset | Hazard | Alert | undefined) {
    if (!entity) return 'No selection';
    return entity.label;
  }

  function resolveSelected(snapshot: Snapshot | null, selected: EntityRef): Track | Asset | Hazard | Alert | undefined {
    if (!snapshot) return undefined;
    if (selected.kind === 'track') return snapshot.tracks.find((track) => track.id === selected.id);
    if (selected.kind === 'asset') return snapshot.assets.find((asset) => asset.id === selected.id);
    if (selected.kind === 'hazard') return snapshot.hazards.find((hazard) => hazard.id === selected.id);
    return snapshot.alerts.find((alert) => alert.id === selected.id);
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
          <dt>Alerts</dt>
          <dd>{snapshot.summary.active_alerts}</dd>
        </div>
        <div>
          <dt>Feeds</dt>
          <dd>{snapshot.feeds.length}</dd>
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
          {#each snapshot.feeds as feed}
            <article class="feed-card" class:live={feed.status === 'live'}>
              <strong>{feed.name}</strong>
              <span>{feed.status}</span>
              <small>{feed.message}</small>
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

{#snippet entityInspector(entity: Track | Asset | Hazard | Alert)}
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

  {#if 'reason' in entity}
    <p class="reason">{entity.reason}</p>
  {/if}
{/snippet}
