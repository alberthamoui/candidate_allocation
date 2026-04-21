import { test, expect } from '@playwright/test';
import path from 'path';

test.describe('Suite 3: Edição Inline e Validação (EntityVerificationView)', () => {
  const sampleFilePath = path.resolve(__dirname, '../../../Execelteste/Base4Restricao.xlsx');

  test.beforeEach(async ({ page }) => {
    // Faz o fluxo inicial silenciosamente
    await page.goto('/');
    await page.setInputFiles('#fileInput', sampleFilePath);
    await page.click('button:has-text("Executar função de arquivo")');
    
    // Esperar a página de Mapeamento
    await expect(page.locator('h1', { hasText: 'Mapeamento de Candidatos' })).toBeVisible({ timeout: 15000 });
    
    // Clicar em "Revisar candidatos" para ir para a tela de Verificação
    await page.waitForTimeout(500); // pequeno timeout para react state settle
    await page.click('button:has-text("Revisar candidatos")');
    
    // Esperar a página de Verificação
    await expect(page.locator('h1', { hasText: 'Verificação de Usuários' })).toBeVisible({ timeout: 15000 });
  });

  test('Deve bloquear salvamento quando houver duplicatas e resolver duplicata com sucesso', async ({ page }) => {
    // Verifica se existem duplicatas baseando-se no texto do botão
    const saveBtn = page.locator('button', { hasText: /(Resolva.*duplicado|Salvar Dados)/i });
    const btnText = await saveBtn.innerText();
    
    if (btnText.includes('Resolva')) {
        // Assert de que o botão está bloqueado
        await expect(saveBtn).toBeDisabled();
        
        // Resolve a primeira duplicata clicando em "Aceitar Este"
        const acceptBtn = page.locator('button', { hasText: 'Aceitar Este' }).first();
        if (await acceptBtn.isVisible()) {
            await acceptBtn.click();
        }
        
        // O texto do botão deve mudar para "Salvar Dados" se não houver mais duplicatas
        // ou o número de duplicatas deve ter diminuído
        const updatedBtnText = await saveBtn.innerText();
        expect(updatedBtnText).not.toEqual(btnText);
    } else {
        // Se a base de testes não tiver duplicatas, o botão já deve estar disponível
        await expect(saveBtn).toBeEnabled();
        await expect(saveBtn).toHaveText(/Salvar Dados/i);
    }
  });

  test('Edição inline de um campo do usuário', async ({ page }) => {
    // Pega o primeiro valor clicável (ex: Nome do primeiro card)
    const firstCell = page.locator('text=Teste 1').first();
    await expect(firstCell).toBeVisible({ timeout: 10000 });
    await firstCell.click();
    
    // Agora o input deve aparecer focado e podemos preenchê-lo
    const activeInput = page.locator('input').first();
    await activeInput.fill('Novo Valor Editado Playwright');
    await activeInput.press('Enter');
    
    // Verifica se a UI mantém o novo valor
    await expect(page.locator('text=Novo Valor Editado Playwright').first()).toBeVisible();
  });
});
