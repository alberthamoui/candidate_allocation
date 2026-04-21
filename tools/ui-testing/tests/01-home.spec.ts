import { test, expect } from '@playwright/test';

test.describe('Suite 1: Home Page & Initial Setup', () => {
  test('A página carrega corretamente e exibe o título principal', async ({ page }) => {
    await page.goto('/');
    
    // Verifica se o título "Candidate Allocator" aparece na home
    const header = page.locator('h1');
    await expect(header).toHaveText('Candidate Allocator');
    await expect(page.getByTestId('page-help-button')).toBeVisible();
  });

  test('Deve interagir com a funcionalidade Greet do Wails', async ({ page }) => {
    await page.goto('/');
    
    const nameInput = page.locator('#name');
    await nameInput.fill('Playwright Agent');
    
    const greetButton = page.locator('button', { hasText: 'Greet' });
    await greetButton.click();
    
    // A interface deve atualizar a mensagem com o nome passado
    const resultDiv = page.locator('#result');
    await expect(resultDiv).toContainText('Playwright Agent');
  });

  test('Deve exibir erro ao tentar prosseguir sem selecionar um arquivo Excel', async ({ page }) => {
    await page.goto('/');
    
    const fileButton = page.locator('button', { hasText: 'Executar função de arquivo' });
    await fileButton.click();
    
    const errorMsg = page.locator('#fileResult');
    await expect(errorMsg).toHaveText('Por favor, selecione um arquivo.');
    await expect(errorMsg).toBeVisible();
  });

  test('Deve abrir a ajuda contextual da página inicial', async ({ page }) => {
    await page.goto('/');

    await page.getByTestId('page-help-button').click();
    await expect(page.locator('text=Ajuda desta página')).toBeVisible();
    await expect(page.locator('text=Entrada controlada')).toBeVisible();
  });
});
