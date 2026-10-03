import { expect, test, type Locator, type Page } from '@playwright/test';

const roomUrlPattern = /\/room\/([0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12})$/i;
const userA = 'T-Shirt Owner';
const userB = 'T-Shirt Guest';

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
  page.getByRole('heading', { name: 'Participants' }).locator('xpath=ancestor::div[2]');

const card = (page: Page, label: string): Locator => page.getByRole('button', { name: label, exact: true });

test('creates a T-shirt room, renders T-shirt cards, and summarizes votes without an average', async ({ browser, baseURL }) => {
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
    await ownerPage.getByLabel('Deck').selectOption('t-shirt');
    const createRoomButton = ownerPage.getByRole('button', { name: 'Create Room' });
    await expect(createRoomButton).toBeEnabled();
    await Promise.all([
      ownerPage.waitForURL(roomUrlPattern),
      createRoomButton.click(),
    ]);

    const roomMatch = new URL(ownerPage.url(), baseURL).pathname.match(roomUrlPattern);
    expect(roomMatch).not.toBeNull();
    const roomId = roomMatch?.[1] ?? '';

    await expect(card(ownerPage, 'XXL')).toBeVisible();
    await expect(ownerPage.getByRole('button', { name: '8', exact: true })).toHaveCount(0);

    await guestPage.goto(`/join/${roomId}`);
    await guestPage.getByPlaceholder('Enter your name').fill(userB);
    await Promise.all([
      guestPage.waitForURL(new RegExp(`/room/${roomId}$`, 'i')),
      guestPage.getByRole('button', { name: 'Join Room' }).click(),
    ]);

    await expect(card(guestPage, 'XXL')).toBeVisible();

    await card(ownerPage, 'M').click();
    await expect(participantsPanel(ownerPage).getByText(userA, { exact: true })).toBeVisible();
    await expect(participantsPanel(guestPage).getByText(userA, { exact: true })).toBeVisible();

    await card(guestPage, 'L').click();

    await expect(ownerPage.getByText('2/2', { exact: true })).toBeVisible();
    await expect(guestPage.getByText('2/2', { exact: true })).toBeVisible();

    await expect(ownerPage.getByText('Results Summary')).toBeVisible();
    await expect(guestPage.getByText('Results Summary')).toBeVisible();
    await expect(ownerPage.getByText('Votes range from M to L')).toBeVisible();
    await expect(guestPage.getByText('Votes range from M to L')).toBeVisible();
    await expect(ownerPage.getByText(/^Average:/)).toHaveCount(0);
    await expect(guestPage.getByText(/^Average:/)).toHaveCount(0);
  } finally {
    await Promise.allSettled([ownerContext.close(), guestContext.close()]);
  }
});
