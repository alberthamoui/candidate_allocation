import { test, expect } from '@playwright/test';
import path from 'path';

test.describe('Allocation Flow UI', () => {
  const sampleFilePath = path.resolve(__dirname, '../../../Execelteste/Base4Restricao.xlsx');

  test('should config, load and display results', async ({ page }) => {
    await page.goto('/');
    await page.setInputFiles('#fileInput', sampleFilePath);
    await page.click('button:has-text("Continuar")');

    await expect(page.getByRole('main').getByRole('heading', { name: 'Mapeamento de Candidatos' })).toBeVisible({ timeout: 15000 });
    await page.getByTestId('mapping-confirm-button').click();

    await expect(page.getByRole('main').getByRole('heading', { name: 'Verificação de Usuários' })).toBeVisible({ timeout: 15000 });
    let resolveBtns = page.locator('button:has-text("Aceitar Este")');
    while (await resolveBtns.count() > 0) {
      const btn = resolveBtns.first();
      if (await btn.isVisible()) {
        await btn.click();
      }
      await page.waitForTimeout(200);
      resolveBtns = page.locator('button:has-text("Aceitar Este")');
    }
    await page.getByTestId('verification-save-button').click();

    await expect(page.getByRole('main').getByRole('heading', { name: 'Mapeamento de Restricoes' })).toBeVisible({ timeout: 15000 });
    await page.getByTestId('mapping-confirm-button').click();

    await expect(page.getByRole('main').getByRole('heading', { name: 'Verificação de Restrições' })).toBeVisible({ timeout: 15000 });
    await page.getByTestId('verification-save-button').click();

    await expect(page.getByRole('main').getByRole('heading', { name: 'Mapeamento de Avaliadores' })).toBeVisible({ timeout: 15000 });
    await page.getByTestId('mapping-confirm-button').click();

    await expect(page.getByRole('main').getByRole('heading', { name: 'Verificação de Avaliadores' })).toBeVisible({ timeout: 15000 });
    resolveBtns = page.locator('button:has-text("Aceitar Este")');
    while (await resolveBtns.count() > 0) {
      const btn = resolveBtns.first();
      if (await btn.isVisible()) {
        await btn.click();
      }
      await page.waitForTimeout(200);
      resolveBtns = page.locator('button:has-text("Aceitar Este")');
    }
    await page.getByTestId('verification-save-button').click();

    await expect(page.locator('h1', { hasText: 'Tudo Pronto!' })).toBeVisible({ timeout: 15000 });

    const configButton = page.getByRole('button', { name: 'Configurar Alocação' });
    await expect(configButton).toBeVisible();
    await configButton.click();
    await expect(page).toHaveURL(/allocation-config/);

    await expect(page.locator('h1', { hasText: 'Configurações de Alocação' })).toBeVisible();
    await page.locator('input[type="number"]').nth(0).fill('3');
    await page.getByLabel('Ajuda da Página').click();
    await expect(page.locator('text=Para que serve')).toBeVisible();
    await page.getByLabel('Fechar').click();

    await page.getByTestId('add-criterion-button').click();
    await expect(page.locator('select').first()).toBeVisible();
    await page.getByTestId('start-allocation-button').click();

    await expect(page.locator('text=Resultados da Alocação')).toBeVisible({ timeout: 15000 });
    await expect(page.locator('text=Destacar Candidatos')).toBeVisible();
    await expect(page.getByRole('heading', { name: 'Não Alocados' })).toBeVisible();
  });
});
