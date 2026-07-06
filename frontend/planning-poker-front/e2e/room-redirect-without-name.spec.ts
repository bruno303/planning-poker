import { expect, test } from '@playwright/test';

test('redirects to join page when accessing room without a name set', async ({ page }) => {
  test.setTimeout(30_000);

  const roomId = '123e4567-e89b-12d3-a456-426614174000';

  // Navigate directly to a room without setting sessionStorage
  await page.goto(`/room/${roomId}`);

  // Should redirect to /join/<roomId> — the "Join Room" button is visible
  // (only present when roomId param exists in the join page)
  await expect(page.getByRole('button', { name: 'Join Room' })).toBeVisible({ timeout: 10000 });
  await expect(page).toHaveURL(`/join/${roomId}`);

  // Verify room-specific elements are NOT visible (no blink)
  await expect(page.getByText('Select Your Card')).not.toBeVisible();
  await expect(page.getByText('Participants')).not.toBeVisible();
  await expect(page.getByText('Current Story')).not.toBeVisible();
});
