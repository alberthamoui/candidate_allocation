import { test, expect } from '@playwright/test';
import path from 'path';

test.describe('Layout polish', () => {
  test('header and help affordances remain visible', async ({ page }) => {
    await page.goto('/');

    await expect(page.locator('header')).toBeVisible();
    await expect(page.locator('header').getByLabel('Ajuda da Página')).toBeVisible();
    await expect(page.getByLabel('Ajuda da Página')).toHaveText('?');
  });

  test('navegacao reseta o scroll da tela', async ({ page }) => {
    const sampleFilePath = path.resolve(__dirname, '../../../Execelteste/Base4Restricao.xlsx');

    await page.goto('/');
    await page.setInputFiles('#fileInput', sampleFilePath);
    await page.click('button:has-text("Continuar")');

    await page.evaluate(() => window.scrollTo(0, document.body.scrollHeight));
    const scrolledY = await page.evaluate(() => window.scrollY);
    expect(scrolledY).toBeGreaterThan(0);

    await page.getByRole('main').getByRole('heading', { name: 'Mapeamento de Candidatos' }).waitFor({ state: 'visible' });
    const topY = await page.evaluate(() => window.scrollY);
    expect(topY).toBe(0);
  });

  test('ajuda contextual nao altera o scroll', async ({ page }) => {
    await page.goto('/');

    await page.evaluate(() => window.scrollTo(0, document.body.scrollHeight));
    const before = await page.evaluate(() => window.scrollY);
    expect(before).toBeGreaterThan(0);

    await page.locator('header').getByLabel('Ajuda da Página').click();
    await expect(page.getByText('Para que serve')).toBeVisible();

    const after = await page.evaluate(() => window.scrollY);
    expect(after).toBe(before);
  });
});
