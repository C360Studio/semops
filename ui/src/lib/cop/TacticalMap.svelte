<script lang="ts">
  import { onDestroy, onMount } from 'svelte';
  import { ClipboardList, Crosshair, MapPinned, MessageSquareText, ShieldCheck } from '@lucide/svelte';
  import {
    tacticalLabels,
    tacticalMapView,
    tacticalPoints,
    tacticalPolygons,
    tacticalSelectionItems,
    type TacticalEntityKind
  } from './mapLayers';
  import type { EntityRef, Snapshot } from './types';

  let {
    snapshot,
    selected,
    onSelect
  }: {
    snapshot: Snapshot;
    selected: EntityRef;
    onSelect: (selected: EntityRef) => void;
  } = $props();

  let container = $state<HTMLDivElement | undefined>();
  let ready = $state(false);
  let loadError = $state<string | undefined>();

  let map: any;
  let overlay: any;
  let resizeObserver: ResizeObserver | undefined;
  let constructors: {
    ScatterplotLayer: any;
    PolygonLayer: any;
    TextLayer: any;
  };
  let lastViewKey = '';
  let destroyed = false;

  const selectionItems = $derived(tacticalSelectionItems(snapshot));

  onMount(() => {
    void bootMap();
    return () => {
      destroyed = true;
      resizeObserver?.disconnect();
      if (map && overlay) {
        map.removeControl(overlay);
      }
      map?.remove();
    };
  });

  onDestroy(() => {
    destroyed = true;
  });

  $effect(() => {
    snapshot;
    selected;
    syncMap();
  });

  async function bootMap() {
    try {
      const [{ default: maplibregl }, { MapboxOverlay }, layers] = await Promise.all([
        import('maplibre-gl'),
        import('@deck.gl/mapbox'),
        import('@deck.gl/layers')
      ]);
      if (!container || destroyed) {
        return;
      }

      constructors = {
        ScatterplotLayer: layers.ScatterplotLayer,
        PolygonLayer: layers.PolygonLayer,
        TextLayer: layers.TextLayer
      };

      const view = tacticalMapView(snapshot);
      map = new maplibregl.Map({
        container,
        style: {
          version: 8,
          sources: {},
          layers: [
            {
              id: 'semops-base',
              type: 'background',
              paint: { 'background-color': '#dbe7e0' }
            }
          ]
        },
        center: view.center,
        zoom: 11,
        attributionControl: false
      });
      map.addControl(new maplibregl.NavigationControl({ showZoom: true, showCompass: false }), 'top-right');

      overlay = new MapboxOverlay({ interleaved: false, layers: [] });
      map.addControl(overlay);
      map.once('load', () => {
        ready = true;
        syncMap();
      });

      resizeObserver = new ResizeObserver(() => map?.resize());
      resizeObserver.observe(container);
    } catch (error) {
      loadError = error instanceof Error ? error.message : 'map unavailable';
    }
  }

  function syncMap() {
    if (!map || !overlay || !constructors) {
      return;
    }
    overlay.setProps({ layers: deckLayers() });

    const view = tacticalMapView(snapshot);
    if (view.key !== lastViewKey) {
      lastViewKey = view.key;
      map.fitBounds(view.bounds, { padding: 48, duration: 350, maxZoom: 13 });
    }
  }

  function deckLayers() {
    const { ScatterplotLayer, PolygonLayer, TextLayer } = constructors;
    const points = tacticalPoints(snapshot, selected);
    const polygons = tacticalPolygons(snapshot, selected);
    const labels = tacticalLabels(snapshot);
    return [
      new PolygonLayer({
        id: 'semops-hazards',
        data: polygons,
        pickable: true,
        stroked: true,
        filled: true,
        getPolygon: (item: (typeof polygons)[number]) => item.polygon,
        getFillColor: (item: (typeof polygons)[number]) => item.fillColor,
        getLineColor: (item: (typeof polygons)[number]) => item.lineColor,
        getLineWidth: (item: (typeof polygons)[number]) => (item.selected ? 3 : 2),
        lineWidthMinPixels: 2,
        onClick: selectPicked
      }),
      new ScatterplotLayer({
        id: 'semops-points',
        data: points,
        pickable: true,
        radiusUnits: 'meters',
        stroked: true,
        filled: true,
        getPosition: (item: (typeof points)[number]) => item.position,
        getRadius: (item: (typeof points)[number]) => item.radius,
        getFillColor: (item: (typeof points)[number]) => item.color,
        getLineColor: [255, 255, 255, 230],
        getLineWidth: (item: (typeof points)[number]) => (item.selected ? 5 : 2),
        lineWidthUnits: 'pixels',
        onClick: selectPicked
      }),
      new TextLayer({
        id: 'semops-labels',
        data: labels,
        getPosition: (item: (typeof labels)[number]) => item.position,
        getText: (item: (typeof labels)[number]) => item.label,
        getSize: 12,
        getPixelOffset: (item: (typeof labels)[number]) => item.offset,
        getTextAnchor: (item: (typeof labels)[number]) => item.anchor,
        getAlignmentBaseline: 'center',
        getColor: [25, 38, 33, 230],
        background: true,
        getBackgroundColor: [248, 251, 249, 220],
        backgroundPadding: [5, 3],
        fontFamily: 'Inter, ui-sans-serif, system-ui, sans-serif',
        billboard: true
      })
    ];
  }

  function selectPicked(info: { object?: { id: string; kind: TacticalEntityKind } }) {
    const object = info.object;
    if (!object) {
      return;
    }
    selectEntity(object.kind, object.id);
  }

  function selectEntity(kind: TacticalEntityKind, id: string) {
    onSelect({ kind, id } as EntityRef);
  }
</script>

<div class="map-surface">
  <div class="map-canvas" bind:this={container}></div>
  {#if !ready && !loadError}
    <div class="map-loading" aria-live="polite">Loading</div>
  {/if}
  {#if loadError}
    <div class="map-error">{loadError}</div>
  {/if}
  <div class="map-selector" aria-label="Map entities">
    {#each selectionItems as item}
      <button
        class:selected={selected.kind === item.kind && selected.id === item.id}
        class={`map-selector-button ${item.kind}`}
        type="button"
        aria-pressed={selected.kind === item.kind && selected.id === item.id}
        aria-label={`Select ${item.label}`}
        title={item.label}
        onclick={() => selectEntity(item.kind, item.id)}
      >
        {#if item.kind === 'track'}
          <Crosshair size={16} />
        {:else if item.kind === 'asset'}
          <ShieldCheck size={16} />
        {:else if item.kind === 'task'}
          <ClipboardList size={16} />
        {:else if item.kind === 'advisory'}
          <MessageSquareText size={16} />
        {:else}
          <MapPinned size={16} />
        {/if}
      </button>
    {/each}
  </div>
</div>
