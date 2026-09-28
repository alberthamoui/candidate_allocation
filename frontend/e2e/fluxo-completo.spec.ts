import { test, expect } from "@playwright/test";
import { statSync } from "node:fs";
import { PLANILHA_OFICIAL, linhaMapeamento } from "./apoio";

// O caminho inteiro pela interface, com a planilha oficial e as sugestões
// aceitas como vieram: upload → 3 abas → parâmetros → resultado → exportação.
test("fluxo completo com a planilha oficial", async ({ page }) => {
	await page.goto("/");
	await page.locator('input[type="file"]').setInputFiles(PLANILHA_OFICIAL);
	await page.getByRole("button", { name: "Iniciar →" }).click();

	// Etapa 1: mapeamento de candidatos sugerido pelo nome das colunas
	await expect(page).toHaveURL(/\/mapping$/);
	await expect(linhaMapeamento(page, "nome")).toContainText("Nome");
	await expect(linhaMapeamento(page, "email_insper")).toContainText("Email Institucional");
	await expect(linhaMapeamento(page, "opcao 5")).toContainText("Opcao 5");
	await page.getByRole("button", { name: "Confirmar Mudanças" }).click();

	// Revisão: 98 candidatos, nenhum com erro
	await expect(page).toHaveURL(/\/verify$/);
	await expect(page.getByText(/^ID: \d+$/)).toHaveCount(98);
	await expect(page.getByText(/\d+ erros?$/)).toHaveCount(0);
	await page.getByRole("button", { name: "Salvar Candidatos" }).first().click();

	// Etapa 2: avaliadores
	await expect(page).toHaveURL(/\/upload-avaliador$/);
	await page.getByRole("button", { name: "Processar Avaliadores" }).click();
	await expect(linhaMapeamento(page, "sigla")).toContainText("Sigla");
	await page.getByRole("button", { name: "Confirmar Mudanças" }).click();

	// Etapa 3: restrições
	await expect(page).toHaveURL(/\/upload-restricao$/);
	await page.getByRole("button", { name: "Processar Restrições" }).click();
	await expect(linhaMapeamento(page, "naoPosso")).toContainText("NaoPosso");
	await page.getByRole("button", { name: "Confirmar Mudanças" }).click();

	// Etapa 4: parâmetros padrão
	await expect(page).toHaveURL(/\/parametros$/);
	await expect(page.getByText("98", { exact: true }).first()).toBeVisible();
	await page.getByRole("button", { name: "Rodar alocação" }).click();

	// Resultado: todos alocados, mesas com 5 a 8 candidatos e 5 avaliadores
	await expect(page).toHaveURL(/\/resultado$/);
	await expect(page.getByText("98 alocados")).toBeVisible({ timeout: 30_000 });
	await expect(page.getByText(/não alocados/)).toHaveCount(0);
	const tamanhos = await page.getByText(/^\d+ candidatos?$/).allTextContents();
	expect(tamanhos.length).toBeGreaterThan(0);
	const total = tamanhos.map((t) => parseInt(t)).reduce((a, b) => a + b, 0);
	expect(total).toBe(98);
	for (const t of tamanhos) {
		expect(parseInt(t)).toBeGreaterThanOrEqual(5);
		expect(parseInt(t)).toBeLessThanOrEqual(8);
	}

	// Exportação
	const [download] = await Promise.all([
		page.waitForEvent("download"),
		page.getByRole("button", { name: "Exportar Excel" }).click(),
	]);
	expect(download.suggestedFilename()).toBe("alocacao.xlsx");
	expect(statSync(await download.path()).size).toBeGreaterThan(1000);

	// Reiniciar volta para a tela inicial
	await page.getByRole("button", { name: "Reiniciar" }).click();
	await page.getByRole("button", { name: "Confirmar" }).click();
	await expect(page).toHaveURL(/\/$/);
	await expect(page.getByRole("heading", { name: "Candidate Allocator" })).toBeVisible();
});
