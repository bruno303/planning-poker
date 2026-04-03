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

const participantRow = (page: Page, participantName: string) =>
  participantsPanel(page)
    .locator('div')
    .filter({ has: page.getByText(participantName, { exact: true }) })
    .first();

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

const expectAdminControlsVisible = async (page: Page) => {
  await expect(page.getByRole('button', { name: 'Edit' })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Reveal Votes' })).toBeVisible();
  await expect(page.getByRole('button', { name: 'New Voting' })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Vote Again' })).toBeVisible();
};

const expectParticipantAdminTogglesVisible = async (page: Page, participantName: string) => {
  const row = participantRow(page, participantName);
  await expect(row.locator('button[title="Make Spectator"]')).toBeVisible();
  await expect(row.locator('button[title="Remove Admin"]')).toBeVisible();
};

test('keeps room interactive for remaining client after peer page closes', async ({ browser, baseURL }) => {
  test.setTimeout(60_000);

  const ownerContext = await browser.newContext();
  const guestContext = await browser.newContext();

  await ownerContext.addInitScript(randomUUIDPolyfillScript);
  await guestContext.addInitScript(randomUUIDPolyfillScript);

  const ownerPage = await ownerContext.newPage();
  const guestPage = await guestContext.newPage();

  try {
    const roomId = await createRoom(ownerPage, baseURL, ownerName);
    await joinRoom(guestPage, roomId, guestName);

    const guestParticipants = participantsPanel(guestPage);

    await expect(guestParticipants.getByText(ownerName, { exact: true })).toBeVisible();
    await expect(guestParticipants.getByText(guestName, { exact: true })).toBeVisible();

    await ownerPage.close();

    await expect(guestParticipants.getByText(ownerName, { exact: true })).toHaveCount(0);
    await expectAdminControlsVisible(guestPage);
    await expectParticipantAdminTogglesVisible(guestPage, guestName);

    await guestPage.getByRole('button', { name: '5', exact: true }).click();

    await expect(guestPage.getByText('1/1', { exact: true })).toBeVisible();
    await expect(guestPage.getByText('Results Summary')).toBeVisible();
  } finally {
    await Promise.allSettled([ownerContext.close(), guestContext.close()]);
  }
});
