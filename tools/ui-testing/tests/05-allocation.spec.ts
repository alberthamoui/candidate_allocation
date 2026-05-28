import { test, expect } from '@playwright/test';
import { completeImportWizard, openPageHelp } from './support/ui';

test.describe('Allocation Flow UI', () => {
  test('should config, load and display results', async ({ page }) => {
    await completeImportWizard(page);

    await page.getByTestId('configure-allocation-button').click();
    await expect(page).toHaveURL(/allocation-config/);

    await expect(page.getByTestId('workflow-main').getByRole('heading', { name: 'Parâmetros centrais' })).toBeVisible();
    await expect(page.getByTestId('groups-per-schedule-input')).toHaveValue('2');
    await expect(page.getByTestId('min-people-per-group-input')).toHaveValue('4');
    await expect(page.getByTestId('max-people-per-group-input')).toHaveValue('8');
    await expect(page.getByTestId('evaluators-per-group-input')).toHaveValue('3');
    await page.getByTestId('groups-per-schedule-input').fill('3');
    await openPageHelp(page);
    await expect(page.getByTestId('page-help-dialog').getByText('Para que serve')).toBeVisible();
    await page.getByTestId('page-help-close-button').click();

    await page.getByTestId('add-criterion-button').click();
    await expect(page.getByTestId('criterion-type-select')).toBeVisible();
    await expect(page.getByTestId('criterion-type-select')).toHaveValue('min_value');
    await page.getByTestId('start-allocation-button').click();

    await expect(page.getByTestId('allocation-result-page')).toBeVisible({ timeout: 30000 });
    await expect(page.getByRole('heading', { name: 'Diagnóstico do solver' })).toBeVisible();
    await expect(page.getByText('Score', { exact: true })).toBeVisible();
    await expect(page.getByRole('heading', { name: 'Destacar Candidatos' })).toBeVisible();
    await expect(page.getByRole('heading', { name: 'Não Alocados' })).toBeVisible();
  });
});
