import { test, expect } from '@playwright/test';
import path from 'path';

test.describe('Suite 4: A Jornada Completa (Happy Path)', () => {
  const sampleFilePath = path.resolve(__dirname, '../../../Execelteste/Base4Restricao.xlsx');

  test('Deve completar o fluxo inteiro com sucesso (Wizard)', async ({ page }) => {
    // 1. Home -> Upload
    await page.goto('/');
    await page.setInputFiles('#fileInput', sampleFilePath);
    await page.click('button:has-text("Continuar")');
    
    // 2. Mapeamento Candidatos -> Verify Candidatos
    await expect(page.locator('h1', { hasText: 'Mapeamento de Candidatos' })).toBeVisible({ timeout: 15000 });
    await page.waitForTimeout(1000);
    await page.getByTestId('mapping-confirm-button').click();
    
    // 3. Verify Candidatos -> Mapeamento Restrições
    await expect(page.locator('h1', { hasText: 'Verificação de Usuários' })).toBeVisible({ timeout: 15000 });
    
    // Resolver todas as duplicatas se houver
    let resolveBtns = page.locator('button:has-text("Aceitar Este")');
    while (await resolveBtns.count() > 0) {
        const btn = resolveBtns.first();
        if (await btn.isVisible()) {
            await btn.click();
        }
        await page.waitForTimeout(200); // Wait for React state
        resolveBtns = page.locator('button:has-text("Aceitar Este")');
    }
    await page.getByTestId('verification-save-button').click();
    
    // 4. Mapeamento Restrições -> Verify Restrições
    await expect(page.locator('h1', { hasText: 'Mapeamento de Restricoes' })).toBeVisible({ timeout: 15000 });
    await page.waitForTimeout(1000);
    await page.getByTestId('mapping-confirm-button').click();

    // 5. Verify Restrições -> Mapeamento Avaliadores
    await expect(page.locator('h1', { hasText: 'Verificação de Restrições' })).toBeVisible({ timeout: 15000 });
    
    // Resolver todas as duplicatas de restrições se houver
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

    // 6. Mapeamento Avaliadores -> Verify Avaliadores
    await expect(page.locator('h1', { hasText: 'Mapeamento de Avaliadores' })).toBeVisible({ timeout: 15000 });
    await page.waitForTimeout(1000);
    await page.getByTestId('mapping-confirm-button').click();

    // 7. Verify Avaliadores -> Success
    await expect(page.locator('h1', { hasText: 'Verificação de Avaliadores' })).toBeVisible({ timeout: 15000 });
    
    // Resolver todas as duplicatas de avaliadores se houver
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

    // 8. Success Page
    await expect(page.locator('h1', { hasText: 'Tudo Pronto!' })).toBeVisible({ timeout: 15000 });
    // await expect(page.locator('text=A preparação da base terminou')).toBeVisible();
    
    // Validate the button to return to home
    const restartBtn = page.locator('button:has-text("Nova Importação")');
    await expect(restartBtn).toBeVisible();
  });
});
