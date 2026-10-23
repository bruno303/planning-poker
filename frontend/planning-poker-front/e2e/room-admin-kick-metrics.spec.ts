import { APIRequestContext, expect, Page, test } from '@playwright/test';

const roomUrlPattern = /\/room\/([0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12})$/i;
const uuidPattern = /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i;

const ownerName = 'Owner';

const backendUrl = process.env.E2E_BACKEND_URL ?? 'http://backend:8080';
const metricsUrl = process.env.E2E_METRICS_URL ?? 'http://backend:9090/metrics';
const adminApiKey = process.env.E2E_ADMIN_API_KEY ?? 'my-secret-key';

const activeUsersMetric = 'planning_poker_active_users';
const activeRoomsMetric = 'planning_poker_active_rooms';

type MetricTotals = {
  activeUsers: number;
  activeRooms: number;
};

const randomUUIDPolyfillScript = () => {
  const buildRandomUUID = () => {
    const bytes = crypto.getRandomValues(new Uint8Array(16));
    bytes[6] = (bytes[6] & 0x0f) | 0x40;
    bytes[8] = (bytes[8] & 0x3f) | 0x80;

    const hex = Array.from(bytes, (byte) => byte.toString(16).padStart(2, '0')).join('');
    return `${hex.slice(0, 8)}-${hex.slice(8, 12)}-${hex.slice(12, 16)}-${hex.slice(16, 20)}-${hex.slice(20, 32)}`;
  };

  if (typeof globalThis.crypto?.randomUUID !== 'function' && globalThis.crypto) {
    Object.defineProperty(globalThis.crypto, 'randomUUID', {
      value: buildRandomUUID,
      configurable: true,
      writable: true,
    });
  }
};

// The counters are OpenTelemetry up-down counters, so a negative sample is a
// real exported value.
const parseMetricTotals = (body: string): MetricTotals => {
  const totals: MetricTotals = { activeUsers: 0, activeRooms: 0 };

  for (const line of body.split('\n')) {
    if (line.startsWith('#')) continue;

    const match = line.match(/^([^ {]+)(?:\{[^}]*\})?\s+(\S+)$/);
    if (!match) continue;

    const name = match[1];
    const rawValue = match[2];
    if (name === undefined || rawValue === undefined) continue;

    const value = Number(rawValue);
    if (!Number.isFinite(value)) continue;

    if (name === activeUsersMetric) totals.activeUsers += value;
    if (name === activeRoomsMetric) totals.activeRooms += value;
  }

  return totals;
};

const readMetricTotals = async (request: APIRequestContext): Promise<MetricTotals> => {
  const response = await request.get(metricsUrl);
  expect(response.ok(), `expected ${metricsUrl} to be scrapable`).toBe(true);

  return parseMetricTotals(await response.text());
};

const readClientId = async (page: Page): Promise<string> => {
  const read = () =>
    page.evaluate(
      () => (window as unknown as { planning_poker?: { clientID?: string } }).planning_poker?.clientID,
    );

  await expect.poll(read).toMatch(uuidPattern);

  const clientId = await read();
  if (!clientId) throw new Error('client id was not exposed on window.planning_poker');

  return clientId;
};

const createRoom = async (page: Page, baseURL: string | undefined, userName: string) => {
  await page.goto('/join');
  await page.getByPlaceholder('Enter your name').fill(userName);

  const createRoomButton = page.getByRole('button', { name: 'Create Room' });
  await expect(createRoomButton).toBeEnabled();

  await Promise.all([
    page.waitForURL(roomUrlPattern),
    createRoomButton.click(),
  ]);

  const roomPath = new URL(page.url(), baseURL).pathname;
  const roomMatch = roomPath.match(roomUrlPattern);

  expect(roomMatch).not.toBeNull();
  return roomMatch?.[1] ?? '';
};

test('admin kick never drives the active metrics negative', async ({ browser, baseURL, request }) => {
  test.setTimeout(60_000);

  const ownerContext = await browser.newContext();
  await ownerContext.addInitScript(randomUUIDPolyfillScript);
  const ownerPage = await ownerContext.newPage();

  try {
    const roomId = await createRoom(ownerPage, baseURL, ownerName);
    const clientId = await readClientId(ownerPage);

    // The gauge is global and shared with every other spec, and a client can
    // legitimately re-join while its previous socket's leave is still in
    // flight, so the value before the kick is recorded for diagnostics only.
    const before = await readMetricTotals(request);

    const response = await request.post(`${backendUrl}/admin/rooms/${roomId}/client/${clientId}/kick`, {
      headers: { Authorization: `Bearer ${adminApiKey}` },
    });
    expect(response.status()).toBe(200);

    await expect(ownerPage.getByText('You have been kicked from the room')).toBeVisible();

    // Regression guard: before the bus was detached before closing, a single
    // kick ran LeaveRoom twice, so this assertion saw -1 active users and -1
    // active rooms.
    const totals = await readMetricTotals(request);
    expect(
      totals.activeUsers,
      `${activeUsersMetric} must never be negative (before kick: ${before.activeUsers}, after kick: ${totals.activeUsers})`,
    ).toBeGreaterThanOrEqual(0);
    expect(
      totals.activeRooms,
      `${activeRoomsMetric} must never be negative (before kick: ${before.activeRooms}, after kick: ${totals.activeRooms})`,
    ).toBeGreaterThanOrEqual(0);
  } finally {
    await ownerContext.close();
  }
});
