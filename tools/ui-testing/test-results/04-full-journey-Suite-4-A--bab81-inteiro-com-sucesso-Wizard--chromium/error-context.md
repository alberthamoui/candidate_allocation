# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: 04-full-journey.spec.ts >> Suite 4: A Jornada Completa (Happy Path) >> Deve completar o fluxo inteiro com sucesso (Wizard)
- Location: tests/04-full-journey.spec.ts:7:7

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
  4  | test.describe('Suite 4: A Jornada Completa (Happy Path)', () => {
  5  |   const sampleFilePath = path.resolve(__dirname, '../../../Execelteste/Base4Restricao.xlsx');
  6  | 
  7  |   test('Deve completar o fluxo inteiro com sucesso (Wizard)', async ({ page }) => {
  8  |     // 1. Home -> Upload
> 9  |     await page.goto('/');
     |                ^ Error: page.goto: net::ERR_CONNECTION_REFUSED at http://localhost:34115/
  10 |     await page.setInputFiles('#fileInput', sampleFilePath);
  11 |     await page.click('button:has-text("Executar função de arquivo")');
  12 |     
  13 |     // 2. Mapeamento Candidatos -> Verify Candidatos
  14 |     await expect(page.locator('h1', { hasText: 'Mapeamento de Candidatos' })).toBeVisible({ timeout: 15000 });
  15 |     await page.waitForTimeout(1000);
  16 |     await page.click('button:has-text("Revisar candidatos")');
  17 |     
  18 |     // 3. Verify Candidatos -> Mapeamento Restrições
  19 |     await expect(page.locator('h1', { hasText: 'Verificação de Usuários' })).toBeVisible({ timeout: 15000 });
  20 |     
  21 |     // Resolver todas as duplicatas se houver
  22 |     let resolveBtns = page.locator('button:has-text("Aceitar Este")');
  23 |     while (await resolveBtns.count() > 0) {
  24 |         const btn = resolveBtns.first();
  25 |         if (await btn.isVisible()) {
  26 |             await btn.click();
  27 |         }
  28 |         await page.waitForTimeout(200); // Wait for React state
  29 |         resolveBtns = page.locator('button:has-text("Aceitar Este")');
  30 |     }
  31 |     await page.click('button:has-text("Salvar Dados")');
  32 |     
  33 |     // 4. Mapeamento Restrições -> Verify Restrições
  34 |     await expect(page.locator('h1', { hasText: 'Mapeamento de Restricoes' })).toBeVisible({ timeout: 15000 });
  35 |     await page.waitForTimeout(1000);
  36 |     await page.click('button:has-text("Revisar restricoes")');
  37 | 
  38 |     // 5. Verify Restrições -> Mapeamento Avaliadores
  39 |     await expect(page.locator('h1', { hasText: 'Verificação de Restrições' })).toBeVisible({ timeout: 15000 });
  40 |     
  41 |     // Resolver todas as duplicatas de restrições se houver
  42 |     resolveBtns = page.locator('button:has-text("Aceitar Este")');
  43 |     while (await resolveBtns.count() > 0) {
  44 |         const btn = resolveBtns.first();
  45 |         if (await btn.isVisible()) {
  46 |             await btn.click();
  47 |         }
  48 |         await page.waitForTimeout(200);
  49 |         resolveBtns = page.locator('button:has-text("Aceitar Este")');
  50 |     }
  51 |     await page.click('button:has-text("Salvar Dados")');
  52 | 
  53 |     // 6. Mapeamento Avaliadores -> Verify Avaliadores
  54 |     await expect(page.locator('h1', { hasText: 'Mapeamento de Avaliadores' })).toBeVisible({ timeout: 15000 });
  55 |     await page.waitForTimeout(1000);
  56 |     await page.click('button:has-text("Revisar avaliadores")');
  57 | 
  58 |     // 7. Verify Avaliadores -> Success
  59 |     await expect(page.locator('h1', { hasText: 'Verificação de Avaliadores' })).toBeVisible({ timeout: 15000 });
  60 |     
  61 |     // Resolver todas as duplicatas de avaliadores se houver
  62 |     resolveBtns = page.locator('button:has-text("Aceitar Este")');
  63 |     while (await resolveBtns.count() > 0) {
  64 |         const btn = resolveBtns.first();
  65 |         if (await btn.isVisible()) {
  66 |             await btn.click();
  67 |         }
  68 |         await page.waitForTimeout(200);
  69 |         resolveBtns = page.locator('button:has-text("Aceitar Este")');
  70 |     }
  71 |     await page.click('button:has-text("Salvar Dados")');
  72 | 
  73 |     // 8. Success Page
  74 |     await expect(page.locator('h1', { hasText: 'Tudo Pronto!' })).toBeVisible({ timeout: 15000 });
  75 |     await expect(page.locator('text=Os candidatos e restrições foram salvos com sucesso')).toBeVisible();
  76 |     
  77 |     // Validate the button to return to home
  78 |     const restartBtn = page.locator('button:has-text("Nova Importação")');
  79 |     await expect(restartBtn).toBeVisible();
  80 |   });
  81 | });
  82 | 
```