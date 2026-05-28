import { test, expect } from '@playwright/test';
import { completeImportWizard } from './support/ui';

test.describe('Suite 4: A Jornada Completa (Happy Path)', () => {
  test('Deve completar o fluxo inteiro com sucesso (Wizard)', async ({ page }) => {
    await completeImportWizard(page);

    await expect(page.getByTestId('restart-import-button')).toBeVisible();
  });
});
