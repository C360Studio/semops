import { expect, test } from '@playwright/test';
import { fixtureSnapshot } from '../src/lib/cop/fixture';
import type { Snapshot } from '../src/lib/cop/types';

const adsbTrack = {
  id: 'c360.edge-compose.cop.adsb.track.a1b2c3',
  label: 'N123AB',
  source: 'adsb',
  status: 'airborne',
  position: { lat: 38.901, lon: -77.041 },
  velocity: '75 m/s @ 181 deg',
  confidence: 0.82,
  updated_at: '2026-06-21T16:20:00Z',
  provenance: {
    owner: 'semops.feed.adsb',
    source_ref: 'adsb://opensky/a1b2c3/2026-06-21T16:20:00Z',
    observed_at: '2026-06-21T16:20:00Z'
  }
};

const snapshotWithADSB: Snapshot = {
  ...fixtureSnapshot,
  generated_at: '2026-06-21T16:21:00Z',
  scenario: 'phase-1-live-graph',
  summary: {
    ...fixtureSnapshot.summary,
    active_tracks: fixtureSnapshot.summary.active_tracks + 1
  },
  diagnostics: {
    discovery: [
      ...(fixtureSnapshot.diagnostics?.discovery ?? []),
      {
        org: 'c360',
        platform: 'edge-compose',
        source: 'adsb',
        family: 'adsb',
        entity_type: 'track',
        prefix: 'c360.edge-compose.cop.adsb.track',
        count: 1,
        limit: 500,
        at_limit: false
      }
    ]
  },
  feeds: [
    ...fixtureSnapshot.feeds,
    {
      id: 'feed.adsb',
      name: 'ADS-B',
      kind: 'air-picture',
      status: 'live',
      last_event_at: '2026-06-21T16:20:00Z',
      message: 'OpenSky-compatible fixture via SemStreams component flow'
    }
  ],
  tracks: [...fixtureSnapshot.tracks, adsbTrack]
};

test('renders API-backed COP state with ADS-B discovery and selection', async ({ page }) => {
  let snapshotRequests = 0;
  await page.route('/api/cop/snapshot', async (route) => {
    snapshotRequests += 1;
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(snapshotWithADSB)
    });
  });

  await page.goto('/');

  await expect(page.getByRole('heading', { name: 'Common Operating Picture' })).toBeVisible();
  await expect(page.getByText(/api\s+\d+[smhd]\s+snapshot/)).toBeVisible();
  await expect(page.getByText('ADS-B')).toBeVisible();
  await expect(page.getByText('OpenSky-compatible fixture via SemStreams component flow')).toBeVisible();
  await expect(page.getByLabel('ADS-B discovery counts')).toContainText('track 1');
  await expect(page.getByRole('button', { name: 'Select N123AB' })).toBeVisible();

  await page.getByRole('button', { name: 'Select N123AB' }).click();
  await expect(page.getByRole('heading', { name: 'N123AB' })).toBeVisible();
  await expect(page.getByText('semops.feed.adsb')).toBeVisible();
  await expect(page.getByText('adsb://opensky/a1b2c3/2026-06-21T16:20:00Z')).toBeVisible();

  await page.getByRole('button', { name: 'Refresh COP snapshot' }).click();
  await expect.poll(() => snapshotRequests).toBeGreaterThanOrEqual(2);
});
