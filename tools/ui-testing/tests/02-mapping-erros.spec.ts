import { test, expect } from '@playwright/test';
import path from 'path';

test.describe('Suite 2: Regras e Erros de Mapeamento (MappingEditorPage)', () => {
  const sampleFilePath = path.resolve(__dirname, '../../../Execelteste/Base4Restricao.xlsx');

  test.beforeEach(async ({ page }) => {
    // Acessar Home e fazer upload do arquivo Excel de testes
    await page.goto('/');
    
    // O Playwright permite upload headless por id
    await page.setInputFiles('#fileInput', sampleFilePath);
    await page.click('button:has-text("Continuar")');
    
    // Esperar navegação para a página de Mapeamento
    const header = page.locator('h1', { hasText: 'Mapeamento de Candidatos' });
    await expect(header).toBeVisible({ timeout: 15000 });
  });

  test('Deve exibir erro ao tentar prosseguir com Campo Extra Vazio', async ({ page }) => {
    // Clicar em "Adicionar Extra"
    const addExtraBtn = page.getByTestId('add-extra-button');
    await addExtraBtn.click();
    
    // Tentar avançar
    const confirmBtn = page.getByTestId('mapping-confirm-button');
    await confirmBtn.click();
    
    // O modal de erro com essa mensagem exata deve aparecer
    const errorModalText = page.locator('text=Todo campo extra criado manualmente precisa ter um nome antes de seguir.');
    await expect(errorModalText).toBeVisible();
    
    // Fechar o modal
    await page.click('button:has-text("Entendido")');
  });

  test('Deve impedir Campo Extra com mesmo nome de um Campo Principal (Core Field)', async ({ page }) => {
    // Adicionar novo extra
    await page.getByTestId('add-extra-button').click();
    
    // A seção Extras tem inputs de texto - pegar o primeiro input
    const extraInput = page.locator('input[placeholder="Nome do campo extra"]').first();
    
    // Inserir um nome que normalmente é um core field (por exemplo "nome" ou "email")
    // Para simplificar e testar a normalização, usaremos "nome"
    await extraInput.fill('nome');
    await extraInput.blur(); // Perde o foco
    
    await page.getByTestId('mapping-confirm-button').click();
    
    const errorModalText = page.locator('text=conflita com um campo principal');
    await expect(errorModalText).toBeVisible();
  });

  test('Deve impedir dois Campos Extras gerando a mesma chave normalizada', async ({ page }) => {
    // Adicionar 2 campos extras
    await page.getByTestId('add-extra-button').click();
    await page.getByTestId('add-extra-button').click();
    
    const inputs = page.locator('input[placeholder="Nome do campo extra"]');
    
    // Preencher o primeiro
    await inputs.nth(0).fill('meu campo');
    
    // Preencher o segundo com um nome que gera o mesmo normalized_key: "Meu-Campo" -> "meu_campo"
    await inputs.nth(1).fill('Meu-Campo');
    
    await page.getByTestId('mapping-confirm-button').click();
    
    const errorModalText = page.locator('text=geram a mesma chave');
    await expect(errorModalText).toBeVisible();
  });
});
