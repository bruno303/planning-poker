import { expect, test, type Page } from '@playwright/test';

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

const openBacklogModal = async (page: Page) => {
  await page.getByRole('button', { name: 'Open backlog' }).click();
};

const backlogDialog = (page: Page) => page.getByRole('dialog', { name: 'Story Backlog' });

const addStoryToBacklog = async (adminPage: Page, story: string) => {
  await openBacklogModal(adminPage);
  const dialog = backlogDialog(adminPage);
  await dialog.getByPlaceholder('Enter story name...').fill(story);
  await dialog.getByRole('button', { name: 'Add' }).click();
  // Wait for the WebSocket response before closing or opening the modal again.
  await expect(dialog.getByText(story, { exact: true })).toBeVisible();
  await dialog.getByRole('button', { name: 'Close' }).click();
};

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

    // 2. Open backlog button should be visible by default
    await expect(ownerPage.getByRole('button', { name: 'Open backlog' })).toBeVisible();

    // 3. Add multiple stories to the backlog
    await addStoryToBacklog(ownerPage, 'Story Alpha');
    await addStoryToBacklog(ownerPage, 'Story Beta');
    await addStoryToBacklog(ownerPage, 'Story Gamma');

    // Verify all stories are visible in the backlog dialog (owner page)
    await openBacklogModal(ownerPage);
    const ownerBacklog = backlogDialog(ownerPage);
    await expect(ownerBacklog.getByText('Story Alpha', { exact: true })).toBeVisible();
    await expect(ownerBacklog.getByText('Story Beta', { exact: true })).toBeVisible();
    await expect(ownerBacklog.getByText('Story Gamma', { exact: true })).toBeVisible();
    await ownerBacklog.getByRole('button', { name: 'Close' }).click();

    // Verify all stories are visible in the backlog dialog (guest page)
    await openBacklogModal(guestPage);
    const guestBacklog = backlogDialog(guestPage);
    await expect(guestBacklog.getByText('Story Alpha', { exact: true })).toBeVisible();
    await expect(guestBacklog.getByText('Story Beta', { exact: true })).toBeVisible();
    await expect(guestBacklog.getByText('Story Gamma', { exact: true })).toBeVisible();
    await guestBacklog.getByRole('button', { name: 'Close' }).click();

    // 4. Verify story position indicator
    await expect(ownerPage.getByText('(Story 1 of 3)')).toBeVisible();

    // Previous Story should be disabled at first story but still visible
    const prevStoryButton = ownerPage.getByRole('button', { name: 'Previous Story' });
    const nextStoryButton = ownerPage.getByRole('button', { name: 'Next Story' });
    await expect(prevStoryButton).toBeVisible();
    await expect(prevStoryButton).toBeDisabled();
    await expect(nextStoryButton).toBeVisible();
    await expect(nextStoryButton).toBeEnabled();

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

    // 8. Advance to last story
    await ownerPage.getByRole('button', { name: 'Next Story' }).click();

    // Should now be on Story 3 of 3
    await expect(ownerPage.getByText('(Story 3 of 3)')).toBeVisible();

    // Next Story should be disabled at last story but still visible
    await expect(ownerPage.getByRole('button', { name: 'Next Story' })).toBeVisible();
    await expect(ownerPage.getByRole('button', { name: 'Next Story' })).toBeDisabled();
    await expect(ownerPage.getByRole('button', { name: 'Previous Story' })).toBeEnabled();

    // 9. Navigate back to first story
    await ownerPage.getByRole('button', { name: 'Previous Story' }).click();
    await expect(ownerPage.getByText('(Story 2 of 3)')).toBeVisible();
    await ownerPage.getByRole('button', { name: 'Previous Story' }).click();

    // Should be back on Story 1 of 3
    await expect(ownerPage.getByText('(Story 1 of 3)')).toBeVisible();

    // Previous Story should be disabled at first story but still visible
    await expect(ownerPage.getByRole('button', { name: 'Previous Story' })).toBeVisible();
    await expect(ownerPage.getByRole('button', { name: 'Previous Story' })).toBeDisabled();
  } finally {
    await Promise.allSettled([ownerContext.close(), guestContext.close()]);
  }
});

test('backlog: does not expose a disable backlog control', async ({ browser, baseURL }) => {
  test.setTimeout(60_000);

  const ownerContext = await browser.newContext();

  await ownerContext.addInitScript(randomUUIDPolyfillScript);

  const ownerPage = await ownerContext.newPage();

  try {
    await createRoom(ownerPage, baseURL, ownerName);

    // Open backlog button should be visible by default
    await expect(ownerPage.getByRole('button', { name: 'Open backlog' })).toBeVisible();

    // Backlog is always enabled for new rooms.
    await openBacklogModal(ownerPage);

    // The modal must not expose a way to disable backlog mode.
    const disableButton = ownerPage.getByRole('button', { name: 'Disable Backlog' });
    await expect(disableButton).toHaveCount(0);
  } finally {
    await Promise.allSettled([ownerContext.close()]);
  }
});
