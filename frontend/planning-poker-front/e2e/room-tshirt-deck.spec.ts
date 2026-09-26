import { expect, test, type Page } from '@playwright/test';

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

const expectTshirtCards = async (page: Page) => {
  await expect(page.getByRole('button', { name: 'M', exact: true })).toBeVisible();
  await expect(page.getByRole('button', { name: 'XL', exact: true })).toBeVisible();
  await expect(page.getByRole('button', { name: '☕', exact: true })).toBeVisible();
  await expect(page.getByRole('button', { name: '13', exact: true })).toHaveCount(0);
};

test('creates a T-shirt deck room, syncs the deck, and reveals a string result', async ({ browser, baseURL }) => {
  test.setTimeout(60_000);

  const ownerContext = await browser.newContext();
  const guestContext = await browser.newContext();

  await ownerContext.addInitScript(randomUUIDPolyfillScript);
  await guestContext.addInitScript(randomUUIDPolyfillScript);

  const ownerPage = await ownerContext.newPage();
  const guestPage = await guestContext.newPage();

  try {
    // Owner creates a room with the T-shirt sizes preset selected.
    await ownerPage.goto('/join');
    await ownerPage.getByPlaceholder('Enter your name').fill(userA);
    await ownerPage.getByLabel('Voting deck').selectOption({ label: 'T-shirt sizes' });

    const createRoomButton = ownerPage.getByRole('button', { name: 'Create Room' });
    await expect(createRoomButton).toBeEnabled();
    await Promise.all([
      ownerPage.waitForURL(roomUrlPattern),
      createRoomButton.click(),
    ]);

    const ownerRoomUrl = new URL(ownerPage.url(), baseURL);
    const roomMatch = ownerRoomUrl.pathname.match(roomUrlPattern);

    expect(roomMatch).not.toBeNull();

    const roomId = roomMatch?.[1] ?? '';

    // Owner sees the T-shirt cards and no Fibonacci card.
    await expectTshirtCards(ownerPage);

    // Guest joins and receives the same deck over the WebSocket room state.
    await guestPage.goto(`/join/${roomId}`);
    await guestPage.getByPlaceholder('Enter your name').fill(userB);
    await Promise.all([
      guestPage.waitForURL(new RegExp(`/room/${roomId}$`, 'i')),
      guestPage.getByRole('button', { name: 'Join Room' }).click(),
    ]);

    await expectTshirtCards(guestPage);

    // Guest votes S, owner votes M; the round reveals automatically once everyone has voted.
    await guestPage.getByRole('button', { name: 'S', exact: true }).click();
    await ownerPage.getByRole('button', { name: 'M', exact: true }).click();

    await expect(ownerPage.getByText('2/2', { exact: true })).toBeVisible();
    await expect(guestPage.getByText('2/2', { exact: true })).toBeVisible();

    await expect(ownerPage.getByText('Results Summary')).toBeVisible();
    await expect(guestPage.getByText('Results Summary')).toBeVisible();

    // Each vote is unique, so both cards are most-common, reported in deck order.
    await expect(ownerPage.getByText('Most Common: S, M')).toBeVisible();
    await expect(guestPage.getByText('Most Common: S, M')).toBeVisible();

    // Non-numeric decks keep only the most-common row: no average and no "Unavailable" metrics.
    await expect(ownerPage.getByText(/Average:/)).toHaveCount(0);
    await expect(guestPage.getByText(/Average:/)).toHaveCount(0);
    await expect(ownerPage.getByText(/Unavailable/)).toHaveCount(0);
    await expect(guestPage.getByText(/Unavailable/)).toHaveCount(0);

    // The deck and the revealed result survive a client reconnect.
    await guestPage.reload();
    await expectTshirtCards(guestPage);
    await expect(guestPage.getByText('Most Common: S, M')).toBeVisible();
  } finally {
    await Promise.allSettled([ownerContext.close(), guestContext.close()]);
  }
});
