import type { EntityRef, GeoPoint, Hazard, Snapshot } from './types';

export type TacticalEntityKind = 'track' | 'asset' | 'hazard';

export type TacticalPoint = {
  id: string;
  kind: Extract<TacticalEntityKind, 'track' | 'asset'>;
  label: string;
  position: [number, number];
  selected: boolean;
  color: [number, number, number, number];
  radius: number;
};

export type TacticalPolygon = {
  id: string;
  kind: 'hazard';
  label: string;
  polygon: [number, number][];
  selected: boolean;
  fillColor: [number, number, number, number];
  lineColor: [number, number, number, number];
};

export type TacticalSelectionItem = {
  id: string;
  kind: TacticalEntityKind;
  label: string;
};

export type TacticalLabel = {
  id: string;
  label: string;
  kind: TacticalEntityKind;
  position: [number, number];
  offset: [number, number];
  anchor: 'start' | 'middle' | 'end';
};

export type TacticalMapView = {
  center: [number, number];
  bounds: [[number, number], [number, number]];
  key: string;
};

const defaultCenter: [number, number] = [-77.0002, 38.9001];

export function tacticalPoints(snapshot: Snapshot, selected: EntityRef): TacticalPoint[] {
  const assetPoints = snapshot.assets.flatMap((asset): TacticalPoint[] => {
    if (!asset.position) {
      return [];
    }
    return [
      {
        id: asset.id,
        kind: 'asset',
        label: asset.label,
        position: lngLat(asset.position),
        selected: selected.kind === 'asset' && selected.id === asset.id,
        color: selected.kind === 'asset' && selected.id === asset.id ? [34, 111, 68, 245] : [44, 122, 75, 210],
        radius: selected.kind === 'asset' && selected.id === asset.id ? 80 : 58
      }
    ];
  });

  const trackPoints = snapshot.tracks.map((track): TacticalPoint => ({
    id: track.id,
    kind: 'track',
    label: track.label,
    position: lngLat(track.position),
    selected: selected.kind === 'track' && selected.id === track.id,
    color: selected.kind === 'track' && selected.id === track.id ? [22, 99, 131, 250] : [29, 120, 150, 230],
    radius: selected.kind === 'track' && selected.id === track.id ? 92 : 66
  }));

  return [...assetPoints, ...trackPoints];
}

export function tacticalPolygons(snapshot: Snapshot, selected: EntityRef): TacticalPolygon[] {
  return snapshot.hazards
    .filter((hazard) => hazard.geometry.length >= 3)
    .map((hazard) => {
      const isSelected = selected.kind === 'hazard' && selected.id === hazard.id;
      return {
        id: hazard.id,
        kind: 'hazard',
        label: hazard.label,
        polygon: hazard.geometry.map(lngLat),
        selected: isSelected,
        fillColor: isSelected ? [207, 91, 49, 78] : [198, 105, 42, 58],
        lineColor: isSelected ? [145, 45, 35, 235] : [168, 79, 40, 205]
      };
    });
}

export function tacticalLabels(snapshot: Snapshot): TacticalLabel[] {
  const pointLabels = [
    ...snapshot.assets.flatMap((asset) =>
      asset.position
        ? [
            {
              id: asset.id,
              kind: 'asset' as const,
              label: asset.label,
              position: lngLat(asset.position),
              offset: [-14, 18] as [number, number],
              anchor: 'end' as const
            }
          ]
        : []
    ),
    ...snapshot.tracks.map((track) => ({
      id: track.id,
      kind: 'track' as const,
      label: track.label,
      position: lngLat(track.position),
      offset: [16, -18] as [number, number],
      anchor: 'start' as const
    }))
  ];
  const hazardLabels = snapshot.hazards
    .filter((hazard) => hazard.geometry.length > 0)
    .map((hazard) => ({
      id: hazard.id,
      kind: 'hazard' as const,
      label: hazard.label,
      position: lngLat(hazardCentroid(hazard)),
      offset: [0, -48] as [number, number],
      anchor: 'middle' as const
    }));

  return [...pointLabels, ...hazardLabels];
}

export function tacticalSelectionItems(snapshot: Snapshot): TacticalSelectionItem[] {
  return [
    ...snapshot.tracks.map((track) => ({ id: track.id, kind: 'track' as const, label: track.label })),
    ...snapshot.assets.map((asset) => ({ id: asset.id, kind: 'asset' as const, label: asset.label })),
    ...snapshot.hazards.map((hazard) => ({ id: hazard.id, kind: 'hazard' as const, label: hazard.label }))
  ];
}

export function tacticalMapView(snapshot: Snapshot): TacticalMapView {
  const points = [
    ...snapshot.tracks.map((track) => track.position),
    ...snapshot.assets.flatMap((asset) => (asset.position ? [asset.position] : [])),
    ...snapshot.hazards.flatMap((hazard) => hazard.geometry)
  ];
  if (points.length === 0) {
    return {
      center: defaultCenter,
      bounds: [
        [defaultCenter[0] - 0.02, defaultCenter[1] - 0.02],
        [defaultCenter[0] + 0.02, defaultCenter[1] + 0.02]
      ],
      key: 'empty'
    };
  }

  const lats = points.map((point) => point.lat);
  const lons = points.map((point) => point.lon);
  const minLon = Math.min(...lons) - 0.01;
  const maxLon = Math.max(...lons) + 0.01;
  const minLat = Math.min(...lats) - 0.008;
  const maxLat = Math.max(...lats) + 0.008;

  return {
    center: [(minLon + maxLon) / 2, (minLat + maxLat) / 2],
    bounds: [
      [minLon, minLat],
      [maxLon, maxLat]
    ],
    key: points.map((point) => `${point.lat.toFixed(5)},${point.lon.toFixed(5)}`).join('|')
  };
}

function hazardCentroid(hazard: Hazard): GeoPoint {
  const sum = hazard.geometry.reduce(
    (next, point) => ({ lat: next.lat + point.lat, lon: next.lon + point.lon }),
    { lat: 0, lon: 0 }
  );
  return { lat: sum.lat / hazard.geometry.length, lon: sum.lon / hazard.geometry.length };
}

function lngLat(point: GeoPoint): [number, number] {
  return [point.lon, point.lat];
}
