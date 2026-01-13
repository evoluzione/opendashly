import { test, expect } from '@playwright/test';

test('correlation page renders', async ({ page }) => {
  await page.goto('/traces/demo-trace');
  await expect(page.getByRole('heading', { name: 'Trace Detail' })).toBeVisible();
});
