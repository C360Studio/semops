import { describe, expect, it } from 'vitest';
import { fixtureSnapshot } from './fixture';
import { reconcileSelection, resolveEntity, resolveMapSelection } from './selection';
import type { Snapshot } from './types';

describe('COP selection helpers', () => {
  it('resolves the selected entity across supported COP collections', () => {
    expect(resolveEntity(fixtureSnapshot, { kind: 'track', id: fixtureSnapshot.tracks[0].id })?.label).toBe('UAS 42');
    expect(resolveEntity(fixtureSnapshot, { kind: 'asset', id: fixtureSnapshot.assets[0].id })?.label).toBe('MAVLink system 42');
    expect(resolveEntity(fixtureSnapshot, { kind: 'task', id: fixtureSnapshot.tasks[0].id })?.label).toBe('North Gate');
    expect(resolveEntity(fixtureSnapshot, { kind: 'advisory', id: fixtureSnapshot.advisories[0].id })?.label).toBe(
      'GeoChat ANDROID-ALPHA'
    );
    expect(resolveEntity(fixtureSnapshot, { kind: 'hazard', id: fixtureSnapshot.hazards[0].id })?.label).toBe(
      'Flood watch sector'
    );
    expect(
      resolveEntity(fixtureSnapshot, { kind: 'sensor-footprint', id: fixtureSnapshot.sensor_footprints[0].id })?.label
    ).toBe('TEST-UAS-01 sensor footprint');
    expect(
      resolveEntity(fixtureSnapshot, { kind: 'weather-observation', id: fixtureSnapshot.weather_observations[0].id })
        ?.label
    ).toBe('29.4 degC temperature_2m');
    expect(resolveEntity(fixtureSnapshot, { kind: 'association', id: fixtureSnapshot.associations[0].id })?.label).toBe(
      'Track association UAS 42 -> N42CX ambiguous'
    );
    expect(resolveEntity(fixtureSnapshot, { kind: 'alert', id: fixtureSnapshot.alerts[0].id })?.label).toBe(
      'Track freshness nominal'
    );
  });

  it('preserves a valid selection after snapshot refresh', () => {
    const selected = { kind: 'hazard' as const, id: fixtureSnapshot.hazards[0].id };

    expect(reconcileSelection(fixtureSnapshot, selected)).toEqual(selected);
  });

  it('uses an alert target as the effective map selection without replacing the alert inspector selection', () => {
    const alert = fixtureSnapshot.alerts[0];

    expect(resolveEntity(fixtureSnapshot, { kind: 'alert', id: alert.id })?.label).toBe(alert.label);
    expect(resolveMapSelection(fixtureSnapshot, { kind: 'alert', id: alert.id })).toEqual({
      kind: 'track',
      id: alert.entity_id
    });
  });

  it('leaves source-health alerts without spatial targets unprojected onto the map', () => {
    const sourceHealthAlert = {
      ...fixtureSnapshot.alerts[0],
      id: 'alert.discovery.feed',
      entity_id: 'feed.mavlink'
    };

    expect(
      resolveMapSelection(
        { ...fixtureSnapshot, alerts: [sourceHealthAlert] },
        { kind: 'alert', id: sourceHealthAlert.id }
      )
    ).toBeUndefined();
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
      kind: 'task',
      id: fixtureSnapshot.tasks[0].id
    });
    expect(
      reconcileSelection(without(without(without(fixtureSnapshot, 'tracks'), 'assets'), 'tasks'), stale)
    ).toEqual({
      kind: 'advisory',
      id: fixtureSnapshot.advisories[0].id
    });
    expect(
      reconcileSelection(
        without(without(without(without(fixtureSnapshot, 'tracks'), 'assets'), 'tasks'), 'advisories'),
        stale
      )
    ).toEqual({
      kind: 'hazard',
      id: fixtureSnapshot.hazards[0].id
    });
    expect(
      reconcileSelection(
        without(
          without(without(without(without(fixtureSnapshot, 'tracks'), 'assets'), 'tasks'), 'advisories'),
          'hazards'
        ),
        stale
      )
    ).toEqual({
      kind: 'sensor-footprint',
      id: fixtureSnapshot.sensor_footprints[0].id
    });
    expect(
      reconcileSelection(
        without(
          without(
            without(without(without(without(fixtureSnapshot, 'tracks'), 'assets'), 'tasks'), 'advisories'),
            'hazards'
          ),
          'sensor_footprints'
        ),
        stale
      )
    ).toEqual({
      kind: 'weather-observation',
      id: fixtureSnapshot.weather_observations[0].id
    });
    expect(
      reconcileSelection(
        without(
          without(
            without(
              without(without(without(without(fixtureSnapshot, 'tracks'), 'assets'), 'tasks'), 'advisories'),
              'hazards'
            ),
            'sensor_footprints'
          ),
          'weather_observations'
        ),
        stale
      )
    ).toEqual({
      kind: 'association',
      id: fixtureSnapshot.associations[0].id
    });
    expect(
      reconcileSelection(
        without(
          without(
            without(
              without(
                without(without(without(without(fixtureSnapshot, 'tracks'), 'assets'), 'tasks'), 'advisories'),
                'hazards'
              ),
              'sensor_footprints'
            ),
            'weather_observations'
          ),
          'associations'
        ),
        stale
      )
    ).toEqual({
      kind: 'alert',
      id: fixtureSnapshot.alerts[0].id
    });
  });
});

function without<
  K extends keyof Pick<
    Snapshot,
    | 'tracks'
    | 'assets'
    | 'tasks'
    | 'advisories'
    | 'hazards'
    | 'sensor_footprints'
    | 'weather_observations'
    | 'associations'
    | 'alerts'
  >
>(
  snapshot: Snapshot,
  key: K
): Snapshot {
  return { ...snapshot, [key]: [] };
}
