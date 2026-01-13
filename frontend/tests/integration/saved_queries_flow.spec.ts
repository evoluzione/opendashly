import { test, expect } from '@playwright/test';

test('saved queries page renders', async ({ page }) => {
  await page.goto('/queries');
  await expect(page.getByRole('heading', { name: 'Saved Queries' })).toBeVisible();
});
