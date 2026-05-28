import { test, expect } from '@playwright/test';
import { confirmMapping, importSampleSpreadsheet, openPageHelp } from './support/ui';

test.describe('Layout polish', () => {
  test('header and help affordances remain visible', async ({ page }) => {
    await page.goto('/');

    await expect(page.getByRole('banner')).toBeVisible();
    await expect(page.getByTestId('page-help-button')).toBeVisible();
    await expect(page.getByTestId('page-help-button')).toHaveText('?');
  });

  test('navegacao reseta o scroll da tela', async ({ page }) => {
    await importSampleSpreadsheet(page);

    await page.evaluate(() => window.scrollTo(0, document.body.scrollHeight));
    const scrolledY = await page.evaluate(() => window.scrollY);
    expect(scrolledY).toBeGreaterThan(0);

    await confirmMapping(page, 'Verificação de Usuários');
    const topY = await page.evaluate(() => window.scrollY);
    expect(topY).toBe(0);
  });

  test('ajuda contextual nao altera o scroll', async ({ page }) => {
    await page.goto('/');

    await page.evaluate(() => window.scrollTo(0, document.body.scrollHeight));
    const before = await page.evaluate(() => window.scrollY);
    expect(before).toBeGreaterThan(0);

    await openPageHelp(page);

    const after = await page.evaluate(() => window.scrollY);
    expect(after).toBe(before);
  });
});
