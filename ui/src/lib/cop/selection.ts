import type { Advisory, Alert, Asset, EntityRef, Hazard, SensorFootprint, Snapshot, Task, Track } from './types';

export type SelectableEntity = Track | Asset | Task | Advisory | Hazard | SensorFootprint | Alert;

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
  return snapshot.alerts.find((alert) => alert.id === selected.id);
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
  if (snapshot.alerts[0]) {
    return { kind: 'alert', id: snapshot.alerts[0].id };
  }
  return selected;
}
