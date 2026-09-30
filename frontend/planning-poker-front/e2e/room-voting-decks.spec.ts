import { expect, test, type Page } from '@playwright/test';

const roomUrlPattern = /\/room\/([0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12})$/i;
const ownerName = 'Deck owner';
const guestName = 'Deck guest';

const fibonacciCards = ['0', '1', '2', '3', '5', '8', '13', '21', '34', '55', '89', '?', '☕'];
const tshirtCards = ['XS', 'S', 'M', 'L', 'XL', 'XXL', '?', '☕'];

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

const roomCards = (page: Page) =>
  page.getByRole('heading', { name: 'Select Your Card' }).locator('xpath=..').getByRole('button');

const backlogDialog = (page: Page) => page.getByRole('dialog', { name: 'Story Backlog' });

const createRoom = async (page: Page, baseURL: string | undefined, userName: string, deckId?: string) => {
  await page.goto('/join');
  const deckSelector = page.getByLabel('Voting Deck');
  await expect(deckSelector).toHaveValue('fibonacci');
  if (deckId) {
    await deckSelector.selectOption(deckId);
  }
  await page.getByPlaceholder('Enter your name').fill(userName);

  await Promise.all([
    page.waitForURL(roomUrlPattern),
    page.getByRole('button', { name: 'Create Room' }).click(),
  ]);

  const roomPath = new URL(page.url(), baseURL).pathname;
  const roomMatch = roomPath.match(roomUrlPattern);
  expect(roomMatch).not.toBeNull();
  return roomMatch?.[1] ?? '';
};

const joinRoom = async (page: Page, roomId: string, userName: string) => {
  await page.goto(`/join/${roomId}`);
  await expect(page.getByLabel('Voting Deck')).toHaveCount(0);
  await page.getByPlaceholder('Enter your name').fill(userName);

  await Promise.all([
    page.waitForURL(new RegExp(`/room/${roomId}$`, 'i')),
    page.getByRole('button', { name: 'Join Room' }).click(),
  ]);
};

const expectDeck = async (page: Page, name: string, cards: readonly string[]) => {
  await expect(page.getByText(`Deck: ${name}`, { exact: true })).toBeVisible();
  await expect(roomCards(page)).toHaveText([...cards]);
};

const addStory = async (page: Page, story: string) => {
  await page.getByRole('button', { name: 'Open backlog' }).click();
  const dialog = backlogDialog(page);
  await dialog.getByPlaceholder('Enter story name...').fill(story);
  await dialog.getByRole('button', { name: 'Add' }).click();
  await expect(dialog.getByText(story, { exact: true })).toBeVisible();
  await dialog.getByRole('button', { name: 'Close' }).click();
  await expect(page.getByText(story)).toBeVisible();
};

test('Fibonacci is the default and two clients share votes and results', async ({ browser, baseURL }) => {
  test.setTimeout(60_000);

  const ownerContext = await browser.newContext();
  const guestContext = await browser.newContext();
  await ownerContext.addInitScript(randomUUIDPolyfillScript);
  await guestContext.addInitScript(randomUUIDPolyfillScript);

  const ownerPage = await ownerContext.newPage();
  const guestPage = await guestContext.newPage();

  try {
    const roomId = await createRoom(ownerPage, baseURL, ownerName);
    await expectDeck(ownerPage, 'Fibonacci', fibonacciCards);
    await joinRoom(guestPage, roomId, guestName);
    await expectDeck(guestPage, 'Fibonacci', fibonacciCards);

    await ownerPage.getByRole('button', { name: '5', exact: true }).click();
    await expect(ownerPage.getByRole('button', { name: '5', exact: true })).toHaveAttribute('aria-pressed', 'true');
    await guestPage.getByRole('button', { name: '8', exact: true }).click();

    for (const page of [ownerPage, guestPage]) {
      await expect(page.getByText('Results Summary')).toBeVisible();
      await expect(page.getByText('Average: 6.5', { exact: true })).toBeVisible();
      await expect(page.getByText('Most Common: 5, 8', { exact: true })).toBeVisible();
    }

    await ownerPage.setViewportSize({ width: 390, height: 844 });
    const votingHeading = await ownerPage.getByRole('heading', { name: 'Select Your Card' }).boundingBox();
    const participantsHeading = await ownerPage.getByRole('heading', { name: 'Participants' }).boundingBox();
    if (!votingHeading || !participantsHeading) {
      throw new Error('Room panels are missing from the mobile layout');
    }
    expect(participantsHeading.y).toBeGreaterThan(votingHeading.y);
    expect(await ownerPage.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);
  } finally {
    await Promise.allSettled([ownerContext.close(), guestContext.close()]);
  }
});

test('T-shirt deck syncs across clients and keeps saved estimates after reconnect', async ({ browser, baseURL }) => {
  test.setTimeout(60_000);

  const ownerContext = await browser.newContext();
  const guestContext = await browser.newContext();
  await ownerContext.addInitScript(randomUUIDPolyfillScript);
  await guestContext.addInitScript(randomUUIDPolyfillScript);

  const ownerPage = await ownerContext.newPage();
  const guestPage = await guestContext.newPage();

  try {
    const roomId = await createRoom(ownerPage, baseURL, ownerName, 'tshirt');
    await expectDeck(ownerPage, 'T-shirt sizes', tshirtCards);
    await joinRoom(guestPage, roomId, guestName);
    await expectDeck(guestPage, 'T-shirt sizes', tshirtCards);

    const story = 'T-shirt lifecycle story';
    await addStory(ownerPage, story);

    await ownerPage.getByRole('button', { name: 'S', exact: true }).click();
    await guestPage.getByRole('button', { name: 'M', exact: true }).click();
    for (const page of [ownerPage, guestPage]) {
      await expect(page.getByText('Results Summary')).toBeVisible();
      await expect(page.getByText('Most Common: S, M', { exact: true })).toBeVisible();
      await expect(page.getByText(/^Average:/)).toHaveCount(0);
      await expect(page.getByText(/Votes range from/)).toHaveCount(0);
    }

    await ownerPage.getByRole('button', { name: 'Open backlog' }).click();
    await expect(backlogDialog(ownerPage).getByText('Estimate: S, M', { exact: true })).toBeVisible();
    await backlogDialog(ownerPage).getByRole('button', { name: 'Close' }).click();

    await guestPage.evaluate(() => {
      const socket = (window as Window & { __ws?: WebSocket }).__ws;
      if (socket?.readyState === WebSocket.OPEN) {
        socket.close();
      }
    });
    const reconnectBanner = guestPage.getByText('Connection lost. Reconnecting...');
    await expect(reconnectBanner).toBeVisible({ timeout: 5000 });
    await expect(reconnectBanner).not.toBeVisible({ timeout: 15000 });
    await expectDeck(guestPage, 'T-shirt sizes', tshirtCards);

    for (const page of [ownerPage, guestPage]) {
      await page.getByRole('button', { name: 'Open backlog' }).click();
      await expect(backlogDialog(page).getByText('Estimate: S, M', { exact: true })).toBeVisible();
      await backlogDialog(page).getByRole('button', { name: 'Close' }).click();
    }

    await ownerPage.setViewportSize({ width: 390, height: 844 });
    const votingHeading = await ownerPage.getByRole('heading', { name: 'Select Your Card' }).boundingBox();
    const participantsHeading = await ownerPage.getByRole('heading', { name: 'Participants' }).boundingBox();
    if (!votingHeading || !participantsHeading) {
      throw new Error('Room panels are missing from the mobile layout');
    }
    expect(participantsHeading.y).toBeGreaterThan(votingHeading.y);
    expect(await ownerPage.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);
  } finally {
    await Promise.allSettled([ownerContext.close(), guestContext.close()]);
  }
});
