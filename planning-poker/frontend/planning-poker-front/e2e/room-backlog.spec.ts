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

const addStoryToBacklog = async (adminPage: Page, story: string) => {
  await adminPage.getByPlaceholder('Enter story name...').fill(story);
  await adminPage.getByRole('button', { name: 'Add' }).click();
};

const backlogPanel = (page: Page) =>
  page
    .getByRole('heading', { name: 'Story Backlog' })
    .locator('xpath=ancestor::div[2]');

test('backlog: adding, navigating, and voting on multiple stories', async ({ browser, baseURL }) => {
  test.setTimeout(60_000);

  const ownerContext = await browser.newContext();
  const guestContext = await browser.newContext();

  await ownerContext.addInitScript(randomUUIDPolyfillScript);
  await guestContext.addInitScript(randomUUIDPolyfillScript);

  const ownerPage = await ownerContext.newPage();
  const guestPage = await guestContext.newPage();

  try {
    // 1. Create room and join guest
    const roomId = await createRoom(ownerPage, baseURL, ownerName);
    await joinRoom(guestPage, roomId, guestName);

    // Verify participants
    const ownerPanel = participantsPanel(ownerPage);
    await expect(ownerPanel.getByText(ownerName, { exact: true })).toBeVisible();
    await expect(ownerPanel.getByText(guestName, { exact: true })).toBeVisible();

    // 2. Backlog panel should be visible by default
    await expect(ownerPage.getByText('Story Backlog')).toBeVisible();

    // 3. Add multiple stories to the backlog
    await addStoryToBacklog(ownerPage, 'Story Alpha');
    await addStoryToBacklog(ownerPage, 'Story Beta');
    await addStoryToBacklog(ownerPage, 'Story Gamma');

    // Verify all stories are visible in the backlog panel (both pages)
    const ownerBacklog = backlogPanel(ownerPage);
    const guestBacklog = backlogPanel(guestPage);
    await expect(ownerBacklog.getByText('Story Alpha', { exact: true })).toBeVisible();
    await expect(ownerBacklog.getByText('Story Beta', { exact: true })).toBeVisible();
    await expect(ownerBacklog.getByText('Story Gamma', { exact: true })).toBeVisible();
    await expect(guestBacklog.getByText('Story Alpha', { exact: true })).toBeVisible();
    await expect(guestBacklog.getByText('Story Beta', { exact: true })).toBeVisible();
    await expect(guestBacklog.getByText('Story Gamma', { exact: true })).toBeVisible();

    // 4. Verify story position indicator
    await expect(ownerPage.getByText('(Story 1 of 3)')).toBeVisible();

    // 5. Vote on first story
    await ownerPage.getByRole('button', { name: '5', exact: true }).click();
    await guestPage.getByRole('button', { name: '8', exact: true }).click();

    await expect(ownerPage.getByText('2/2', { exact: true })).toBeVisible();
    await expect(ownerPage.getByText('Results Summary')).toBeVisible();
    await expect(ownerPage.getByText('Average: 6.5')).toBeVisible();

    // 6. Advance to next story (clears votes/reveal state)
    await ownerPage.getByRole('button', { name: 'Next Story' }).click();

    // Should now be on Story 2 of 3
    await expect(ownerPage.getByText('(Story 2 of 3)')).toBeVisible();

    // 7. Vote on second story
    await ownerPage.getByRole('button', { name: '3', exact: true }).click();
    await guestPage.getByRole('button', { name: '3', exact: true }).click();

    await expect(ownerPage.getByText('2/2', { exact: true })).toBeVisible();
    await expect(ownerPage.getByText('Results Summary')).toBeVisible();
    await expect(ownerPage.getByText('Average: 3.0')).toBeVisible();

    // 8. Go back to previous story
    await ownerPage.getByRole('button', { name: 'Previous Story' }).click();

    // Should be back on Story 1 of 3
    await expect(ownerPage.getByText('(Story 1 of 3)')).toBeVisible();
  } finally {
    await Promise.allSettled([ownerContext.close(), guestContext.close()]);
  }
});

test('backlog: disable and re-enable backlog mode', async ({ browser, baseURL }) => {
  test.setTimeout(60_000);

  const ownerContext = await browser.newContext();

  await ownerContext.addInitScript(randomUUIDPolyfillScript);

  const ownerPage = await ownerContext.newPage();

  try {
    await createRoom(ownerPage, baseURL, ownerName);

    // Backlog panel should be visible by default
    await expect(ownerPage.getByText('Story Backlog')).toBeVisible();

    // Disable Backlog button should be present
    const disableButton = ownerPage.getByRole('button', { name: 'Disable Backlog' });
    await expect(disableButton).toBeVisible();

    // Click Disable Backlog - should show confirmation dialog
    await disableButton.click();

    // Confirmation dialog should appear
    await expect(ownerPage.getByText('This will remove the story backlog and keep only the current story. Are you sure?')).toBeVisible();

    // Click Cancel - backlog should remain enabled
    await ownerPage.getByRole('button', { name: 'Cancel' }).click();
    await expect(ownerPage.getByText('Story Backlog')).toBeVisible();

    // Click Disable Backlog again and confirm
    await disableButton.click();
    await ownerPage.getByRole('button', { name: 'Disable', exact: true }).click();

    // Backlog panel should be hidden
    await expect(ownerPage.getByText('Story Backlog')).not.toBeVisible();

    // Enable Backlog button should appear
    await expect(ownerPage.getByRole('button', { name: 'Enable Backlog' })).toBeVisible();
  } finally {
    await Promise.allSettled([ownerContext.close()]);
  }
});
