# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: 01-home.spec.ts >> Suite 1: Home Page & Initial Setup >> A página carrega corretamente e exibe o título principal
- Location: tests/01-home.spec.ts:4:7

# Error details

```
Error: page.goto: net::ERR_CONNECTION_REFUSED at http://localhost:34115/
Call log:
  - navigating to "http://localhost:34115/", waiting until "load"

```

# Test source

```ts
  1  | import { test, expect } from '@playwright/test';
  2  | 
  3  | test.describe('Suite 1: Home Page & Initial Setup', () => {
  4  |   test('A página carrega corretamente e exibe o título principal', async ({ page }) => {
> 5  |     await page.goto('/');
     |                ^ Error: page.goto: net::ERR_CONNECTION_REFUSED at http://localhost:34115/
  6  |     
  7  |     // Verifica se o título "Candidate Allocator" aparece na home
  8  |     const header = page.locator('h1');
  9  |     await expect(header).toHaveText('Candidate Allocator');
  10 |   });
  11 | 
  12 |   test('Deve interagir com a funcionalidade Greet do Wails', async ({ page }) => {
  13 |     await page.goto('/');
  14 |     
  15 |     const nameInput = page.locator('#name');
  16 |     await nameInput.fill('Playwright Agent');
  17 |     
  18 |     const greetButton = page.locator('button', { hasText: 'Greet' });
  19 |     await greetButton.click();
  20 |     
  21 |     // A interface deve atualizar a mensagem com o nome passado
  22 |     const resultDiv = page.locator('#result');
  23 |     await expect(resultDiv).toContainText('Playwright Agent');
  24 |   });
  25 | 
  26 |   test('Deve exibir erro ao tentar prosseguir sem selecionar um arquivo Excel', async ({ page }) => {
  27 |     await page.goto('/');
  28 |     
  29 |     const fileButton = page.locator('button', { hasText: 'Executar função de arquivo' });
  30 |     await fileButton.click();
  31 |     
  32 |     const errorMsg = page.locator('#fileResult');
  33 |     await expect(errorMsg).toHaveText('Por favor, selecione um arquivo.');
  34 |     await expect(errorMsg).toBeVisible();
  35 |   });
  36 | });
  37 | 
```