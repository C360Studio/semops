<script lang="ts">
  import { onMount } from 'svelte';
  import { Activity, AlertTriangle, Crosshair, Database, MapPinned, RefreshCcw, ShieldCheck } from '@lucide/svelte';
  import { loadSnapshot, freshnessLabel } from '$lib/cop/client';
  import type { Alert, Asset, EntityRef, GeoPoint, Hazard, Snapshot, Track } from '$lib/cop/types';

  let snapshot = $state<Snapshot | null>(null);
  let source = $state<'api' | 'fixture'>('fixture');
  let error = $state<string | undefined>();
  let selected = $state<EntityRef>({ kind: 'track', id: 'c360.edge.cop.mavlink.track.system-42' });
  let loading = $state(true);

  const selectedEntity = $derived(resolveSelected(snapshot, selected));
  const mapBounds = $derived(boundsFor(snapshot));

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

  function pointStyle(point: GeoPoint) {
    const bounds = mapBounds;
    const x = ((point.lon - bounds.minLon) / Math.max(0.0001, bounds.maxLon - bounds.minLon)) * 100;
    const y = (1 - (point.lat - bounds.minLat) / Math.max(0.0001, bounds.maxLat - bounds.minLat)) * 100;
    return `left:${Math.min(94, Math.max(6, x))}%;top:${Math.min(92, Math.max(8, y))}%`;
  }

  function polygonPoints(hazard: Hazard) {
    return hazard.geometry
      .map((point) => {
        const bounds = mapBounds;
        const x = ((point.lon - bounds.minLon) / Math.max(0.0001, bounds.maxLon - bounds.minLon)) * 100;
        const y = (1 - (point.lat - bounds.minLat) / Math.max(0.0001, bounds.maxLat - bounds.minLat)) * 100;
        return `${Math.min(96, Math.max(4, x))},${Math.min(94, Math.max(6, y))}`;
      })
      .join(' ');
  }

  function hazardCentroid(hazard: Hazard): GeoPoint {
    if (hazard.geometry.length === 0) {
      return { lat: 0, lon: 0 };
    }
    const sum = hazard.geometry.reduce(
      (next, point) => ({ lat: next.lat + point.lat, lon: next.lon + point.lon }),
      { lat: 0, lon: 0 }
    );
    return { lat: sum.lat / hazard.geometry.length, lon: sum.lon / hazard.geometry.length };
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

  function boundsFor(snapshot: Snapshot | null) {
    const points = [
      ...(snapshot?.tracks.map((track) => track.position) ?? []),
      ...(snapshot?.assets.map((asset) => asset.position).filter(isGeoPoint) ?? []),
      ...(snapshot?.hazards.flatMap((hazard) => hazard.geometry) ?? [])
    ];
    if (points.length === 0) {
      return { minLat: 38.88, maxLat: 38.92, minLon: -77.03, maxLon: -76.98 };
    }
    const lats = points.map((point) => point.lat);
    const lons = points.map((point) => point.lon);
    return {
      minLat: Math.min(...lats) - 0.006,
      maxLat: Math.max(...lats) + 0.006,
      minLon: Math.min(...lons) - 0.008,
      maxLon: Math.max(...lons) + 0.008
    };
  }

  function isGeoPoint(point: GeoPoint | undefined): point is GeoPoint {
    return point !== undefined;
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

        <div class="map-surface">
          <svg class="hazard-layer" viewBox="0 0 100 100" preserveAspectRatio="none" aria-hidden="true">
            {#each snapshot.hazards as hazard}
              <polygon
                class:selected={selected.kind === 'hazard' && selected.id === hazard.id}
                points={polygonPoints(hazard)}
              />
            {/each}
          </svg>

          {#each snapshot.hazards as hazard}
            <button
              class:selected={selected.kind === 'hazard' && selected.id === hazard.id}
              class="map-marker hazard"
              style={pointStyle(hazardCentroid(hazard))}
              type="button"
              aria-label={`Select ${hazard.label}`}
              title={hazard.label}
              onclick={() => selectEntity('hazard', hazard.id)}
            >
              <MapPinned size={16} />
            </button>
          {/each}

          {#each snapshot.assets as asset}
            {#if asset.position}
              <button
                class:selected={selected.kind === 'asset' && selected.id === asset.id}
                class="map-marker asset"
                style={pointStyle(asset.position)}
                type="button"
                aria-label={`Select ${asset.label}`}
                title={asset.label}
                onclick={() => selectEntity('asset', asset.id)}
              >
                <ShieldCheck size={16} />
              </button>
            {/if}
          {/each}

          {#each snapshot.tracks as track}
            <button
              class:selected={selected.kind === 'track' && selected.id === track.id}
              class="map-marker track"
              style={pointStyle(track.position)}
              type="button"
              aria-label={`Select ${track.label}`}
              title={track.label}
              onclick={() => selectEntity('track', track.id)}
            >
              <Crosshair size={18} />
            </button>
          {/each}
        </div>
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
