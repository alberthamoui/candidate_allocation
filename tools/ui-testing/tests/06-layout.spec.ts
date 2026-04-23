import { test, expect } from '@playwright/test';

test.describe('Layout polish', () => {
  test('header and help affordances remain visible', async ({ page }) => {
    await page.goto('/');

    await expect(page.locator('header')).toBeVisible();
    await expect(page.getByLabel('Ajuda da Página')).toBeVisible();
    await expect(page.getByLabel('Ajuda da Página')).toHaveText('?');
  });
});
