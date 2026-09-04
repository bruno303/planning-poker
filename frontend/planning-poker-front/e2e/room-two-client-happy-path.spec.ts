import { expect, test, type Locator, type Page } from '@playwright/test';

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

const backlogDialog = (page: Page) => page.getByRole('dialog', { name: 'Story Backlog' });
const roomClock = (page: Page) => page.getByRole('timer', { name: 'Elapsed room time' });

const openBacklogModal = async (page: Page) => {
  await page.getByRole('button', { name: 'Open backlog' }).click();
};

const storyCard = (page: Page) => page.getByText('Current Story').locator('xpath=ancestor::div[2]');

const participantAvatar = (participantsPanel: Locator, participantName: string) =>
  participantsPanel.getByText(participantName, { exact: true }).locator('xpath=..').getByRole('img');

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

    await expect(roomClock(ownerPage)).toBeVisible();
    await expect(roomClock(guestPage)).toBeVisible();
    await expect.poll(async () => {
      const ownerReading = await roomClock(ownerPage).textContent();
      const guestReading = await roomClock(guestPage).textContent();
      return ownerReading === guestReading ? ownerReading : null;
    }).toMatch(/^(?:\d+:)?\d{2}:\d{2}$/);

    await expect(ownerParticipantsPanel.getByText(userA, { exact: true })).toBeVisible();
    await expect(ownerParticipantsPanel.getByText(userB, { exact: true })).toBeVisible();
    await expect(guestParticipantsPanel.getByText(userA, { exact: true })).toBeVisible();
    await expect(guestParticipantsPanel.getByText(userB, { exact: true })).toBeVisible();

    for (const participantName of [userA, userB]) {
      const ownerAvatar = participantAvatar(ownerParticipantsPanel, participantName);
      const guestAvatar = participantAvatar(guestParticipantsPanel, participantName);

      await expect(ownerAvatar).toBeVisible();
      await expect(guestAvatar).toBeVisible();
      await expect(ownerAvatar).toHaveAttribute('src', /.+/);
      await expect(guestAvatar).toHaveAttribute('src', /.+/);
      await expect
        .poll(() => ownerAvatar.evaluate((image) => (image as HTMLImageElement).naturalWidth))
        .toBeGreaterThan(0);
      await expect
        .poll(() => guestAvatar.evaluate((image) => (image as HTMLImageElement).naturalWidth))
        .toBeGreaterThan(0);

      const ownerAvatarSrc = await ownerAvatar.getAttribute('src');
      await expect(guestAvatar).toHaveAttribute('src', ownerAvatarSrc ?? '');
    }

    // Add a story to the backlog so the Edit button becomes available
    const storyName = `Story: estimate websocket sync ${Date.now()}`;
    await ownerPage.getByRole('button', { name: 'Open backlog' }).click();
    const ownerBacklogDialog = backlogDialog(ownerPage);
    await ownerBacklogDialog.getByPlaceholder('Enter story name...').fill(storyName);
    await ownerBacklogDialog.getByRole('button', { name: 'Add' }).click();
    // Verify the server response appears inside the backlog dialog (owner page)
    await expect(ownerBacklogDialog.getByText(storyName, { exact: true })).toBeVisible();

    // Open the backlog dialog on the guest page and assert the same
    await guestPage.getByRole('button', { name: 'Open backlog' }).click();
    const guestBacklogDialog = backlogDialog(guestPage);
    await expect(guestBacklogDialog.getByText(storyName, { exact: true })).toBeVisible();

    // Close both dialogs
    await ownerBacklogDialog.getByRole('button', { name: 'Close' }).click();
    await guestBacklogDialog.getByRole('button', { name: 'Close' }).click();

    await ownerPage.getByRole('button', { name: '5', exact: true }).click();
    const ownerVoteTime = ownerParticipantsPanel.getByText(/^Voted at (?:\d+:)?\d{2}:\d{2}$/);
    await expect(ownerVoteTime).toBeVisible();
    const propagatedVoteTime = await ownerVoteTime.textContent();
    expect(propagatedVoteTime).not.toBeNull();
    await expect(guestParticipantsPanel.getByText(propagatedVoteTime!, { exact: true })).toBeVisible();

    await guestPage.getByRole('button', { name: '8', exact: true }).click();

    await expect(ownerPage.getByText('2/2', { exact: true })).toBeVisible();
    await expect(guestPage.getByText('2/2', { exact: true })).toBeVisible();

    await expect(ownerPage.getByText('Results Summary')).toBeVisible();
    await expect(guestPage.getByText('Results Summary')).toBeVisible();
    await expect(ownerPage.getByText('Average: 6.5')).toBeVisible();
    await expect(guestPage.getByText('Average: 6.5')).toBeVisible();

    const updatedStory = `${storyName} - updated`;

    const ownerEditStoryButton = ownerPage.getByRole('button', { name: 'Edit' });
    const guestEditStoryButton = guestPage.getByRole('button', { name: 'Edit' });

    await expect
      .poll(async () => (await ownerEditStoryButton.count()) + (await guestEditStoryButton.count()))
      .toBe(1);

    const adminPage = (await ownerEditStoryButton.count()) === 1 ? ownerPage : guestPage;

    await adminPage.getByRole('button', { name: 'Edit' }).click();
    // Scope to the story card to avoid matching the backlog add input
    await storyCard(adminPage).getByRole('textbox').fill(updatedStory);
    await adminPage.getByRole('button', { name: 'Save' }).click();

    await expect(storyCard(ownerPage)).toContainText(updatedStory);
    await expect(storyCard(guestPage)).toContainText(updatedStory);

    // Escape should cancel an edit and restore the previous story name.
    await adminPage.getByRole('button', { name: 'Edit' }).click();
    const storyInput = storyCard(adminPage).getByRole('textbox');
    await storyInput.fill('Temporary story name');
    await storyInput.press('Escape');
    await expect(storyCard(adminPage)).toContainText(updatedStory);

    // Enter with an empty name should remove the current story instead of saving an empty one.
    await adminPage.getByRole('button', { name: 'Edit' }).click();
    const emptyStoryInput = storyCard(adminPage).getByRole('textbox');
    await emptyStoryInput.fill('');
    let confirmationType = '';
    const confirmationDialog = new Promise<void>((resolve) => {
      adminPage.once('dialog', async (dialog) => {
        confirmationType = dialog.type();
        await dialog.dismiss();
        resolve();
      });
    });
    await emptyStoryInput.press('Enter');
    await confirmationDialog;
    expect(confirmationType).toBe('confirm');
    await expect(storyCard(adminPage).getByRole('textbox')).toBeVisible();

    const removeDialog = new Promise<void>((resolve) => {
      adminPage.once('dialog', async (dialog) => {
        await dialog.accept();
        resolve();
      });
    });
    await storyCard(adminPage).getByRole('textbox').press('Enter');
    await removeDialog;
    await expect(storyCard(adminPage).getByText(updatedStory, { exact: true })).not.toBeVisible();
    await openBacklogModal(adminPage);
    const adminBacklogDialog = backlogDialog(adminPage);
    await expect(adminBacklogDialog.getByText(updatedStory, { exact: true })).not.toBeVisible();
    await adminBacklogDialog.getByRole('button', { name: 'Close' }).click();
  } finally {
    await Promise.allSettled([ownerContext.close(), guestContext.close()]);
  }
});
