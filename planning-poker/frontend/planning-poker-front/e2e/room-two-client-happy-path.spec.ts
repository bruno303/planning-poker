import { expect, test } from '@playwright/test';

const roomUrlPattern = /\/room\/([0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12})$/i;
const userA = 'User A';
const userB = 'User B';

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

test('allows two users to join, vote, and sync story updates', async ({ browser, baseURL }) => {
  test.setTimeout(60_000);

  const ownerContext = await browser.newContext();
  const guestContext = await browser.newContext();

  await ownerContext.addInitScript(randomUUIDPolyfillScript);
  await guestContext.addInitScript(randomUUIDPolyfillScript);

  const ownerPage = await ownerContext.newPage();
  const guestPage = await guestContext.newPage();

  try {
    await ownerPage.goto('/join');
    await ownerPage.getByPlaceholder('Enter your name').fill(userA);
    const createRoomButton = ownerPage.getByRole('button', { name: 'Create Room' });
    await expect(createRoomButton).toBeEnabled();
    await Promise.all([
      ownerPage.waitForURL(roomUrlPattern),
      createRoomButton.click(),
    ]);

    const ownerRoomUrl = new URL(ownerPage.url(), baseURL);
    const roomPath = ownerRoomUrl.pathname;
    const roomMatch = roomPath.match(roomUrlPattern);

    expect(roomMatch).not.toBeNull();

    const roomId = roomMatch?.[1] ?? '';

    await guestPage.goto(`/join/${roomId}`);
    await guestPage.getByPlaceholder('Enter your name').fill(userB);
    await Promise.all([
      guestPage.waitForURL(new RegExp(`/room/${roomId}$`, 'i')),
      guestPage.getByRole('button', { name: 'Join Room' }).click(),
    ]);

    const ownerParticipantsPanel = ownerPage
      .getByRole('heading', { name: 'Participants' })
      .locator('xpath=ancestor::div[2]');
    const guestParticipantsPanel = guestPage
      .getByRole('heading', { name: 'Participants' })
      .locator('xpath=ancestor::div[2]');

    await expect(ownerParticipantsPanel.getByText(userA, { exact: true })).toBeVisible();
    await expect(ownerParticipantsPanel.getByText(userB, { exact: true })).toBeVisible();
    await expect(guestParticipantsPanel.getByText(userA, { exact: true })).toBeVisible();
    await expect(guestParticipantsPanel.getByText(userB, { exact: true })).toBeVisible();

    await ownerPage.getByRole('button', { name: '5', exact: true }).click();
    await guestPage.getByRole('button', { name: '8', exact: true }).click();

    await expect(ownerPage.getByText('2/2', { exact: true })).toBeVisible();
    await expect(guestPage.getByText('2/2', { exact: true })).toBeVisible();

    await expect(ownerPage.getByText('Results Summary')).toBeVisible();
    await expect(guestPage.getByText('Results Summary')).toBeVisible();
    await expect(ownerPage.getByText('Average: 6.5')).toBeVisible();
    await expect(guestPage.getByText('Average: 6.5')).toBeVisible();

    const updatedStory = `Story: estimate websocket sync ${Date.now()}`;

    const ownerEditStoryButton = ownerPage.getByRole('button', { name: 'Edit' });
    const guestEditStoryButton = guestPage.getByRole('button', { name: 'Edit' });

    await expect
      .poll(async () => (await ownerEditStoryButton.count()) + (await guestEditStoryButton.count()))
      .toBe(1);

    const adminPage = (await ownerEditStoryButton.count()) === 1 ? ownerPage : guestPage;

    await adminPage.getByRole('button', { name: 'Edit' }).click();
    await adminPage.getByRole('textbox').fill(updatedStory);
    await adminPage.getByRole('button', { name: 'Save' }).click();

    await expect(ownerPage.getByText(updatedStory, { exact: true })).toBeVisible();
    await expect(guestPage.getByText(updatedStory, { exact: true })).toBeVisible();
  } finally {
    await Promise.allSettled([ownerContext.close(), guestContext.close()]);
  }
});
