# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: 02-mapping-erros.spec.ts >> Suite 2: Regras e Erros de Mapeamento (MappingEditorPage) >> Deve impedir Campo Extra com mesmo nome de um Campo Principal (Core Field)
- Location: tests/02-mapping-erros.spec.ts:37:7

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
  4  | test.describe('Suite 2: Regras e Erros de Mapeamento (MappingEditorPage)', () => {
  5  |   const sampleFilePath = path.resolve(__dirname, '../../../Execelteste/Base4Restricao.xlsx');
  6  | 
  7  |   test.beforeEach(async ({ page }) => {
  8  |     // Acessar Home e fazer upload do arquivo Excel de testes
> 9  |     await page.goto('/');
     |                ^ Error: page.goto: net::ERR_CONNECTION_REFUSED at http://localhost:34115/
  10 |     
  11 |     // O Playwright permite upload headless por id
  12 |     await page.setInputFiles('#fileInput', sampleFilePath);
  13 |     await page.click('button:has-text("Executar função de arquivo")');
  14 |     
  15 |     // Esperar navegação para a página de Mapeamento
  16 |     const header = page.locator('h1', { hasText: 'Mapeamento de Candidatos' });
  17 |     await expect(header).toBeVisible({ timeout: 15000 });
  18 |   });
  19 | 
  20 |   test('Deve exibir erro ao tentar prosseguir com Campo Extra Vazio', async ({ page }) => {
  21 |     // Clicar em "Adicionar Extra"
  22 |     const addExtraBtn = page.locator('button:has-text("Adicionar Extra")');
  23 |     await addExtraBtn.click();
  24 |     
  25 |     // Tentar avançar
  26 |     const confirmBtn = page.locator('button:has-text("Revisar candidatos")');
  27 |     await confirmBtn.click();
  28 |     
  29 |     // O modal de erro com essa mensagem exata deve aparecer
  30 |     const errorModalText = page.locator('text=Todo campo extra criado manualmente precisa ter um nome antes de seguir.');
  31 |     await expect(errorModalText).toBeVisible();
  32 |     
  33 |     // Fechar o modal
  34 |     await page.click('button:has-text("Entendido")');
  35 |   });
  36 | 
  37 |   test('Deve impedir Campo Extra com mesmo nome de um Campo Principal (Core Field)', async ({ page }) => {
  38 |     // Adicionar novo extra
  39 |     await page.click('button:has-text("Adicionar Extra")');
  40 |     
  41 |     // A seção Extras tem inputs de texto - pegar o primeiro input
  42 |     const extraInput = page.locator('input[placeholder="Nome do campo extra"]').first();
  43 |     
  44 |     // Inserir um nome que normalmente é um core field (por exemplo "nome" ou "email")
  45 |     // Para simplificar e testar a normalização, usaremos "nome"
  46 |     await extraInput.fill('nome');
  47 |     await extraInput.blur(); // Perde o foco
  48 |     
  49 |     await page.click('button:has-text("Revisar candidatos")');
  50 |     
  51 |     const errorModalText = page.locator('text=conflita com um campo principal');
  52 |     await expect(errorModalText).toBeVisible();
  53 |   });
  54 | 
  55 |   test('Deve impedir dois Campos Extras gerando a mesma chave normalizada', async ({ page }) => {
  56 |     // Adicionar 2 campos extras
  57 |     await page.click('button:has-text("Adicionar Extra")');
  58 |     await page.click('button:has-text("Adicionar Extra")');
  59 |     
  60 |     const inputs = page.locator('input[placeholder="Nome do campo extra"]');
  61 |     
  62 |     // Preencher o primeiro
  63 |     await inputs.nth(0).fill('meu campo');
  64 |     
  65 |     // Preencher o segundo com um nome que gera o mesmo normalized_key: "Meu-Campo" -> "meu_campo"
  66 |     await inputs.nth(1).fill('Meu-Campo');
  67 |     
  68 |     await page.click('button:has-text("Revisar candidatos")');
  69 |     
  70 |     const errorModalText = page.locator('text=geram a mesma chave');
  71 |     await expect(errorModalText).toBeVisible();
  72 |   });
  73 | });
  74 | 
```