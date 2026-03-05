import { test, expect } from '@playwright/test';

test('admin dashboard designer renders controls and reacts to reset', async ({ page }) => {
  await page.goto('/admin/dashboard');

  await expect(page.getByRole('heading', { name: 'Designer dashboard' })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Reset layout' })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Annulla' })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Salva' })).toBeVisible();

  const saveButton = page.getByRole('button', { name: 'Salva' });
  const resetButton = page.getByRole('button', { name: 'Reset layout' });

  await expect(saveButton).toBeDisabled();
  await resetButton.click();
  await expect(saveButton).toBeEnabled();
});
