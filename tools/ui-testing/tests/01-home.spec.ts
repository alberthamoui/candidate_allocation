import { test, expect } from '@playwright/test';

test.describe('Suite 1: Home Page & Initial Setup', () => {
  test('A página carrega corretamente e exibe o título principal', async ({ page }) => {
    await page.goto('/');
    
    // Verifica se o título "Candidate Allocator" aparece na home
    const header = page.locator('h1').first();
    await expect(header).toHaveText('Candidate Allocator');
    await expect(page.getByLabel('Ajuda da Página')).toBeVisible();
  });

  test('Deve exibir erro ao tentar prosseguir sem selecionar um arquivo Excel', async ({ page }) => {
    await page.goto('/');
    
    const fileButton = page.locator('button', { hasText: 'Continuar' });
    await fileButton.click();
    
    const errorMsg = page.locator('text=Por favor, selecione um arquivo.');
    await expect(errorMsg).toBeVisible();
  });

  test('Deve abrir a ajuda contextual da página inicial', async ({ page }) => {
    await page.goto('/');

    await page.getByLabel('Ajuda da Página').click();
    await expect(page.locator('text=Para que serve')).toBeVisible();
    // await expect(page.locator('text=Inicie uma nova')).toBeVisible();
  });
});
