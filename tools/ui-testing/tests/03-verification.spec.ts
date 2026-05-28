import { test, expect } from '@playwright/test';
import { confirmMapping, importSampleSpreadsheet } from './support/ui';

test.describe('Suite 3: Edição Inline e Validação (EntityVerificationView)', () => {
  test.beforeEach(async ({ page }) => {
    await importSampleSpreadsheet(page);
    await confirmMapping(page, 'Verificação de Usuários');
  });

  test('Deve bloquear salvamento quando houver duplicatas e resolver duplicata com sucesso', async ({ page }) => {
    // Verifica se existem duplicatas baseando-se no texto do botão
    const saveBtn = page.getByTestId('verification-save-button');
    const btnText = await saveBtn.innerText();
    
    if (btnText.includes('Resolva')) {
        // Assert de que o botão está bloqueado
        await expect(saveBtn).toBeDisabled();
        
        // Resolve a primeira duplicata clicando em "Aceitar Este"
        const acceptBtn = page.getByTestId('accept-duplicate-record-button').first();
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
    const firstNameField = page.getByTestId('user-card-field-nome').first();
    await expect(firstNameField).toBeVisible({ timeout: 10000 });
    await firstNameField.getByTestId('editable-cell-display').click();

    const activeInput = firstNameField.getByTestId('editable-cell-input');
    await activeInput.fill('Novo Valor Editado Playwright');
    await activeInput.press('Enter');

    await expect(firstNameField).toContainText('Novo Valor Editado Playwright');
  });
});
