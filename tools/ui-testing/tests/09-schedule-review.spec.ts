import {test,expect} from '@playwright/test';
import {completeImportWizard,reviewSchedules,importSampleSpreadsheet,confirmMapping,resolveDuplicateCards} from './support/ui';

test('falha de gravação recebida como texto continua visível para correção',async({page})=>{
 await importSampleSpreadsheet(page);
 await confirmMapping(page,'Verificação de Usuários');
 await resolveDuplicateCards(page);
 await page.evaluate(()=>{(window as any).go.main.App.SaveUsuariosFromMaps=async()=>{throw 'candidatos, registro 2: campo semestre precisa ser numérico';};});
 await page.getByTestId('verification-save-button').click();
 await expect(page.getByText('Erro ao salvar dados: candidatos, registro 2: campo semestre precisa ser numérico',{exact:true})).toBeVisible();
 await page.getByTestId('verification-error-close-button').click();
 await expect(page.getByTestId('verification-save-button')).toBeVisible();
 await expect(page).toHaveURL(/\/verify$/);
});

test('revisão de horários mostra erros e mantém a configuração editável',async({page})=>{
 await completeImportWizard(page);
 await page.getByTestId('configure-allocation-button').click();
 await reviewSchedules(page);
 await page.getByLabel('Intervalo 1',{exact:true}).fill('16:00-14:00');
 await page.getByTestId('start-allocation-button').click();
 await expect(page.getByRole('alert')).toContainText('fim do intervalo');
 await expect(page.getByTestId('schedule-review')).toBeVisible();
 await page.getByLabel('Intervalo 1',{exact:true}).fill('14:00-16:00');
 await page.getByTestId('min-people-per-group-input').fill('1');
 await page.getByTestId('evaluators-per-group-input').fill('1');
 await page.getByTestId('start-allocation-button').click();
 await expect(page.getByTestId('allocation-result-page')).toBeVisible({timeout:30000});
 if(await page.getByTestId('stop-search-button').isVisible()) await page.getByTestId('stop-search-button').click();
});

test('rodada sem preferências permite informar dia e intervalo separadamente',async({page})=>{
 await completeImportWizard(page,{preferences:false});
 await page.getByTestId('configure-allocation-button').click();
 await page.getByLabel('Dia ou data 1',{exact:true}).fill('04/11/2026');
 await page.getByLabel('Intervalo 1',{exact:true}).fill('14-16');
 await page.getByRole('button',{name:'Adicionar horário',exact:true}).click();
 await page.getByLabel('Dia ou data 2',{exact:true}).fill('sexta');
 await page.getByLabel('Intervalo 2',{exact:true}).fill('10-12');
 await page.getByRole('button',{name:'Remover horário 2',exact:true}).click();
 await expect(page.getByTestId('schedule-mapping-row')).toHaveCount(1);
 await page.getByTestId('min-people-per-group-input').fill('1');
 await page.getByTestId('evaluators-per-group-input').fill('1');
 await page.getByTestId('start-allocation-button').click();
 await expect(page.getByTestId('allocation-result-page')).toBeVisible({timeout:30000});
 if(await page.getByTestId('stop-search-button').isVisible()) await page.getByTestId('stop-search-button').click();
 await expect(page.getByRole('heading',{name:'Horário: 2026-11-04 14:00-16:00',exact:true})).toBeVisible();
});
