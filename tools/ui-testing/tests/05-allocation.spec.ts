import { test, expect } from '@playwright/test';
import { completeImportWizard, openPageHelp } from './support/ui';

test.describe('Allocation Flow UI', () => {
  test('explains quality, highlights candidates and exposes person preferences', async ({ page }) => {
    await page.goto('/');
    await page.waitForFunction(() => Boolean((window as any).go?.main?.App && (window as any).runtime));

    await page.evaluate(() => {
      const app = (window as any).go.main.App;
      const runtime = (window as any).runtime;
      app.GetWorkflowDefinition = async () => ({
        defaultAllocationParams: {
          gruposPorHorario: 1,
          minPessoasPorGrupo: 1,
          maxPessoasPorGrupo: 3,
          avaliadoresPorGrupo: 1,
          softCriteria: [],
        },
      });
      app.BuildAllocationConfigurationFromDatabase = async (params: unknown) => ({ normalized: { params, preferenceMappings: [] } });
      const provisionalResult = {
        status: 'Sucesso!',
        solverStatus: 'optimal',
        mesas: [{
          id: 1,
          horario: 'Segunda 10h',
          descricao: 'Mesa 1',
          candidatos: [
            { id: 1, nome: 'Ana Lima', semestre: 2, curso: 'ADM', emailSecundario: 'ana@insper.edu.br', emailPessoal: '', opcoes: ['Segunda 10h', 'Terça 14h'], naoPosso: [], prefiroNao: [], extras: {} },
            { id: 2, nome: 'Bruno Reis', semestre: 4, curso: ' adm ', emailSecundario: 'bruno@insper.edu.br', emailPessoal: '', opcoes: ['Terça 14h', 'Segunda 10h'], naoPosso: [], prefiroNao: ['Profa. Eva (EV)'], extras: {} },
            { id: 3, nome: 'Carla Luz', semestre: 2, curso: 'ECO', emailSecundario: 'carla@insper.edu.br', emailPessoal: '', opcoes: ['Segunda 10h'], naoPosso: [], prefiroNao: [], extras: {} },
          ],
          avaliadores: [{ id: 10, nome: 'Profa. Eva', sigla: 'EV', email: 'eva@insper.edu.br', naoPosso: [], prefiroNao: ['Bruno Reis'], extras: {} }],
        }],
        naoAlocados: [],
        score: { totalPenalty: 4, components: [{ code: 'preference_rank', penalty: 1, message: 'Bruno foi alocado na preferência 2' }, { code: 'avoid_evaluator', penalty: 3, message: 'Bruno ficou com avaliador marcado como Prefiro não' }] },
        quality: { characteristics: [
          { code: 'preference_rank_1', label: 'Preferência 1 de horário', description: 'Candidatos alocados na 1ª opção.', value: 2, valueLabel: 'candidatos', penalty: 0, tone: 'success', candidateIds: [1, 3], evaluatorIds: [], groupIds: [1] },
          { code: 'preference_rank_2', label: 'Preferência 2 de horário', description: 'Candidatos alocados na 2ª opção.', value: 1, valueLabel: 'candidato', penalty: 1, tone: 'neutral', candidateIds: [2], evaluatorIds: [], groupIds: [1] },
          { code: 'avoid_evaluator', label: 'Alocações em “Prefiro não”', description: 'Preferência negativa.', value: 1, valueLabel: 'candidato', penalty: 3, tone: 'warning', candidateIds: [2], evaluatorIds: [10], groupIds: [1] },
        ] },
        hardViolations: [],
        metrics: { nodesVisited: 10, nodesPrunedByHard: 2, nodesPrunedByFlow: 1, branchesSkippedBySymmetry: 1 },
        debugNotes: [],
      };
      app.StopAllocation = async () => {
        runtime.EventsEmit('allocation:complete', { ...provisionalResult, status: 'Solução provisória', solverStatus: 'feasible' });
        return true;
      };
      app.StartAllocation = async () => {
        window.setTimeout(() => runtime.EventsEmit('allocation:progress', {
          percent: 25,
          branchesResolved: '250000',
          totalBranches: '1000000',
          branchesPruned: '200000',
          nodesVisited: 42,
          prunedSubtrees: 3,
        }), 10);
        window.setTimeout(() => runtime.EventsEmit('allocation:solution', provisionalResult), 20);
      };

      window.history.pushState({}, '', '/allocation-loading');
      window.dispatchEvent(new PopStateEvent('popstate'));
    });

    await expect(page.getByTestId('allocation-result-page')).toBeVisible();
    await expect(page.getByText('Solução válida encontrada — verificação continua')).toBeVisible();
    await expect(page.getByTestId('result-total-possibilities')).toHaveText('1.000.000');
    await expect(page.getByTestId('result-progress-percent')).toHaveText('25.0%');
    await page.getByTestId('continue-search-button').click();
    await expect(page.getByText('A verificação continuará; esta tela será atualizada quando surgir uma solução melhor.')).toBeVisible();
    await page.getByTestId('stop-search-button').click();
    await expect(page.getByText('Verificação interrompida')).toBeVisible();
    await expect(page.getByLabel('Pesquisar candidato')).toHaveAttribute('type', 'search');

    await page.getByTestId('quality-preference_rank_2').click();
    await expect(page.locator('[data-candidate-id="2"]')).toHaveClass(/ring-violet/);
    await expect(page.locator('[data-candidate-id="1"]')).toHaveClass(/opacity-45/);

    await page.getByTestId('course-filter-adm').click();
    await expect(page.locator('[data-candidate-id="1"]')).not.toHaveClass(/opacity-45/);
    await expect(page.locator('[data-candidate-id="2"]')).not.toHaveClass(/opacity-45/);
    await expect(page.locator('[data-candidate-id="3"]')).toHaveClass(/opacity-45/);

    await page.getByTestId('score-metric').hover();
    await expect(page.getByText('Como o score foi calculado')).toBeVisible();
    await expect(page.getByText('Bruno foi alocado na preferência 2')).toBeVisible();

    await page.getByRole('button', { name: /Ana Lima/ }).click();
    await expect(page.getByTestId('person-details-tooltip')).toContainText('Segunda 10h');
    await page.getByLabel('Fechar informações').click();
    await page.getByRole('button', { name: /Profa. Eva/ }).click();
    await expect(page.getByTestId('person-details-tooltip')).toContainText('Prefere não avaliar: Bruno Reis');

    const groupsTop = await page.getByText('Mesa 1', { exact: true }).evaluate((node) => node.getBoundingClientRect().top + window.scrollY);
    const unallocatedTop = await page.getByRole('heading', { name: 'Candidatos não alocados' }).evaluate((node) => node.getBoundingClientRect().top + window.scrollY);
    expect(unallocatedTop).toBeGreaterThan(groupsTop);
  });

  test('shows a fixed total of possibilities and live analyzed percentage', async ({ page }) => {
    await page.goto('/');
    await page.waitForFunction(() => Boolean((window as any).go?.main?.App && (window as any).runtime));

    await page.evaluate(() => {
      const app = (window as any).go.main.App;
      const runtime = (window as any).runtime;

      app.StartAllocation = async () => {
        window.setTimeout(() => runtime.EventsEmit('allocation:progress', {
            percent: 62.5,
            branchesResolved: '625000',
            totalBranches: '1000000',
            branchesPruned: '600000',
            nodesVisited: 128,
            prunedSubtrees: 7,
          }), 50);
      };

      app.GetWorkflowDefinition = async () => ({
        defaultAllocationParams: {
          gruposPorHorario: 2,
          minPessoasPorGrupo: 1,
          maxPessoasPorGrupo: 2,
          avaliadoresPorGrupo: 1,
          softCriteria: [],
        },
      });
      app.BuildAllocationConfigurationFromDatabase = async (params: unknown) => ({
        normalized: { params, preferenceMappings: [] },
      });
      window.history.pushState({}, '', '/allocation-loading');
      window.dispatchEvent(new PopStateEvent('popstate'));
    });

    const progressbar = page.getByRole('progressbar', { name: 'Progresso da alocação' });
    await expect(progressbar).toHaveAttribute('aria-valuenow', '62.5');
    await expect(page.getByTestId('allocation-progress-percent')).toHaveText('62.5%');
    await expect(page.getByTestId('allocation-total-possibilities')).toHaveText('1.000.000');
    await expect(page.getByText('Branches eliminadas por poda')).toHaveCount(0);
  });

  test('should config, load and display results', async ({ page }) => {
    await completeImportWizard(page);

    await page.getByTestId('configure-allocation-button').click();
    await expect(page).toHaveURL(/allocation-config/);

    await expect(page.getByTestId('workflow-main').getByRole('heading', { name: 'Parâmetros centrais' })).toBeVisible();
    await expect(page.getByTestId('groups-per-schedule-input')).toHaveValue('2');
    await expect(page.getByTestId('min-people-per-group-input')).toHaveValue('4');
    await expect(page.getByTestId('max-people-per-group-input')).toHaveValue('8');
    await expect(page.getByTestId('evaluators-per-group-input')).toHaveValue('3');
    await expect(page.getByText('Critérios adicionais', { exact: true })).toBeVisible();
    await page.getByTestId('groups-per-schedule-input').fill('3');
    await openPageHelp(page);
    await expect(page.getByTestId('page-help-dialog').getByText('Para que serve')).toBeVisible();
    await page.getByTestId('page-help-close-button').click();

    await page.getByTestId('add-criterion-button').click();
    await expect(page.getByTestId('criterion-type-select')).toBeVisible();
    await expect(page.getByTestId('criterion-type-select')).toHaveValue('min_value');
    await page.getByTestId('start-allocation-button').click();

    await expect(page.getByTestId('allocation-result-page')).toBeVisible({ timeout: 30000 });
    await expect(page.getByText('Diagnóstico técnico do solver', { exact: true })).toBeVisible();
    await expect(page.getByTestId('score-metric')).toBeVisible();
    await expect(page.getByRole('heading', { name: 'Qualidade da alocação' })).toBeVisible();
    await expect(page.getByRole('heading', { name: 'Encontrar e destacar candidatos' })).toBeVisible();
    await expect(page.getByRole('heading', { name: 'Candidatos não alocados' })).toBeVisible();
  });
});
