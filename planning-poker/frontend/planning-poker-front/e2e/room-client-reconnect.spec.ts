import { expect, Page, test } from '@playwright/test';

const roomUrlPattern = /\/room\/([0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12})$/i;

const ownerName = 'User A';
const guestName = 'User B';

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

const participantsPanel = (page: Page) =>
  page
    .getByRole('heading', { name: 'Participants' })
    .locator('xpath=ancestor::div[2]');

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

const joinRoom = async (page: Page, roomId: string, userName: string) => {
  await page.goto(`/join/${roomId}`);
  await page.getByPlaceholder('Enter your name').fill(userName);

  const joinRoomButton = page.getByRole('button', { name: 'Join Room' });
  await expect(joinRoomButton).toBeEnabled();

  await Promise.all([
    page.waitForURL(new RegExp(`/room/${roomId}$`, 'i')),
    joinRoomButton.click(),
  ]);
};

test('client auto-reconnects after WebSocket disconnect from network', async ({ browser, baseURL }) => {
  test.setTimeout(60_000);

  const ownerContext = await browser.newContext();
  const guestContext = await browser.newContext();

  await ownerContext.addInitScript(randomUUIDPolyfillScript);
  await guestContext.addInitScript(randomUUIDPolyfillScript);

  const ownerPage = await ownerContext.newPage();
  const guestPage = await guestContext.newPage();

  try {
    // 1. Owner creates room
    const roomId = await createRoom(ownerPage, baseURL, ownerName);

    // 2. Guest joins room
    await joinRoom(guestPage, roomId, guestName);

    // 3. Both see each other in the participant list
    const ownerParticipants = participantsPanel(ownerPage);
    await expect(ownerParticipants.getByText(ownerName, { exact: true })).toBeVisible();
    await expect(ownerParticipants.getByText(guestName, { exact: true })).toBeVisible();

    // 4. Simulate network disconnect — close the WebSocket from browser side
    await guestPage.evaluate(() => {
      const ws = (window as any).__ws;
      if (ws && ws.readyState === WebSocket.OPEN) {
        ws.close();
      }
    });

    // 5. Client auto-reconnects — expect the disconnected banner
    await expect(guestPage.getByText('Connection lost. Reconnecting...')).toBeVisible({ timeout: 5000 });

    // 6. Wait for auto-reconnect — owner sees guest again (new client ID)
    await expect(ownerParticipants.getByText(guestName, { exact: true })).toBeVisible({ timeout: 15000 });

    // 7. Banner should be gone after reconnection
    await expect(guestPage.getByText('Connection lost. Reconnecting...')).not.toBeVisible({ timeout: 10000 });
  } finally {
    await Promise.allSettled([ownerContext.close(), guestContext.close()]);
  }
});
