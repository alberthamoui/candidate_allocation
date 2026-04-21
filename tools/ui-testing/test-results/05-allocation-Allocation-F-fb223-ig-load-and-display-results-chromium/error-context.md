# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: 05-allocation.spec.ts >> Allocation Flow UI >> should config, load and display results
- Location: tests/05-allocation.spec.ts:4:7

# Error details

```
Error: page.goto: net::ERR_CONNECTION_REFUSED at http://localhost:5173/success
Call log:
  - navigating to "http://localhost:5173/success", waiting until "load"

```

# Test source

```ts
  1  | import { test, expect } from '@playwright/test';
  2  | 
  3  | test.describe('Allocation Flow UI', () => {
  4  |   test('should config, load and display results', async ({ page }) => {
  5  |     // Start from success
> 6  |     await page.goto('http://localhost:5173/success');
     |                ^ Error: page.goto: net::ERR_CONNECTION_REFUSED at http://localhost:5173/success
  7  |     
  8  |     // Go to config
  9  |     const btn = page.locator('text=Configurar Alocação');
  10 |     await expect(btn).toBeVisible();
  11 |     await btn.click();
  12 | 
  13 |     // Check config page
  14 |     await expect(page.locator('text=Configurações de Alocação')).toBeVisible();
  15 |     await page.locator('input[type="number"]').nth(0).fill('3');
  16 |     
  17 |     // Add a soft criterion test
  18 |     await page.locator('text=Adicionar').click();
  19 |     await expect(page.locator('text=Tipo de Regra')).toBeVisible();
  20 |     await page.locator('input[type="text"]').nth(0).fill('curso');
  21 |     await page.locator('input[type="text"]').nth(1).fill('Computação');
  22 |     
  23 |     await page.locator('text=Iniciar Simulação').click();
  24 | 
  25 |     // Loading page
  26 |     await expect(page.locator('text=jeitos possíveis de alocação')).toBeVisible({ timeout: 10000 });
  27 |     
  28 |     // Result page
  29 |     await expect(page.locator('text=Resultados da Alocação')).toBeVisible({ timeout: 15000 });
  30 |     
  31 |     // Filters
  32 |     await expect(page.locator('text=Destacar Candidatos')).toBeVisible();
  33 |     
  34 |     // Unallocated box
  35 |     await expect(page.locator('text=Não Alocados')).toBeVisible();
  36 |   });
  37 | });
  38 | 
```