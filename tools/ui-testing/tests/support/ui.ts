import { expect, type Page } from '@playwright/test';
import path from 'path';

export const sampleFilePath = path.resolve(
  process.cwd(),
  '../../Execelteste/Base4Restricao.xlsx'
);

export async function expectMainHeading(page: Page, name: string | RegExp) {
  await expect(page.getByRole('main').getByRole('heading', { name })).toBeVisible({
    timeout: 15000,
  });
}

export async function importSampleSpreadsheet(page: Page, options?: {preferences?: boolean}) {
  await page.goto('/');
 if(options?.preferences===false) await page.getByLabel('Candidatos informam preferências de horário').uncheck();
  await page.getByTestId('excel-file-input').setInputFiles(sampleFilePath);
  await page.getByTestId('start-import-button').click();
  await expectMainHeading(page, 'Mapeamento de Candidatos');
}

export async function confirmMapping(page: Page, nextHeading: string | RegExp) {
  await page.getByTestId('mapping-confirm-button').click();
  await expectMainHeading(page, nextHeading);
}

export async function resolveDuplicateCards(page: Page) {
  const acceptButtons = page.getByTestId('accept-duplicate-record-button');
  while ((await acceptButtons.count()) > 0) {
    const previousCount = await acceptButtons.count();
    await acceptButtons.first().click();
    await expect.poll(() => acceptButtons.count(), { timeout: 5000 }).toBeLessThan(previousCount);
  }
}

export async function saveVerification(page: Page, nextHeading: string | RegExp) {
  await page.getByTestId('verification-save-button').click();
  await expectMainHeading(page, nextHeading);
}

export async function resolveAndSaveVerification(page: Page, nextHeading: string | RegExp) {
  await resolveDuplicateCards(page);
  await saveVerification(page, nextHeading);
}

export async function completeImportWizard(page: Page, options?: {preferences?: boolean}) {
  await importSampleSpreadsheet(page,options);

  await confirmMapping(page, 'Verificação de Usuários');
  await resolveAndSaveVerification(page, 'Mapeamento de Restricoes');

  await confirmMapping(page, 'Verificação de Restrições');
  await saveVerification(page, 'Mapeamento de Avaliadores');

  await confirmMapping(page, 'Verificação de Avaliadores');
  await resolveDuplicateCards(page);
  await page.getByTestId('verification-save-button').click();
  await expect(page.getByTestId('success-page')).toBeVisible({ timeout: 15000 });
}

export async function openPageHelp(page: Page) {
  await page.getByTestId('page-help-button').click();
  await expect(page.getByTestId('page-help-dialog')).toBeVisible();
}

export async function reviewSchedules(page: Page) {
 const rows=page.getByTestId('schedule-mapping-row');
 await expect(rows.first()).toBeVisible();
 for(let i=0;i<await rows.count();i++) {
  const fields=rows.nth(i).locator('input');
  if(!(await fields.nth(0).inputValue())) await fields.nth(0).fill('quarta');
  if(!(await fields.nth(1).inputValue())) await fields.nth(1).fill(`${String(8+i%12).padStart(2,'0')}:00-${String(9+i%12).padStart(2,'0')}:00`);
 }
}
