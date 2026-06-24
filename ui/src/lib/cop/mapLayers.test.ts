import { describe, expect, it } from 'vitest';
import { fixtureSnapshot } from './fixture';
import {
  tacticalLabels,
  tacticalMapView,
  tacticalPoints,
  tacticalPolygons,
  tacticalRays,
  tacticalSelectionItems
} from './mapLayers';

describe('tactical map layer helpers', () => {
  it('projects COP entities into deck.gl point and polygon data', () => {
    const selected = { kind: 'track' as const, id: fixtureSnapshot.tracks[0].id };

    const points = tacticalPoints(fixtureSnapshot, selected);
    const polygons = tacticalPolygons(fixtureSnapshot, selected);
    const rays = tacticalRays(fixtureSnapshot, selected);

    expect(points.map((point) => point.kind)).toEqual([
      'asset',
      'track',
      'track',
      'track',
      'task',
      'advisory',
      'sensor-footprint',
      'sensor-footprint',
      'weather-observation'
    ]);
    expect(points[1]).toMatchObject({
      id: fixtureSnapshot.tracks[0].id,
      position: [-77.0002, 38.9001],
      selected: true
    });
    expect(polygons).toHaveLength(1);
    expect(polygons[0].polygon[0]).toEqual([-77.012, 38.895]);
    expect(rays).toHaveLength(1);
    expect(rays[0]).toMatchObject({
      id: fixtureSnapshot.sensor_footprints[0].id,
      source: [-77.0254, 38.9022],
      target: [-77.0108, 38.8956]
    });
    expect(points.at(-1)).toMatchObject({
      id: fixtureSnapshot.weather_observations[0].id,
      kind: 'weather-observation',
      position: [-77.04, 38.9]
    });
  });

  it('builds readable labels, selection affordances, and stable map bounds', () => {
    const view = tacticalMapView(fixtureSnapshot);
    const items = tacticalSelectionItems(fixtureSnapshot);
    const labels = tacticalLabels(fixtureSnapshot);

    expect(view.center[0]).toBeGreaterThan(-77.03);
    expect(view.center[0]).toBeLessThan(-76.99);
    expect(view.bounds[0][0]).toBeLessThan(-77.04);
    expect(view.bounds[1][1]).toBeGreaterThan(38.908);
    expect(items.map((item) => item.kind)).toEqual([
      'track',
      'track',
      'track',
      'asset',
      'task',
      'task',
      'advisory',
      'hazard',
      'sensor-footprint',
      'weather-observation',
      'association'
    ]);
    expect(items.map((item) => item.label)).toContain('Ambiguous association UAS 42 -> N42CX');
    expect(items.map((item) => item.label)).toContain('Route MAVLink system 42 to North Gate');
    expect(labels.map((label) => [label.kind, label.anchor, label.offset])).toEqual([
      ['asset', 'end', [-14, 18]],
      ['track', 'start', [16, -18]],
      ['track', 'start', [16, -18]],
      ['track', 'start', [16, -18]],
      ['task', 'start', [14, 18]],
      ['advisory', 'middle', [0, 30]],
      ['hazard', 'middle', [0, -48]],
      ['sensor-footprint', 'start', [18, 20]],
      ['weather-observation', 'end', [-16, -28]]
    ]);
  });
});
