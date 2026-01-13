import { test, expect } from '@playwright/test';

test('query flow renders', async ({ page }) => {
  await page.goto('/query');
  await expect(page.getByRole('heading', { name: 'Telemetry Dashboard' })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Last 5 minutes' })).toBeVisible();
  await expect(page.getByLabel('Auto Refresh')).toBeVisible();
  await expect(page.getByLabel('Page Size')).toBeVisible();
});
