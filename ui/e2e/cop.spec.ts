import { expect, test } from '@playwright/test';
import type { Page } from '@playwright/test';
import { fixtureSnapshot } from '../src/lib/cop/fixture';
import type { RuntimeSnapshot, Snapshot } from '../src/lib/cop/types';

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

const runtimeSnapshot: RuntimeSnapshot = {
  generated_at: '2026-06-21T16:21:00Z',
  feeds: [
    {
      id: 'feed.mavlink',
      name: 'MAVLink',
      status: 'flowing',
      message: 'component flow active',
      healthy_components: 3,
      total_components: 3,
      messages_per_second: 8.5,
      last_activity: '2026-06-21T16:20:58Z',
      last_activity_age_seconds: 2
    },
    {
      id: 'feed.tak',
      name: 'TAK/CoT',
      status: 'flowing',
      message: 'component flow active',
      healthy_components: 3,
      total_components: 3,
      messages_per_second: 2,
      last_activity: '2026-06-21T16:20:55Z',
      last_activity_age_seconds: 5
    },
    {
      id: 'feed.adsb',
      name: 'ADS-B',
      status: 'flowing',
      message: 'component flow active',
      healthy_components: 3,
      total_components: 3,
      messages_per_second: 4.5,
      last_activity: '2026-06-21T16:20:59Z',
      last_activity_age_seconds: 1
    },
    {
      id: 'feed.sapient',
      name: 'SAPIENT',
      status: 'idle',
      message: 'components healthy; no recent flow',
      healthy_components: 2,
      total_components: 2,
      messages_per_second: 0
    }
  ],
  components: []
};

async function routeCOPState(page: Page) {
  let snapshotRequests = 0;
  await page.route('/api/cop/snapshot', async (route) => {
    snapshotRequests += 1;
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(snapshotWithADSB)
    });
  });
  await page.route('/api/cop/runtime', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(runtimeSnapshot)
    });
  });
  return {
    snapshotRequests: () => snapshotRequests
  };
}

async function expectNoHorizontalOverflow(page: Page) {
  await expect
    .poll(() =>
      page.evaluate(() => {
        const root = document.documentElement;
        return root.scrollWidth <= root.clientWidth + 1;
      })
    )
    .toBe(true);
}

test('renders API-backed COP state with ADS-B discovery and selection', async ({ page }) => {
  const routes = await routeCOPState(page);

  await page.goto('/');

  await expect(page.getByRole('heading', { name: 'Common Operating Picture' })).toBeVisible();
  await expect(page.getByText(/api\s+\d+[smhd]\s+snapshot/)).toBeVisible();
  await expect(page.getByLabel('ADS-B source state')).toBeVisible();
  await expect(page.getByText('OpenSky-compatible fixture via SemStreams component flow')).toBeVisible();
  await expect(page.getByLabel('ADS-B discovery counts')).toContainText('track 1');
  await expect(page.getByLabel('ADS-B runtime flow')).toContainText('4.5 msg/s');
  await expect(page.getByLabel('ADS-B runtime flow')).toContainText('3/3 healthy');
  await expect(page.getByLabel('SAPIENT source state')).toBeVisible();
  await expect(page.getByLabel('SAPIENT runtime flow')).toContainText('2/2 healthy');
  await expect(page.getByRole('button', { name: 'Select N123AB' })).toBeVisible();

  await page.getByRole('button', { name: 'Select N123AB' }).click();
  await expect(page.getByRole('heading', { name: 'N123AB' })).toBeVisible();
  await expect(page.getByText('semops.feed.adsb')).toBeVisible();
  await expect(page.getByText('adsb://opensky/a1b2c3/2026-06-21T16:20:00Z')).toBeVisible();

  await page.getByRole('button', { name: 'Refresh COP snapshot' }).click();
  await expect.poll(routes.snapshotRequests).toBeGreaterThanOrEqual(2);
});

test('keeps core operator loop accessible in a narrow viewport', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 900 });
  await routeCOPState(page);

  await page.goto('/');

  await expect(page.getByRole('heading', { name: 'Common Operating Picture' })).toBeVisible();
  await expect(page.getByLabel('Tactical map')).toBeVisible();
  await expect(page.getByLabel('Map entities')).toBeVisible();
  await expect(page.getByLabel('ADS-B source state')).toBeVisible();
  await expect(page.getByLabel('SAPIENT source state')).toBeVisible();
  await expectNoHorizontalOverflow(page);

  const aircraftButton = page.getByRole('button', { name: 'Select N123AB' });
  await aircraftButton.focus();
  await expect(aircraftButton).toBeFocused();
  await page.keyboard.press('Enter');
  await expect(page.getByRole('heading', { name: 'N123AB' })).toBeVisible();
  await expect(page.getByLabel('Entity inspector')).toContainText('semops.feed.adsb');

  const alertButton = page.getByRole('button', { name: /Track freshness nominal/ });
  await alertButton.focus();
  await expect(alertButton).toBeFocused();
  await page.keyboard.press('Enter');
  await expect(page.getByRole('heading', { name: 'Track freshness nominal' })).toBeVisible();
  await expectNoHorizontalOverflow(page);
});
