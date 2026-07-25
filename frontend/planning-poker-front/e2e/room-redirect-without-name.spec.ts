import { expect, test } from '@playwright/test';

test('redirects to join page when accessing room without a name set', async ({ page }) => {
  test.setTimeout(30_000);

  const roomId = '123e4567-e89b-12d3-a456-426614174000';
  const consoleLogs: string[] = [];
  page.on('console', (msg) => consoleLogs.push(`[${msg.type()}] ${msg.text()}`));

  // Navigate directly to a room without setting sessionStorage
  await page.goto(`/room/${roomId}`, { waitUntil: 'networkidle' });

  // Wait a moment for React to hydrate and run effects
  await page.waitForTimeout(2000);

  // Debug: log current URL and console output
  const currentUrl = page.url();
  const pageContent = await page.content();
  const hasJoinRoom = pageContent.includes('Join Room');
  const hasCreateRoom = pageContent.includes('Create Room');
  console.log(`[TEST] Current URL: ${currentUrl}`);
  console.log(`[TEST] Has Join Room button: ${hasJoinRoom}`);
  console.log(`[TEST] Has Create Room button: ${hasCreateRoom}`);
  console.log(`[TEST] Console logs: ${JSON.stringify(consoleLogs)}`);

  // Should redirect to /join/<roomId> — the "Join Room" button is visible
  // (only present when roomId param exists in the join page)
  await expect(page.getByRole('button', { name: 'Join Room' })).toBeVisible({ timeout: 20000 });
  await expect(page).toHaveURL(`/join/${roomId}`);

  // Verify room-specific elements are NOT visible (no blink)
  await expect(page.getByText('Select Your Card')).not.toBeVisible();
  await expect(page.getByText('Participants')).not.toBeVisible();
  await expect(page.getByText('Current Story')).not.toBeVisible();
});
