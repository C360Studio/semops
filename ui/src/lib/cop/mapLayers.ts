import type { EntityRef, GeoPoint, Hazard, Snapshot } from './types';

export type TacticalEntityKind =
  | 'track'
  | 'asset'
  | 'task'
  | 'advisory'
  | 'hazard'
  | 'sensor-footprint'
  | 'weather-observation'
  | 'association';

export type TacticalPoint = {
  id: string;
  kind: Extract<
    TacticalEntityKind,
    'track' | 'asset' | 'task' | 'advisory' | 'sensor-footprint' | 'weather-observation'
  >;
  label: string;
  position: [number, number];
  selected: boolean;
  color: [number, number, number, number];
  radius: number;
  role?: 'sensor' | 'frame-center';
};

export type TacticalPolygon = {
  id: string;
  kind: Extract<TacticalEntityKind, 'hazard' | 'sensor-footprint'>;
  label: string;
  polygon: [number, number][];
  selected: boolean;
  fillColor: [number, number, number, number];
  lineColor: [number, number, number, number];
};

export type TacticalRay = {
  id: string;
  kind: 'sensor-footprint';
  label: string;
  source: [number, number];
  target: [number, number];
  selected: boolean;
  color: [number, number, number, number];
  width: number;
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

  const taskPoints = snapshot.tasks.flatMap((task): TacticalPoint[] => {
    if (!task.position) {
      return [];
    }
    return [
      {
        id: task.id,
        kind: 'task',
        label: task.label,
        position: lngLat(task.position),
        selected: selected.kind === 'task' && selected.id === task.id,
        color: selected.kind === 'task' && selected.id === task.id ? [98, 74, 145, 248] : [112, 86, 158, 222],
        radius: selected.kind === 'task' && selected.id === task.id ? 86 : 62
      }
    ];
  });

  const advisoryPoints = snapshot.advisories.flatMap((advisory): TacticalPoint[] => {
    if (!advisory.position) {
      return [];
    }
    return [
      {
        id: advisory.id,
        kind: 'advisory',
        label: advisory.label,
        position: lngLat(advisory.position),
        selected: selected.kind === 'advisory' && selected.id === advisory.id,
        color:
          selected.kind === 'advisory' && selected.id === advisory.id ? [180, 116, 42, 248] : [189, 126, 48, 224],
        radius: selected.kind === 'advisory' && selected.id === advisory.id ? 74 : 52
      }
    ];
  });

  const footprintPoints = (snapshot.sensor_footprints ?? []).flatMap((footprint): TacticalPoint[] => {
    const isSelected = selected.kind === 'sensor-footprint' && selected.id === footprint.id;
    return [
      {
        id: footprint.id,
        kind: 'sensor-footprint',
        label: `${footprint.label} sensor`,
        position: lngLat(footprint.sensor_position),
        selected: isSelected,
        color: isSelected ? [18, 92, 113, 250] : [23, 103, 122, 224],
        radius: isSelected ? 86 : 62,
        role: 'sensor'
      },
      {
        id: footprint.id,
        kind: 'sensor-footprint',
        label: `${footprint.label} frame center`,
        position: lngLat(footprint.frame_center),
        selected: isSelected,
        color: isSelected ? [211, 126, 44, 250] : [204, 137, 59, 224],
        radius: isSelected ? 72 : 50,
        role: 'frame-center'
      }
    ];
  });

  const weatherPoints = (snapshot.weather_observations ?? []).flatMap((observation): TacticalPoint[] => {
    if (!observation.position) {
      return [];
    }
    const isSelected = selected.kind === 'weather-observation' && selected.id === observation.id;
    return [
      {
        id: observation.id,
        kind: 'weather-observation',
        label: observation.label,
        position: lngLat(observation.position),
        selected: isSelected,
        color: isSelected ? [62, 115, 131, 248] : [78, 134, 151, 224],
        radius: isSelected ? 82 : 58
      }
    ];
  });

  return [...assetPoints, ...trackPoints, ...taskPoints, ...advisoryPoints, ...footprintPoints, ...weatherPoints];
}

export function tacticalPolygons(snapshot: Snapshot, selected: EntityRef): TacticalPolygon[] {
  const hazards = snapshot.hazards
    .filter((hazard) => hazard.geometry.length >= 3)
    .map((hazard): TacticalPolygon => {
      const isSelected = selected.kind === 'hazard' && selected.id === hazard.id;
      return {
        id: hazard.id,
        kind: 'hazard' as const,
        label: hazard.label,
        polygon: hazard.geometry.map(lngLat),
        selected: isSelected,
        fillColor: isSelected ? [207, 91, 49, 78] : [198, 105, 42, 58],
        lineColor: isSelected ? [145, 45, 35, 235] : [168, 79, 40, 205]
      };
    });
  const footprints = (snapshot.sensor_footprints ?? [])
    .filter((footprint) => (footprint.footprint ?? []).length >= 3)
    .map((footprint): TacticalPolygon => {
      const isSelected = selected.kind === 'sensor-footprint' && selected.id === footprint.id;
      return {
        id: footprint.id,
        kind: 'sensor-footprint' as const,
        label: footprint.label,
        polygon: (footprint.footprint ?? []).map(lngLat),
        selected: isSelected,
        fillColor: isSelected ? [24, 120, 136, 68] : [32, 130, 145, 46],
        lineColor: isSelected ? [16, 92, 108, 235] : [34, 113, 126, 196]
      };
    });
  return [...hazards, ...footprints];
}

export function tacticalRays(snapshot: Snapshot, selected: EntityRef): TacticalRay[] {
  return (snapshot.sensor_footprints ?? []).map((footprint) => {
    const isSelected = selected.kind === 'sensor-footprint' && selected.id === footprint.id;
    return {
      id: footprint.id,
      kind: 'sensor-footprint',
      label: footprint.label,
      source: lngLat(footprint.sensor_position),
      target: lngLat(footprint.frame_center),
      selected: isSelected,
      color: isSelected ? [20, 86, 99, 235] : [42, 104, 113, 180],
      width: isSelected ? 4 : 2
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
    })),
    ...snapshot.tasks.flatMap((task) =>
      task.position
        ? [
            {
              id: task.id,
              kind: 'task' as const,
              label: task.label,
              position: lngLat(task.position),
              offset: [14, 18] as [number, number],
              anchor: 'start' as const
            }
          ]
        : []
    ),
    ...snapshot.advisories.flatMap((advisory) =>
      advisory.position
        ? [
            {
              id: advisory.id,
              kind: 'advisory' as const,
              label: advisory.label,
              position: lngLat(advisory.position),
              offset: [0, 30] as [number, number],
              anchor: 'middle' as const
            }
          ]
        : []
    )
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

  const footprintLabels = (snapshot.sensor_footprints ?? []).map((footprint) => ({
    id: footprint.id,
    kind: 'sensor-footprint' as const,
    label: footprint.label,
    position: lngLat(footprint.frame_center),
    offset: [18, 20] as [number, number],
    anchor: 'start' as const
  }));

  const weatherLabels = (snapshot.weather_observations ?? []).flatMap((observation) =>
    observation.position
      ? [
          {
            id: observation.id,
            kind: 'weather-observation' as const,
            label: observation.label,
            position: lngLat(observation.position),
            offset: [-16, -28] as [number, number],
            anchor: 'end' as const
          }
        ]
      : []
  );

  return [...pointLabels, ...hazardLabels, ...footprintLabels, ...weatherLabels];
}

export function tacticalSelectionItems(snapshot: Snapshot): TacticalSelectionItem[] {
  return [
    ...snapshot.tracks.map((track) => ({ id: track.id, kind: 'track' as const, label: track.label })),
    ...snapshot.assets.map((asset) => ({ id: asset.id, kind: 'asset' as const, label: asset.label })),
    ...snapshot.tasks.map((task) => ({ id: task.id, kind: 'task' as const, label: task.label })),
    ...snapshot.advisories.map((advisory) => ({ id: advisory.id, kind: 'advisory' as const, label: advisory.label })),
    ...snapshot.hazards.map((hazard) => ({ id: hazard.id, kind: 'hazard' as const, label: hazard.label })),
    ...(snapshot.sensor_footprints ?? []).map((footprint) => ({
      id: footprint.id,
      kind: 'sensor-footprint' as const,
      label: footprint.label
    })),
    ...(snapshot.weather_observations ?? []).map((observation) => ({
      id: observation.id,
      kind: 'weather-observation' as const,
      label: observation.label
    })),
    ...(snapshot.associations ?? []).map((association) => ({
      id: association.id,
      kind: 'association' as const,
      label: association.label
    }))
  ];
}

export function tacticalMapView(snapshot: Snapshot): TacticalMapView {
  const points = [
    ...snapshot.tracks.map((track) => track.position),
    ...snapshot.assets.flatMap((asset) => (asset.position ? [asset.position] : [])),
    ...snapshot.tasks.flatMap((task) => (task.position ? [task.position] : [])),
    ...snapshot.advisories.flatMap((advisory) => (advisory.position ? [advisory.position] : [])),
    ...snapshot.hazards.flatMap((hazard) => hazard.geometry),
    ...(snapshot.sensor_footprints ?? []).flatMap((footprint) => [
      footprint.sensor_position,
      footprint.frame_center,
      ...(footprint.footprint ?? [])
    ]),
    ...(snapshot.weather_observations ?? []).flatMap((observation) =>
      observation.position ? [observation.position] : []
    )
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
