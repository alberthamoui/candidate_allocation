import { test, expect } from '@playwright/test';
import { openPageHelp } from './support/ui';

test.describe('Suite 1: Home Page & Initial Setup', () => {
  test('A página carrega corretamente e exibe o título principal', async ({ page }) => {
    await page.goto('/');

    await expect(page.getByTestId('workflow-main').getByRole('heading', { name: 'Importação de Dados' })).toBeVisible();
  });

  test('Deve exibir erro ao tentar prosseguir sem selecionar um arquivo Excel', async ({ page }) => {
    await page.goto('/');

    await page.getByTestId('start-import-button').click();

    await expect(page.getByTestId('import-error-message')).toContainText('Por favor, selecione um arquivo.');
  });

  test('Deve abrir a ajuda contextual da página inicial', async ({ page }) => {
    await page.goto('/');

    await openPageHelp(page);
    await expect(page.getByTestId('page-help-dialog').getByText('Para que serve')).toBeVisible();
  });
});
