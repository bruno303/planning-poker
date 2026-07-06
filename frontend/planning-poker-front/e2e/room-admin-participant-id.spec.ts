import { expect, Page, test } from '@playwright/test';

const roomUrlPattern = /\/room\/([0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12})$/i;
const uuidPattern = /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i;

const ownerName = 'Owner';
const guestName = 'Guest';

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

test('shows participant ID badge to owner and allows copy on click', async ({ browser, baseURL }) => {
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

    // Wait for participants to appear on both pages
    const ownerPanel = participantsPanel(ownerPage);
    const guestPanel = participantsPanel(guestPage);

    await expect(ownerPanel.getByText(ownerName, { exact: true })).toBeVisible();
    await expect(ownerPanel.getByText(guestName, { exact: true })).toBeVisible();
    await expect(guestPanel.getByText(ownerName, { exact: true })).toBeVisible();
    await expect(guestPanel.getByText(guestName, { exact: true })).toBeVisible();

    // Admin (owner) should see the ID badge button
    const badgeButton = ownerPanel.getByRole('button', { name: 'Click to copy participant ID' });
    // Expect at least one badge (the guests and self)
    await expect(badgeButton.first()).toBeVisible();

    // Guest should NOT see any ID badge
    await expect(
      guestPanel.getByRole('button', { name: 'Click to copy participant ID' })
    ).toHaveCount(0);

    // Hover over the badge to reveal the tooltip with UUID
    await badgeButton.first().hover();
    const tooltip = ownerPanel.locator('code');
    await expect(tooltip.first()).toBeVisible();
    const tooltipText = await tooltip.first().textContent();
    expect(tooltipText).toMatch(uuidPattern);

    // Click to copy and verify toast confirmation
    // Override navigator.clipboard in test environment (non-HTTPS Docker)
    // so the success toast path is exercised.
    await ownerPage.evaluate(() => {
      const fakeClipboard = { writeText: () => Promise.resolve() };
      // @ts-expect-error - replacing clipboard for test
      navigator.clipboard = fakeClipboard;
    });
    await badgeButton.first().click();

    // Verify toast confirmation appears
    await expect(ownerPage.getByText('Participant ID copied!')).toBeVisible();
  } finally {
    await Promise.allSettled([ownerContext.close(), guestContext.close()]);
  }
});
