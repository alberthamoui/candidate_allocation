# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: 03-verification.spec.ts >> Suite 3: Edição Inline e Validação (EntityVerificationView) >> Edição inline de um campo do usuário
- Location: tests/03-verification.spec.ts:50:7

# Error details

```
Error: page.goto: net::ERR_CONNECTION_REFUSED at http://localhost:34115/
Call log:
  - navigating to "http://localhost:34115/", waiting until "load"

```

# Test source

```ts
  1  | import { test, expect } from '@playwright/test';
  2  | import path from 'path';
  3  | 
  4  | test.describe('Suite 3: Edição Inline e Validação (EntityVerificationView)', () => {
  5  |   const sampleFilePath = path.resolve(__dirname, '../../../Execelteste/Base4Restricao.xlsx');
  6  | 
  7  |   test.beforeEach(async ({ page }) => {
  8  |     // Faz o fluxo inicial silenciosamente
> 9  |     await page.goto('/');
     |                ^ Error: page.goto: net::ERR_CONNECTION_REFUSED at http://localhost:34115/
  10 |     await page.setInputFiles('#fileInput', sampleFilePath);
  11 |     await page.click('button:has-text("Executar função de arquivo")');
  12 |     
  13 |     // Esperar a página de Mapeamento
  14 |     await expect(page.locator('h1', { hasText: 'Mapeamento de Candidatos' })).toBeVisible({ timeout: 15000 });
  15 |     
  16 |     // Clicar em "Revisar candidatos" para ir para a tela de Verificação
  17 |     await page.waitForTimeout(500); // pequeno timeout para react state settle
  18 |     await page.click('button:has-text("Revisar candidatos")');
  19 |     
  20 |     // Esperar a página de Verificação
  21 |     await expect(page.locator('h1', { hasText: 'Verificação de Usuários' })).toBeVisible({ timeout: 15000 });
  22 |   });
  23 | 
  24 |   test('Deve bloquear salvamento quando houver duplicatas e resolver duplicata com sucesso', async ({ page }) => {
  25 |     // Verifica se existem duplicatas baseando-se no texto do botão
  26 |     const saveBtn = page.locator('button', { hasText: /(Resolva.*duplicado|Salvar Dados)/i });
  27 |     const btnText = await saveBtn.innerText();
  28 |     
  29 |     if (btnText.includes('Resolva')) {
  30 |         // Assert de que o botão está bloqueado
  31 |         await expect(saveBtn).toBeDisabled();
  32 |         
  33 |         // Resolve a primeira duplicata clicando em "Aceitar Este"
  34 |         const acceptBtn = page.locator('button', { hasText: 'Aceitar Este' }).first();
  35 |         if (await acceptBtn.isVisible()) {
  36 |             await acceptBtn.click();
  37 |         }
  38 |         
  39 |         // O texto do botão deve mudar para "Salvar Dados" se não houver mais duplicatas
  40 |         // ou o número de duplicatas deve ter diminuído
  41 |         const updatedBtnText = await saveBtn.innerText();
  42 |         expect(updatedBtnText).not.toEqual(btnText);
  43 |     } else {
  44 |         // Se a base de testes não tiver duplicatas, o botão já deve estar disponível
  45 |         await expect(saveBtn).toBeEnabled();
  46 |         await expect(saveBtn).toHaveText(/Salvar Dados/i);
  47 |     }
  48 |   });
  49 | 
  50 |   test('Edição inline de um campo do usuário', async ({ page }) => {
  51 |     // Pega o primeiro valor clicável (ex: Nome do primeiro card)
  52 |     const firstCell = page.locator('text=Teste 1').first();
  53 |     await expect(firstCell).toBeVisible({ timeout: 10000 });
  54 |     await firstCell.click();
  55 |     
  56 |     // Agora o input deve aparecer focado e podemos preenchê-lo
  57 |     const activeInput = page.locator('input').first();
  58 |     await activeInput.fill('Novo Valor Editado Playwright');
  59 |     await activeInput.press('Enter');
  60 |     
  61 |     // Verifica se a UI mantém o novo valor
  62 |     await expect(page.locator('text=Novo Valor Editado Playwright').first()).toBeVisible();
  63 |   });
  64 | });
  65 | 
```