import { expect, test } from '@playwright/test';
import type { Page } from '@playwright/test';
import { fixtureSnapshot } from '../src/lib/cop/fixture';
import type { AssociationReview, RuntimeSnapshot, Snapshot } from '../src/lib/cop/types';

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
      },
      {
        org: 'c360',
        platform: 'edge-compose',
        source: 'command',
        family: 'command',
        entity_type: 'task',
        prefix: 'c360.edge-compose.cop.command.task',
        count: 1,
        limit: 500,
        at_limit: false
      },
      {
        org: 'c360',
        platform: 'edge-compose',
        source: 'weather',
        family: 'weather',
        entity_type: 'weather_observation',
        prefix: 'c360.edge-compose.cop.weather.weather_observation',
        count: 1,
        limit: 500,
        at_limit: false
      },
      {
        org: 'c360',
        platform: 'edge-compose',
        source: 'klv',
        family: 'klv',
        entity_type: 'sensor_footprint',
        prefix: 'c360.edge-compose.cop.klv.sensor_footprint',
        count: 1,
        limit: 500,
        at_limit: false
      },
      {
        org: 'c360',
        platform: 'edge-compose',
        source: 'fusion',
        family: 'fusion',
        entity_type: 'association',
        prefix: 'c360.edge-compose.cop.fusion.association',
        count: 1,
        limit: 500,
        at_limit: false
      }
    ]
  },
  feeds: fixtureSnapshot.feeds.map((feed) =>
    feed.id === 'feed.adsb'
      ? {
          ...feed,
          last_event_at: '2026-06-21T16:20:00Z',
          message: 'OpenSky-compatible fixture via SemStreams component flow'
        }
      : feed
  ),
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
      id: 'feed.klv',
      name: 'KLV',
      status: 'flowing',
      message: 'component flow active',
      healthy_components: 4,
      total_components: 4,
      messages_per_second: 1.25,
      last_activity: '2026-06-21T16:20:54Z',
      last_activity_age_seconds: 6
    },
    {
      id: 'feed.weather',
      name: 'Weather',
      status: 'flowing',
      message: 'fixture-backed point observation flow active',
      healthy_components: 3,
      total_components: 3,
      messages_per_second: 1,
      last_activity: '2026-06-21T16:20:52Z',
      last_activity_age_seconds: 8
    },
    {
      id: 'feed.fusion',
      name: 'Fusion',
      status: 'flowing',
      message: 'component flow active',
      healthy_components: 2,
      total_components: 2,
      messages_per_second: 0.5,
      last_activity: '2026-06-21T16:20:51Z',
      last_activity_age_seconds: 9
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
  let associationReview: AssociationReview | undefined;
  await page.route('/api/cop/snapshot', async (route) => {
    snapshotRequests += 1;
    const snapshot = associationReview
      ? {
          ...snapshotWithADSB,
          associations: snapshotWithADSB.associations.map((association) =>
            association.id === associationReview?.association_id
              ? { ...association, operator_review: associationReview }
              : association
          )
        }
      : snapshotWithADSB;
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(snapshot)
    });
  });
  await page.route(/\/api\/cop\/associations\/.+\/review$/, async (route) => {
    const body = (await route.request().postDataJSON()) as { decision: AssociationReview['decision']; reviewed_by?: string };
    const associationID = decodeURIComponent(route.request().url().split('/api/cop/associations/')[1].replace('/review', ''));
    associationReview = {
      association_id: associationID,
      decision: body.decision,
      reviewed_by: body.reviewed_by ?? 'operator.local',
      reviewed_at: '2026-06-21T16:21:10Z'
    };
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(associationReview)
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
  await expect(page.getByLabel('Command source state')).toBeVisible();
  await expect(page.getByLabel('Command discovery counts')).toContainText('task 1');
  const commandTaskRow = page.getByRole('button', { name: 'Route MAVLink system 42 to North Gate cancel_requested' });
  await expect(commandTaskRow).toBeVisible();
  await expect(page.getByLabel('KLV source state')).toBeVisible();
  await expect(page.getByLabel('KLV discovery counts')).toContainText('sensor footprint 1');
  await expect(page.getByLabel('KLV runtime flow')).toContainText('1.3 msg/s');
  await expect(page.getByRole('button', { name: 'Select TEST-UAS-01 sensor footprint' })).toBeVisible();
  await expect(page.getByLabel('Weather source state')).toBeVisible();
  await expect(page.getByLabel('Weather discovery counts')).toContainText('weather observation 1');
  await expect(page.getByLabel('Weather runtime flow')).toContainText('1 msg/s');
  await expect(page.getByRole('button', { name: 'Select 29.4 degC temperature_2m' })).toBeVisible();
  await expect(page.getByLabel('Fusion source state')).toBeVisible();
  await expect(page.getByLabel('Fusion discovery counts')).toContainText('association 1');
  await expect(page.getByLabel('Fusion runtime flow')).toContainText('0.5 msg/s');
  const associationRow = page.getByRole('button', { name: 'Inspect Ambiguous association UAS 42 -> N42CX' });
  await expect(associationRow).toBeVisible();
  await expect(associationRow).toContainText('ambiguous evidence');
  await expect(page.getByLabel('SAPIENT source state')).toBeVisible();
  await expect(page.getByLabel('SAPIENT runtime flow')).toContainText('2/2 healthy');
  await expect(page.getByRole('button', { name: 'Select N123AB' })).toBeVisible();

  await page.getByRole('button', { name: 'Select N123AB' }).click();
  await expect(page.getByRole('heading', { name: 'N123AB' })).toBeVisible();
  await expect(page.getByText('semops.feed.adsb')).toBeVisible();
  await expect(page.getByText('adsb://opensky/a1b2c3/2026-06-21T16:20:00Z')).toBeVisible();

  await commandTaskRow.click();
  await expect(page.getByRole('heading', { name: 'Route MAVLink system 42 to North Gate' })).toBeVisible();
  await expect(page.getByText('semops.command.intent')).toBeVisible();
  await expect(page.getByText('command://fixture/hadr-command/0004-route-cancel-requested')).toBeVisible();
  await expect(page.getByText('ui:cancel-route-42')).toBeVisible();

  await page.getByRole('button', { name: 'Select TEST-UAS-01 sensor footprint' }).click();
  await expect(page.getByRole('heading', { name: 'TEST-UAS-01 sensor footprint' })).toBeVisible();
  await expect(page.getByText('object://semops/klv/deterministic-001.ts')).toBeVisible();
  await expect(page.getByText('klv://packet/deterministic/00000001').first()).toBeVisible();
  await expect(page.getByText(/no footprint polygon/)).toBeVisible();

  await page.getByRole('button', { name: 'Select 29.4 degC temperature_2m' }).click();
  await expect(page.getByRole('heading', { name: '29.4 degC temperature_2m' })).toBeVisible();
  await expect(page.getByText('open-meteo', { exact: true })).toBeVisible();
  await expect(page.getByText('POINT(-77.0400000 38.9000000)')).toBeVisible();
  await expect(page.getByText(/no live provider/)).toBeVisible();

  await associationRow.click();
  await expect(page.getByRole('heading', { name: 'Ambiguous association UAS 42 -> N42CX' })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Association Evidence' })).toBeVisible();
  await expect(page.getByLabel('Entity inspector')).toContainText('ambiguous evidence');
  await expect(page.getByText('c360.edge.cop.mavlink.track.system-42')).toBeVisible();
  await expect(page.getByText('c360.edge.cop.adsb.track.a1b2c3')).toBeVisible();
  await expect(page.getByText('semops.association.geotemporal.v1')).toBeVisible();
  await expect(page.getByText(/no source-track merge/)).toBeVisible();
  await expect(page.getByText('semops.fusion.structural')).toBeVisible();
  await expect(page.getByLabel('Entity inspector')).toContainText('unreviewed');
  await page.getByRole('button', { name: 'Acknowledge association evidence' }).click();
  await expect(page.getByLabel('Entity inspector')).toContainText('acknowledged');
  await expect(page.getByText('operator.local')).toBeVisible();
  await page.getByRole('button', { name: 'Challenge association evidence' }).click();
  await expect(page.getByLabel('Entity inspector')).toContainText('challenged');

  await page.getByRole('button', { name: 'Refresh COP snapshot' }).click();
  await expect.poll(routes.snapshotRequests).toBeGreaterThanOrEqual(2);
  await expect(page.getByLabel('Entity inspector')).toContainText('challenged');
});

test('keeps core operator loop accessible in a narrow viewport', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 900 });
  await routeCOPState(page);

  await page.goto('/');

  await expect(page.getByRole('heading', { name: 'Common Operating Picture' })).toBeVisible();
  await expect(page.getByLabel('Tactical map')).toBeVisible();
  await expect(page.getByLabel('Map entities')).toBeVisible();
  await expect(page.getByLabel('ADS-B source state')).toBeVisible();
  await expect(page.getByLabel('KLV source state')).toBeVisible();
  await expect(page.getByLabel('Weather source state')).toBeVisible();
  await expect(page.getByLabel('Fusion source state')).toBeVisible();
  await expect(page.getByLabel('SAPIENT source state')).toBeVisible();
  await expectNoHorizontalOverflow(page);

  const aircraftButton = page.getByRole('button', { name: 'Select N123AB' });
  await aircraftButton.focus();
  await expect(aircraftButton).toBeFocused();
  await page.keyboard.press('Enter');
  await expect(page.getByRole('heading', { name: 'N123AB' })).toBeVisible();
  await expect(page.getByLabel('Entity inspector')).toContainText('semops.feed.adsb');

  const klvButton = page.getByRole('button', { name: 'Select TEST-UAS-01 sensor footprint' });
  await klvButton.focus();
  await expect(klvButton).toBeFocused();
  await page.keyboard.press('Enter');
  await expect(page.getByRole('heading', { name: 'TEST-UAS-01 sensor footprint' })).toBeVisible();
  await expect(page.getByLabel('Entity inspector')).toContainText('no STANAG conformance');

  const weatherButton = page.getByRole('button', { name: 'Select 29.4 degC temperature_2m' });
  await weatherButton.focus();
  await expect(weatherButton).toBeFocused();
  await page.keyboard.press('Enter');
  await expect(page.getByRole('heading', { name: '29.4 degC temperature_2m' })).toBeVisible();
  await expect(page.getByLabel('Entity inspector')).toContainText('no live provider');

  const associationButton = page.getByRole('button', {
    name: 'Inspect Ambiguous association UAS 42 -> N42CX'
  });
  await associationButton.focus();
  await expect(associationButton).toBeFocused();
  await page.keyboard.press('Enter');
  await expect(page.getByRole('heading', { name: 'Ambiguous association UAS 42 -> N42CX' })).toBeVisible();
  await expect(page.getByLabel('Entity inspector')).toContainText('no source-track merge');

  const alertButton = page.getByRole('button', { name: /Track freshness nominal/ });
  await alertButton.focus();
  await expect(alertButton).toBeFocused();
  await page.keyboard.press('Enter');
  await expect(page.getByRole('heading', { name: 'Track freshness nominal' })).toBeVisible();
  await expect(page.getByText('c360.edge.cop.mavlink.track.system-42')).toBeVisible();
  await expect(alertButton).toHaveAttribute('aria-pressed', 'true');
  await expect(page.getByRole('button', { name: 'Select UAS 42' })).toHaveAttribute('aria-pressed', 'true');
  await expectNoHorizontalOverflow(page);
});
