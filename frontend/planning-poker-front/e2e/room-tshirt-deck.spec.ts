import { expect, test } from '@playwright/test';

const roomUrlPattern = /\/room\/([0-9a-f-]{36})$/i;
const labels = ['XS', 'S', 'M', 'L', 'XL', 'XXL', '?', '☕'];

test('creates a T-shirt deck room and preserves its deck for everyone and after reconnect', async ({ browser }) => {
  test.setTimeout(60_000);
  const ownerContext = await browser.newContext();
  const guestContext = await browser.newContext();
  const owner = await ownerContext.newPage();
  const guest = await guestContext.newPage();
  try {
    await owner.goto('/join');
    await owner.getByPlaceholder('Enter your name').fill('Deck owner');
    await owner.getByLabel('Voting deck').selectOption('t-shirt');
    await Promise.all([owner.waitForURL(roomUrlPattern), owner.getByRole('button', { name: 'Create Room' }).click()]);
    const roomId = owner.url().match(roomUrlPattern)?.[1];
    expect(roomId).toBeTruthy();
    await guest.goto(`/join/${roomId}`);
    await guest.getByPlaceholder('Enter your name').fill('Deck guest');
    await Promise.all([guest.waitForURL(new RegExp(`/room/${roomId}$`)), guest.getByRole('button', { name: 'Join Room' }).click()]);
    for (const page of [owner, guest]) {
      for (const label of labels) await expect(page.getByRole('button', { name: label, exact: true })).toBeVisible();
    }
    await owner.getByRole('button', { name: '?', exact: true }).click();
    await guest.getByRole('button', { name: '☕', exact: true }).click();
    await expect(owner.getByText('2/2', { exact: true })).toBeVisible();
    await owner.getByRole('button', { name: 'Reveal Votes' }).click();
    await expect(owner.getByText('Most Common: ?, ☕')).toBeVisible();
    await owner.reload();
    for (const label of labels) await expect(owner.getByRole('button', { name: label, exact: true })).toBeVisible();
  } finally {
    await ownerContext.close();
    await guestContext.close();
  }
});
