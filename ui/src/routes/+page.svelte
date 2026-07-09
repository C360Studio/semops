<script lang="ts">
  import { onMount } from 'svelte';
  import {
    Activity,
    AlertTriangle,
    CheckCircle2,
    CircleAlert,
    Database,
    Link2,
    RefreshCcw,
    UserRound
  } from '@lucide/svelte';
  import {
    freshnessLabel,
    loadRuntime,
    loadScenarioControls,
    loadScenarioStatus,
    loadSnapshot,
    reviewAssociation
  } from '$lib/cop/client';
  import { reconcileSelection, resolveEntity, resolveMapSelection, type SelectableEntity } from '$lib/cop/selection';
  import SourceCard from '$lib/cop/SourceCard.svelte';
  import TacticalMap from '$lib/cop/TacticalMap.svelte';
  import { buildFeedRows, discoveryDiagnosticsForFeed } from '$lib/cop/sourceHealth';
  import type {
    Advisory,
    Alert,
    Association,
    AssociationReview,
    AssociationReviewDecision,
    Asset,
    CompanionFleet,
    EntityRef,
    Hazard,
    RuntimeSnapshot,
    ScenarioControls,
    ScenarioStatus,
    SensorFootprint,
    Snapshot,
    Task,
    Track,
    WeatherObservation
  } from '$lib/cop/types';

  let snapshot = $state<Snapshot | null>(null);
  let runtime = $state<RuntimeSnapshot | null>(null);
  let scenarioStatus = $state<ScenarioStatus | null>(null);
  let scenarioControls = $state<ScenarioControls | null>(null);
  let source = $state<'api' | 'fixture'>('fixture');
  let error = $state<string | undefined>();
  let selected = $state<EntityRef>({ kind: 'track', id: 'c360.edge.cop.mavlink.track.system-42' });
  let loading = $state(true);
  let associationReviews = $state<Record<string, AssociationReview>>({});
  let operatorID = $state('operator.local');

  const selectedEntity = $derived(resolveEntity(snapshot, selected));
  const mapSelected = $derived(resolveMapSelection(snapshot, selected) ?? selected);
  const feedRows = $derived(buildFeedRows(snapshot, runtime));
  const effectiveOperatorID = $derived(operatorID.trim() || 'operator.local');
  const operatorStorageKey = 'semops.operator.id';

  async function refresh() {
    loading = true;
    const [snapshotResult, runtimeResult, scenarioResult, controlsResult] = await Promise.all([
      loadSnapshot(),
      loadRuntime(),
      loadScenarioStatus(),
      loadScenarioControls()
    ]);
    snapshot = snapshotResult.snapshot;
    runtime = runtimeResult.runtime;
    scenarioStatus = scenarioResult.status;
    scenarioControls = controlsResult.controls;
    source = snapshotResult.source;
    error = [
      snapshotResult.error,
      runtimeResult.error,
      scenarioResult.error,
      controlsResult.error
    ].filter(Boolean).join('; ') || undefined;
    selected = reconcileSelection(snapshot, selected);
    loading = false;
  }

  onMount(() => {
    const storedOperatorID = localStorage.getItem(operatorStorageKey)?.trim();
    if (storedOperatorID) {
      operatorID = storedOperatorID;
    }
    void refresh();
  });

  function setOperatorID(value: string) {
    operatorID = value;
    const next = effectiveOperatorID;
    if (next) {
      localStorage.setItem(operatorStorageKey, next);
    } else {
      localStorage.removeItem(operatorStorageKey);
    }
  }

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

  function scenarioTone(status: ScenarioStatus) {
    if (status.state === 'failed' || status.summary.errors > 0 || status.failed_steps > 0) return 'failed';
    if (
      status.state === 'succeeded' &&
      status.ingress_mode === 'feed-boundary' &&
      status.summary.mutations === 0 &&
      status.summary.contract_graph_mutation_attempts === 0
    ) {
      return 'validated';
    }
    if (status.state === 'running' || status.state === 'idle') return 'running';
    return 'attention';
  }

  function scenarioIDLabel(id: string) {
    return id.replace(/-/g, ' ');
  }

  function scenarioControlLabel(action: string) {
    return action.replace(/_/g, ' ');
  }

  function associationStatusLabel(status: string | undefined) {
    switch ((status ?? '').toLowerCase()) {
      case 'ambiguous':
        return 'ambiguous evidence';
      case 'associated':
        return 'candidate evidence';
      case 'stale':
        return 'stale evidence';
      default:
        return status || 'evidence';
    }
  }

  function associationReviewFor(association: Association) {
    return associationReviews[association.id] ?? association.operator_review;
  }

  function associationReviewLabel(review: AssociationReview | undefined) {
    switch (review?.decision) {
      case 'acknowledged':
        return 'acknowledged';
      case 'challenged':
        return 'challenged';
      default:
        return 'unreviewed';
    }
  }

  async function submitAssociationReview(association: Association, decision: AssociationReviewDecision) {
    const optimisticReview: AssociationReview = {
      association_id: association.id,
      decision,
      reviewed_by: effectiveOperatorID,
      reviewed_at: new Date().toISOString(),
      reviewer_role: 'operator.unverified',
      authority_scope: 'local.display_only',
      conflict_policy: 'latest_review_wins_display_only'
    };
    associationReviews = { ...associationReviews, [association.id]: optimisticReview };
    try {
      const review = await reviewAssociation(association.id, {
        decision,
        reviewed_by: optimisticReview.reviewed_by
      }, {
        operatorID: effectiveOperatorID
      });
      associationReviews = { ...associationReviews, [association.id]: review };
      error = undefined;
    } catch (reviewError) {
      error = reviewError instanceof Error ? reviewError.message : 'association review unavailable';
    }
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
          <dt>Assoc</dt>
          <dd>{snapshot.summary.active_associations}</dd>
        </div>
        <div>
          <dt>Nodes</dt>
          <dd>{snapshot.summary.active_companion_nodes}</dd>
        </div>
        <div>
          <dt>Feeds</dt>
          <dd>{feedRows.length}</dd>
        </div>
      </dl>
    {/if}
    <label class="operator-control">
      <span>
        <UserRound size={14} />
        Operator
      </span>
      <input
        aria-describedby="operator-scope"
        autocomplete="off"
        maxlength="128"
        spellcheck="false"
        bind:value={operatorID}
        oninput={(event) => setOperatorID((event.target as HTMLInputElement).value)}
      />
      <small id="operator-scope">local.display_only</small>
    </label>
    <button
      class="icon-button"
      type="button"
      onclick={refresh}
      aria-label="Refresh COP snapshot"
      title="Refresh COP snapshot"
    >
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

      <section class="right-rail">
        {#if scenarioStatus}
          {@const tone = scenarioTone(scenarioStatus)}
          <section class="scenario-strip" aria-label="Scenario evidence">
            <div class="scenario-title">
              {#if tone === 'validated'}
                <CheckCircle2 size={17} />
              {:else if tone === 'failed'}
                <CircleAlert size={17} />
              {:else}
                <Activity size={17} />
              {/if}
              <h2>Scenario</h2>
            </div>
            <strong>{scenarioIDLabel(scenarioStatus.scenario_id)}</strong>
            <div class="scenario-state" class:validated={tone === 'validated'} class:failed={tone === 'failed'}>
              <span>{scenarioStatus.state}</span>
              <span>{scenarioStatus.ingress_mode ?? 'unknown ingress'}</span>
            </div>
            <dl class="scenario-metrics">
              <div>
                <dt>Steps</dt>
                <dd>{scenarioStatus.completed_steps}/{scenarioStatus.completed_steps + scenarioStatus.failed_steps}</dd>
              </div>
              <div>
                <dt>Delivered</dt>
                <dd>{scenarioStatus.summary.feed_boundary_deliveries}</dd>
              </div>
              <div>
                <dt>Graph</dt>
                <dd>{scenarioStatus.summary.mutations}</dd>
              </div>
            </dl>
            {#if scenarioStatus.last_error}
              <p class="scenario-error">{scenarioStatus.last_error}</p>
            {/if}
          </section>
        {/if}

        {#if scenarioControls}
          <section class="scenario-control-strip" aria-label="Scenario control policy">
            <div class="scenario-title blocked">
              <CircleAlert size={17} />
              <h2>Controls</h2>
            </div>
            <div class="scenario-state failed">
              <span>{scenarioControls.state}</span>
              <span>{scenarioControls.required_claim_scope}</span>
            </div>
            <div class="control-action-list" aria-label="Blocked scenario actions">
              {#each scenarioControls.supported_actions as action}
                <span>{scenarioControlLabel(action)}</span>
              {/each}
            </div>
            {#if scenarioControls.checkpoint_id || scenarioControls.checkpoint_state}
              <dl class="scenario-metrics">
                <div>
                  <dt>Checkpoint</dt>
                  <dd>{scenarioControls.checkpoint_id ?? 'missing'}</dd>
                </div>
                <div>
                  <dt>State</dt>
                  <dd>{scenarioControls.checkpoint_state ?? 'missing'}</dd>
                </div>
                <div>
                  <dt>Mode</dt>
                  <dd>{scenarioControls.enabled ? 'open' : 'closed'}</dd>
                </div>
              </dl>
            {/if}
            <p class="scenario-error">{scenarioControls.reason}</p>
          </section>
        {/if}

        {#if (snapshot.companion_fleets ?? []).length > 0}
          <section class="companion-strip" aria-label="SemLink companion fleet evidence">
            <h2>Companions</h2>
            {#each snapshot.companion_fleets as fleet}
              <button
                class:selected={selected.kind === 'companion-fleet' && selected.id === fleet.id}
                class="companion-row"
                type="button"
                aria-label={`Inspect ${fleet.label}`}
                aria-pressed={selected.kind === 'companion-fleet' && selected.id === fleet.id}
                onclick={() => selectEntity('companion-fleet', fleet.id)}
              >
                <Activity size={16} />
                <span>{fleet.label}</span>
                <small>
                  {fleet.node_count} nodes
                  {#if (fleet.sitl_backed_nodes ?? 0) > 0}
                    / {fleet.sitl_backed_nodes} SITL
                  {/if}
                </small>
              </button>
            {/each}
          </section>
        {/if}

        <section class="task-strip">
          <h2>Tasks</h2>
          {#each snapshot.tasks as task}
            <button
              class:selected={selected.kind === 'task' && selected.id === task.id}
              class="task-row"
              type="button"
              aria-pressed={selected.kind === 'task' && selected.id === task.id}
              onclick={() => selectEntity('task', task.id)}
            >
              <Activity size={16} />
              <span>{task.label}</span>
              <small>{task.status}</small>
            </button>
          {/each}
        </section>

        <section class="association-strip">
          <h2>Associations</h2>
          {#each snapshot.associations as association}
            <button
              class:selected={selected.kind === 'association' && selected.id === association.id}
              class="association-row"
              type="button"
              aria-label={`Inspect ${association.label}`}
              aria-pressed={selected.kind === 'association' && selected.id === association.id}
              onclick={() => selectEntity('association', association.id)}
            >
              <Link2 size={16} />
              <span>{association.label}</span>
              <small>{associationStatusLabel(association.status)}</small>
            </button>
          {/each}
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

{#snippet entityInspector(
  entity:
    | Track
    | Asset
    | Task
    | Advisory
    | Hazard
    | SensorFootprint
    | WeatherObservation
    | Association
    | CompanionFleet
    | Alert
)}
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
        <strong>{'primary_track_id' in entity ? associationStatusLabel(entity.status) : entity.status}</strong>
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

  {#if 'vehicle_profile' in entity && 'nodes' in entity}
    <section class="provenance">
      <h3>SemLink Demo Evidence</h3>
      <dl class="detail-list">
        <div>
          <dt>Evidence kind</dt>
          <dd>{entity.evidence_kind}</dd>
        </div>
        <div>
          <dt>Vehicle profile</dt>
          <dd>{entity.vehicle_profile}</dd>
        </div>
        <div>
          <dt>Nodes</dt>
          <dd>{entity.node_count}</dd>
        </div>
        <div>
          <dt>Vehicles</dt>
          <dd>{entity.vehicle_count}</dd>
        </div>
        <div>
          <dt>Expected summaries</dt>
          <dd>{entity.expected_summaries ?? 'unavailable'}</dd>
        </div>
        <div>
          <dt>Assertions</dt>
          <dd>{entity.assertion_state}</dd>
        </div>
        {#if entity.source_fidelity}
          <div>
            <dt>Fidelity</dt>
            <dd>{entity.source_fidelity}</dd>
          </div>
        {/if}
        {#if entity.live_source_summary}
          <div>
            <dt>Live source</dt>
            <dd>{entity.live_source_summary}</dd>
          </div>
        {/if}
        <div>
          <dt>SITL nodes</dt>
          <dd>{entity.sitl_backed_nodes ?? 0}</dd>
        </div>
        {#if entity.semlink_version || entity.semlink_commit}
          <div>
            <dt>SemLink build</dt>
            <dd>{entity.semlink_version || entity.semlink_commit}</dd>
          </div>
        {/if}
        {#if entity.generator_profile}
          <div>
            <dt>Generator</dt>
            <dd>{entity.generator_profile}</dd>
          </div>
        {/if}
        {#if entity.simulator_family}
          <div>
            <dt>Simulator</dt>
            <dd>{entity.simulator_family}</dd>
          </div>
        {/if}
        <div>
          <dt>Raw MAVLink</dt>
          <dd>{entity.raw_mavlink_excluded ? 'excluded from mesh summaries' : 'exclusion not proven'}</dd>
        </div>
        <div>
          <dt>Readback adapter</dt>
          <dd>{entity.readback.adapter_status}</dd>
        </div>
        <div>
          <dt>COMMAND_ACK</dt>
          <dd>
            {entity.readback.command_ack_status}
            {#if entity.readback.command_ack_result}
              / {entity.readback.command_ack_result}
            {/if}
          </dd>
        </div>
        <div>
          <dt>Readback result</dt>
          <dd>
            {entity.readback.result_observed_property || entity.readback.result_status}
          </dd>
        </div>
        <div>
          <dt>Native transmit</dt>
          <dd>{entity.readback.native_execution_allowed ? 'allowed' : 'not allowed'}</dd>
        </div>
        <div>
          <dt>Companion transmit</dt>
          <dd>{entity.readback.companion_transmit_allowed ? 'allowed' : 'not allowed'}</dd>
        </div>
        <div>
          <dt>Command posture</dt>
          <dd>{entity.command_posture.status}</dd>
        </div>
        <div>
          <dt>Hardware transmit</dt>
          <dd>{entity.command_posture.hardware_transmit_authorized ? 'authorized' : 'not authorized'}</dd>
        </div>
        {#if entity.raw_mavlink_policy}
          <div>
            <dt>Raw policy</dt>
            <dd>{entity.raw_mavlink_policy}</dd>
          </div>
        {/if}
      </dl>
      <p class="reason">{entity.no_transmit_posture}</p>
      <p class="reason">{entity.demo_evidence_label}</p>
    </section>

    <section class="provenance">
      <h3>Companion Nodes</h3>
      <div class="companion-node-list" aria-label="Companion node evidence">
        {#each entity.nodes as node}
          <div class="companion-node-card">
            <strong>{node.id}</strong>
            <span>
              {#if node.source_fidelity}
                {node.source_fidelity} /
              {/if}
              {node.vehicle_count} {node.vehicle_count === 1 ? 'vehicle' : 'vehicles'} /
              {node.peer_count} peers /
              {node.watermark_count ?? 0} watermarks
            </span>
            <small>
              summaries {node.initial_summary_count ?? 'unavailable'} -> {node.final_summary_count ?? 'unavailable'};
              diffs {node.applied_diff_count ?? 'unavailable'}/{node.diff_item_count ?? 'unavailable'}
            </small>
            {#if node.ttl_merge_posture}
              <small>{node.ttl_merge_posture}</small>
            {/if}
            {#if node.live_source_posture}
              <small>{node.live_source_posture}</small>
            {/if}
            {#if node.simulator_family || node.vehicle_source || node.mavlink_system_id}
              <small>
                {node.simulator_family ?? 'source'} /
                {node.vehicle_source ?? 'vehicle source unavailable'}
                {#if node.mavlink_system_id}
                  / system {node.mavlink_system_id}
                {/if}
              </small>
            {/if}
            {#if node.route}
              <small>{node.route}</small>
            {/if}
          </div>
        {/each}
      </div>
    </section>

    {#if entity.assertions.length > 0}
      <section class="provenance">
        <h3>Assertions</h3>
        <dl class="detail-list">
          {#each entity.assertions as assertion}
            <div>
              <dt>{assertion.name}</dt>
              <dd>
                {assertion.passed ? 'passed' : 'failed'}
                {#if assertion.detail}
                  / {assertion.detail}
                {/if}
              </dd>
            </div>
          {/each}
        </dl>
      </section>
    {/if}
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
        {#if entity.footprint && entity.footprint.length >= 3}
          <div>
            <dt>Footprint</dt>
            <dd>{entity.footprint.length} corners</dd>
          </div>
        {/if}
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

  {#if 'primary_track_id' in entity}
    {@const review = associationReviewFor(entity)}
    <section class="provenance">
      <h3>Association Evidence</h3>
      <dl class="detail-list">
        <div>
          <dt>Primary track</dt>
          <dd>{entity.primary_track_id}</dd>
        </div>
        <div>
          <dt>Candidate track</dt>
          <dd>{entity.candidate_track_id}</dd>
        </div>
        <div>
          <dt>Algorithm</dt>
          <dd>{entity.algorithm}</dd>
        </div>
        {#if entity.distance_meters !== undefined}
          <div>
            <dt>Distance</dt>
            <dd>{entity.distance_meters.toFixed(1).replace(/\.0$/, '')} m</dd>
          </div>
        {/if}
        {#if entity.time_delta_seconds !== undefined}
          <div>
            <dt>Time delta</dt>
            <dd>{entity.time_delta_seconds.toFixed(1).replace(/\.0$/, '')} s</dd>
          </div>
        {/if}
        <div>
          <dt>Operator review</dt>
          <dd>{associationReviewLabel(review)}</dd>
        </div>
        {#if review}
          <div>
            <dt>Reviewed by</dt>
            <dd>{review.reviewed_by}</dd>
          </div>
          <div>
            <dt>Reviewed</dt>
            <dd>{formatInstant(review.reviewed_at)}</dd>
          </div>
          <div>
            <dt>Role</dt>
            <dd>{review.reviewer_role}</dd>
          </div>
          <div>
            <dt>Authority</dt>
            <dd>{review.authority_scope}</dd>
          </div>
          <div>
            <dt>Conflict</dt>
            <dd>{review.conflict_policy}</dd>
          </div>
        {/if}
      </dl>
      <div class="association-actions" aria-label="Association operator review">
        <button
          class:active={review?.decision === 'acknowledged'}
          type="button"
          aria-label="Acknowledge association evidence"
          aria-pressed={review?.decision === 'acknowledged'}
          onclick={() => submitAssociationReview(entity, 'acknowledged')}
        >
          <CheckCircle2 size={16} />
          <span>Acknowledge</span>
        </button>
        <button
          class:active={review?.decision === 'challenged'}
          type="button"
          aria-label="Challenge association evidence"
          aria-pressed={review?.decision === 'challenged'}
          onclick={() => submitAssociationReview(entity, 'challenged')}
        >
          <CircleAlert size={16} />
          <span>Challenge</span>
        </button>
      </div>
      <p class="reason">{entity.reason}</p>
      <p class="reason">{entity.claim_posture}</p>
    </section>
  {/if}

  {#if 'target_id' in entity && entity.target_id}
    <section class="provenance">
      <h3>Command Intent</h3>
      <dl class="detail-list">
        <div>
          <dt>Target</dt>
          <dd>{entity.target_id}</dd>
        </div>
        {#if entity.authority}
          <div>
            <dt>Authority</dt>
            <dd>{entity.authority}</dd>
          </div>
        {/if}
        {#if entity.priority !== undefined}
          <div>
            <dt>Priority</dt>
            <dd>{entity.priority}</dd>
          </div>
        {/if}
        {#if entity.expires_at}
          <div>
            <dt>Expires</dt>
            <dd>{formatInstant(entity.expires_at)}</dd>
          </div>
        {/if}
        {#if entity.requested_by}
          <div>
            <dt>Requested by</dt>
            <dd>{entity.requested_by}</dd>
          </div>
        {/if}
        {#if entity.correlation_id}
          <div>
            <dt>Correlation</dt>
            <dd>{entity.correlation_id}</dd>
          </div>
        {/if}
        {#if entity.local_override_policy}
          <div>
            <dt>Local override</dt>
            <dd>{entity.local_override_policy}</dd>
          </div>
        {/if}
        {#if entity.desired_state}
          <div>
            <dt>Desired</dt>
            <dd>{entity.desired_state}</dd>
          </div>
        {/if}
      </dl>
      {#if entity.claim_posture}
        <p class="reason">{entity.claim_posture}</p>
      {/if}
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

  {#if 'entity_id' in entity}
    <dl class="detail-list">
      <div>
        <dt>Target</dt>
        <dd>{entity.entity_id}</dd>
      </div>
    </dl>
    <p class="reason">{entity.reason}</p>
  {/if}
{/snippet}
