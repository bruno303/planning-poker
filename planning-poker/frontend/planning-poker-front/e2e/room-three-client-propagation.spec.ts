import { expect, Page, test } from '@playwright/test';

const roomUrlPattern = /\/room\/([0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12})$/i;

const ownerName = 'User A';
const guestBName = 'User B';
const guestCName = 'User C';

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

test('propagates participants and reveal state consistently across three clients', async ({ browser, baseURL }) => {
  test.setTimeout(60_000);

  const ownerContext = await browser.newContext();
  const guestBContext = await browser.newContext();
  const guestCContext = await browser.newContext();

  await ownerContext.addInitScript(randomUUIDPolyfillScript);
  await guestBContext.addInitScript(randomUUIDPolyfillScript);
  await guestCContext.addInitScript(randomUUIDPolyfillScript);

  const ownerPage = await ownerContext.newPage();
  const guestBPage = await guestBContext.newPage();
  const guestCPage = await guestCContext.newPage();

  try {
    const roomId = await createRoom(ownerPage, baseURL, ownerName);

    await joinRoom(guestBPage, roomId, guestBName);
    await joinRoom(guestCPage, roomId, guestCName);

    const pages = [ownerPage, guestBPage, guestCPage];
    const participants = [ownerName, guestBName, guestCName];

    for (const page of pages) {
      const panel = participantsPanel(page);
      for (const participant of participants) {
        await expect(panel.getByText(participant, { exact: true })).toBeVisible();
      }
    }

    await ownerPage.getByRole('button', { name: '3', exact: true }).click();
    await guestBPage.getByRole('button', { name: '5', exact: true }).click();
    await guestCPage.getByRole('button', { name: '8', exact: true }).click();

    for (const page of pages) {
      await expect(page.getByText('3/3', { exact: true })).toBeVisible();
    }

    for (const page of pages) {
      await expect(page.getByText('Results Summary')).toBeVisible();
      await expect(page.getByText('Average: 5.3')).toBeVisible();
      await expect(page.getByText('Most Common')).toBeVisible();
      await expect(page.getByText('3/3', { exact: true })).toBeVisible();
    }
  } finally {
    await Promise.allSettled([ownerContext.close(), guestBContext.close(), guestCContext.close()]);
  }
});
