import type {
  Advisory,
  Alert,
  Asset,
  EntityRef,
  Hazard,
  SensorFootprint,
  Snapshot,
  Task,
  Track,
  WeatherObservation
} from './types';

export type SelectableEntity = Track | Asset | Task | Advisory | Hazard | SensorFootprint | WeatherObservation | Alert;

export function resolveEntity(snapshot: Snapshot | null, selected: EntityRef): SelectableEntity | undefined {
  if (!snapshot) {
    return undefined;
  }
  if (selected.kind === 'track') {
    return snapshot.tracks.find((track) => track.id === selected.id);
  }
  if (selected.kind === 'asset') {
    return snapshot.assets.find((asset) => asset.id === selected.id);
  }
  if (selected.kind === 'task') {
    return snapshot.tasks.find((task) => task.id === selected.id);
  }
  if (selected.kind === 'advisory') {
    return snapshot.advisories.find((advisory) => advisory.id === selected.id);
  }
  if (selected.kind === 'hazard') {
    return snapshot.hazards.find((hazard) => hazard.id === selected.id);
  }
  if (selected.kind === 'sensor-footprint') {
    return (snapshot.sensor_footprints ?? []).find((footprint) => footprint.id === selected.id);
  }
  if (selected.kind === 'weather-observation') {
    return (snapshot.weather_observations ?? []).find((observation) => observation.id === selected.id);
  }
  return snapshot.alerts.find((alert) => alert.id === selected.id);
}

export function resolveMapSelection(snapshot: Snapshot | null, selected: EntityRef): EntityRef | undefined {
  if (!snapshot) {
    return undefined;
  }
  if (selected.kind !== 'alert') {
    return selected;
  }
  const alert = snapshot.alerts.find((candidate) => candidate.id === selected.id);
  if (!alert) {
    return undefined;
  }
  return resolveEntityRefByID(snapshot, alert.entity_id);
}

export function reconcileSelection(snapshot: Snapshot | null, selected: EntityRef): EntityRef {
  if (!snapshot || resolveEntity(snapshot, selected)) {
    return selected;
  }
  if (snapshot.tracks[0]) {
    return { kind: 'track', id: snapshot.tracks[0].id };
  }
  if (snapshot.assets[0]) {
    return { kind: 'asset', id: snapshot.assets[0].id };
  }
  if (snapshot.tasks[0]) {
    return { kind: 'task', id: snapshot.tasks[0].id };
  }
  if (snapshot.advisories[0]) {
    return { kind: 'advisory', id: snapshot.advisories[0].id };
  }
  if (snapshot.hazards[0]) {
    return { kind: 'hazard', id: snapshot.hazards[0].id };
  }
  if ((snapshot.sensor_footprints ?? [])[0]) {
    return { kind: 'sensor-footprint', id: snapshot.sensor_footprints[0].id };
  }
  if ((snapshot.weather_observations ?? [])[0]) {
    return { kind: 'weather-observation', id: snapshot.weather_observations[0].id };
  }
  if (snapshot.alerts[0]) {
    return { kind: 'alert', id: snapshot.alerts[0].id };
  }
  return selected;
}

function resolveEntityRefByID(snapshot: Snapshot, id: string): EntityRef | undefined {
  if (snapshot.tracks.some((track) => track.id === id)) {
    return { kind: 'track', id };
  }
  if (snapshot.assets.some((asset) => asset.id === id)) {
    return { kind: 'asset', id };
  }
  if (snapshot.tasks.some((task) => task.id === id)) {
    return { kind: 'task', id };
  }
  if (snapshot.advisories.some((advisory) => advisory.id === id)) {
    return { kind: 'advisory', id };
  }
  if (snapshot.hazards.some((hazard) => hazard.id === id)) {
    return { kind: 'hazard', id };
  }
  if ((snapshot.sensor_footprints ?? []).some((footprint) => footprint.id === id)) {
    return { kind: 'sensor-footprint', id };
  }
  if ((snapshot.weather_observations ?? []).some((observation) => observation.id === id)) {
    return { kind: 'weather-observation', id };
  }
  return undefined;
}
