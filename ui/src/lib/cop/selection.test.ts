import { describe, expect, it } from 'vitest';
import { fixtureSnapshot } from './fixture';
import { reconcileSelection, resolveEntity } from './selection';
import type { Snapshot } from './types';

describe('COP selection helpers', () => {
  it('resolves the selected entity across supported COP collections', () => {
    expect(resolveEntity(fixtureSnapshot, { kind: 'track', id: fixtureSnapshot.tracks[0].id })?.label).toBe('UAS 42');
    expect(resolveEntity(fixtureSnapshot, { kind: 'asset', id: fixtureSnapshot.assets[0].id })?.label).toBe('MAVLink system 42');
    expect(resolveEntity(fixtureSnapshot, { kind: 'hazard', id: fixtureSnapshot.hazards[0].id })?.label).toBe(
      'Flood watch sector'
    );
    expect(resolveEntity(fixtureSnapshot, { kind: 'alert', id: fixtureSnapshot.alerts[0].id })?.label).toBe(
      'Track freshness nominal'
    );
  });

  it('preserves a valid selection after snapshot refresh', () => {
    const selected = { kind: 'hazard' as const, id: fixtureSnapshot.hazards[0].id };

    expect(reconcileSelection(fixtureSnapshot, selected)).toEqual(selected);
  });

  it('falls back to the first available operator entity when the selection goes stale', () => {
    const stale = { kind: 'track' as const, id: 'missing' };

    expect(reconcileSelection(fixtureSnapshot, stale)).toEqual({
      kind: 'track',
      id: fixtureSnapshot.tracks[0].id
    });
    expect(reconcileSelection(without(fixtureSnapshot, 'tracks'), stale)).toEqual({
      kind: 'asset',
      id: fixtureSnapshot.assets[0].id
    });
    expect(reconcileSelection(without(without(fixtureSnapshot, 'tracks'), 'assets'), stale)).toEqual({
      kind: 'hazard',
      id: fixtureSnapshot.hazards[0].id
    });
    expect(
      reconcileSelection(without(without(without(fixtureSnapshot, 'tracks'), 'assets'), 'hazards'), stale)
    ).toEqual({
      kind: 'alert',
      id: fixtureSnapshot.alerts[0].id
    });
  });
});

function without<K extends keyof Pick<Snapshot, 'tracks' | 'assets' | 'hazards' | 'alerts'>>(
  snapshot: Snapshot,
  key: K
): Snapshot {
  return { ...snapshot, [key]: [] };
}
