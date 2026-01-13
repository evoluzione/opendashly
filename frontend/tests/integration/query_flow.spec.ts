import { test, expect } from '@playwright/test';

test('query flow renders', async ({ page }) => {
  await page.goto('/query');
  await expect(page.getByRole('heading', { name: 'Telemetry Dashboard' })).toBeVisible();
});
