<script lang="ts">
  import { onMount } from 'svelte';
  import { Activity, AlertTriangle, Database, RefreshCcw } from '@lucide/svelte';
  import { loadRuntime, loadSnapshot, freshnessLabel } from '$lib/cop/client';
  import { reconcileSelection, resolveEntity, resolveMapSelection, type SelectableEntity } from '$lib/cop/selection';
  import SourceCard from '$lib/cop/SourceCard.svelte';
  import TacticalMap from '$lib/cop/TacticalMap.svelte';
  import { buildFeedRows, discoveryDiagnosticsForFeed } from '$lib/cop/sourceHealth';
  import type {
    Advisory,
    Alert,
    Asset,
    EntityRef,
    Hazard,
    RuntimeSnapshot,
    SensorFootprint,
    Snapshot,
    Task,
    Track,
    WeatherObservation
  } from '$lib/cop/types';

  let snapshot = $state<Snapshot | null>(null);
  let runtime = $state<RuntimeSnapshot | null>(null);
  let source = $state<'api' | 'fixture'>('fixture');
  let error = $state<string | undefined>();
  let selected = $state<EntityRef>({ kind: 'track', id: 'c360.edge.cop.mavlink.track.system-42' });
  let loading = $state(true);

  const selectedEntity = $derived(resolveEntity(snapshot, selected));
  const mapSelected = $derived(resolveMapSelection(snapshot, selected) ?? selected);
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

  function formatWeatherValue(entity: WeatherObservation) {
    const value = Number.isFinite(entity.value) ? entity.value.toFixed(1).replace(/\.0$/, '') : 'unknown';
    return entity.unit ? `${value} ${entity.unit}` : value;
  }

  function formatInstant(isoTime: string | undefined) {
    if (!isoTime) return 'unknown';
    return isoTime.replace('T', ' ').replace('Z', 'Z');
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
          <dt>Weather</dt>
          <dd>{snapshot.summary.active_weather_observations}</dd>
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

        <TacticalMap {snapshot} selected={mapSelected} onSelect={(next) => selectEntity(next.kind, next.id)} />
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
            <SourceCard {feed} diagnostics={discoveryDiagnosticsForFeed(snapshot, feed.id)} />
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

{#snippet entityInspector(entity: Track | Asset | Task | Advisory | Hazard | SensorFootprint | WeatherObservation | Alert)}
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

  {#if 'sensor_position' in entity}
    <section class="provenance">
      <h3>KLV Evidence</h3>
      <dl class="detail-list">
        {#if entity.platform_designation}
          <div>
            <dt>Platform</dt>
            <dd>{entity.platform_designation}</dd>
          </div>
        {/if}
        <div>
          <dt>Sensor point</dt>
          <dd>{entity.sensor_position.lat.toFixed(5)}, {entity.sensor_position.lon.toFixed(5)}</dd>
        </div>
        <div>
          <dt>Frame center</dt>
          <dd>{entity.frame_center.lat.toFixed(5)}, {entity.frame_center.lon.toFixed(5)}</dd>
        </div>
        {#if entity.sensor_azimuth_degrees !== undefined}
          <div>
            <dt>Azimuth</dt>
            <dd>{entity.sensor_azimuth_degrees.toFixed(2)} deg</dd>
          </div>
        {/if}
        {#if entity.sensor_elevation_degrees !== undefined}
          <div>
            <dt>Elevation</dt>
            <dd>{entity.sensor_elevation_degrees.toFixed(2)} deg</dd>
          </div>
        {/if}
        <div>
          <dt>Frame time</dt>
          <dd>{freshnessLabel(entity.frame_time)}</dd>
        </div>
        <div>
          <dt>Media ref</dt>
          <dd>{entity.media_ref}</dd>
        </div>
        <div>
          <dt>Packet ref</dt>
          <dd>{entity.packet_ref}</dd>
        </div>
        <div>
          <dt>Decoded fields</dt>
          <dd>{entity.decoded_fields.join(', ')}</dd>
        </div>
      </dl>
      <p class="reason">{entity.claim_posture}</p>
      {#if entity.warnings.length > 0}
        <p class="reason">{entity.warnings.join('; ')}</p>
      {/if}
    </section>
  {/if}

  {#if 'variable' in entity}
    <section class="provenance">
      <h3>Weather Evidence</h3>
      <dl class="detail-list">
        <div>
          <dt>Provider</dt>
          <dd>{entity.provider || 'unknown'}</dd>
        </div>
        <div>
          <dt>Variable</dt>
          <dd>{entity.variable}</dd>
        </div>
        <div>
          <dt>Value</dt>
          <dd>{formatWeatherValue(entity)}</dd>
        </div>
        <div>
          <dt>Query shape</dt>
          <dd>{entity.query_shape || 'unknown'}</dd>
        </div>
        <div>
          <dt>Valid time</dt>
          <dd>{formatInstant(entity.valid_time)}</dd>
        </div>
        <div>
          <dt>Model time</dt>
          <dd>{formatInstant(entity.model_time)}</dd>
        </div>
        <div>
          <dt>Fresh until</dt>
          <dd>{formatInstant(entity.fresh_until)}</dd>
        </div>
        <div>
          <dt>Query geometry</dt>
          <dd>{entity.query_geometry_wkt || 'unknown'}</dd>
        </div>
      </dl>
      <p class="reason">{entity.claim_posture}</p>
    </section>
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
    <dl class="detail-list">
      <div>
        <dt>Target</dt>
        <dd>{entity.entity_id}</dd>
      </div>
    </dl>
    <p class="reason">{entity.reason}</p>
  {/if}
{/snippet}
