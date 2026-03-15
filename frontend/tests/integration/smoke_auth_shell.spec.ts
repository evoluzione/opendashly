import { test, expect } from '@playwright/test';

test('login page renderizza i controlli principali', async ({ page }) => {
  await page.goto('/login');
  await expect(page.getByRole('heading', { name: 'Bentornato' })).toBeVisible();
  await expect(page.getByLabel('Nome utente')).toBeVisible();
  await expect(page.getByLabel('Password')).toBeVisible();
  await expect(page.getByRole('button', { name: 'Accedi' })).toBeVisible();
});

test('route protetta reindirizza al login senza sessione', async ({ page }) => {
  await page.goto('/query');
  await expect(page).toHaveURL(/\/login$/);
  await expect(page.getByRole('heading', { name: 'Bentornato' })).toBeVisible();
});

test('first-login page e raggiungibile e renderizza il form', async ({ page }) => {
  await page.goto('/first-login');
  await expect(page.getByRole('heading', { name: 'Imposta una nuova password' })).toBeVisible();
  await expect(page.getByLabel('Nuova password')).toBeVisible();
  await expect(page.getByRole('button', { name: 'Aggiorna password' })).toBeVisible();
});
