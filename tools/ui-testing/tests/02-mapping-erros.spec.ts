import { test, expect } from '@playwright/test';
import { expectMainHeading, importSampleSpreadsheet } from './support/ui';

test.describe('Suite 2: Regras e Erros de Mapeamento (MappingEditorPage)', () => {
  test.beforeEach(async ({ page }) => {
    await importSampleSpreadsheet(page);
    await expect(page.getByText('Mapeie primeiro os atributos estruturais.')).toHaveCount(0);
    await expect(page.getByText('Use extras para não perder informação útil')).toHaveCount(0);
    await expect(page.getByText('O inventário restante permite revisar rapidamente')).toHaveCount(0);
    await expectMainHeading(page, 'Mapeamento de Candidatos');
  });

  test('Deve exibir erro ao tentar prosseguir com Campo Extra Vazio', async ({ page }) => {
    await page.getByTestId('add-extra-button').click();
    await page.getByTestId('mapping-confirm-button').click();

    await expect(page.getByTestId('mapping-error-dialog')).toContainText(
      'Todo campo extra criado manualmente precisa ter um nome antes de seguir.'
    );

    await page.getByTestId('mapping-error-close-button').click();
    await expect(page.getByTestId('mapping-error-dialog')).toBeHidden();
  });

  test('Deve impedir Campo Extra com mesmo nome de um Campo Principal (Core Field)', async ({ page }) => {
    await page.getByTestId('add-extra-button').click();
    await page.getByTestId('extra-field-name-input').first().fill('nome');
    await page.getByTestId('mapping-confirm-button').click();

    await expect(page.getByTestId('mapping-error-dialog')).toContainText('conflita com um campo principal');
  });

  test('Deve impedir dois Campos Extras gerando a mesma chave normalizada', async ({ page }) => {
    await page.getByTestId('add-extra-button').click();
    await page.getByTestId('add-extra-button').click();

    const inputs = page.getByTestId('extra-field-name-input');
    await inputs.first().fill('meu campo');
    await inputs.last().fill('Meu-Campo');

    await page.getByTestId('mapping-confirm-button').click();

    await expect(page.getByTestId('mapping-error-dialog')).toContainText('geram a mesma chave');
  });
});
