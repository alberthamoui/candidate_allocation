import { test, expect } from "@playwright/test";
import { fileURLToPath } from "node:url";
import { PLANILHA_EXEMPLO, PLANILHA_OFICIAL, colunaMapeada, linhaMapeamento } from "./apoio";

async function enviar(page: import("@playwright/test").Page, planilha: string) {
	await page.goto("/");
	await page.locator('input[type="file"]').setInputFiles(planilha);
	await page.getByRole("button", { name: "Iniciar →" }).click();
	await expect(page).toHaveURL(/\/mapping$/);
}

test("sugestão pelo nome com cabeçalhos diferentes dos campos", async ({ page }) => {
	await enviar(page, PLANILHA_EXEMPLO);
	await expect(colunaMapeada(page, "timestamp")).toContainText("TimeStamp");
	await expect(colunaMapeada(page, "email_insper")).toContainText("Email Insper");
	await expect(colunaMapeada(page, "opcao 1")).toContainText("Primeira Opção");
	await expect(colunaMapeada(page, "opcao 4")).toContainText("Quarta Opção");
	// a planilha tem 4 opções e o app conta pelo cabeçalho: não pede a 5ª
	await expect(linhaMapeamento(page, "opcao 5")).toHaveCount(0);
});

test("arrastar troca as colunas entre dois campos", async ({ page }) => {
	await enviar(page, PLANILHA_OFICIAL);
	const nome = colunaMapeada(page, "nome");
	const cpf = colunaMapeada(page, "cpf");
	await expect(nome).toHaveText("Nome");
	await expect(cpf).toHaveText("CPF");

	await nome.dragTo(cpf);
	await expect(nome).toHaveText("CPF");
	await expect(cpf).toHaveText("Nome");

	// a troca chega na revisão: o CPF passa a ter o nome da pessoa e dá erro
	await page.getByRole("button", { name: "Continuar para a revisão" }).click();
	await expect(page).toHaveURL(/\/verify$/);
	await expect(page.getByText("cpf inválido").first()).toBeVisible();
});

test("avaliadores e restrições com cabeçalhos diferentes", async ({ page }) => {
	await enviar(page, PLANILHA_EXEMPLO);
	await page.getByRole("button", { name: "Continuar para a revisão" }).click();
	await page.getByRole("button", { name: "Salvar Candidatos" }).first().click();

	await page.getByRole("button", { name: "Processar Avaliadores" }).click();
	await expect(colunaMapeada(page, "nome")).toHaveText("Avaliador");
	await page.getByRole("button", { name: "Salvar avaliadores" }).click();

	await page.getByRole("button", { name: "Processar Restrições" }).click();
	await expect(colunaMapeada(page, "candidato")).toContainText("CANDIDATOS");
	await expect(colunaMapeada(page, "naoPosso")).toContainText("NÃO POSSO");
	await expect(colunaMapeada(page, "prefiroNao")).toContainText("PREFIRO NÃO");
});

test("erro ao salvar avaliadores aparece na tela", async ({ page }) => {
	await enviar(page, fileURLToPath(new URL("./planilhas/avaliador_repetido.xlsx", import.meta.url)));
	await page.getByRole("button", { name: "Continuar para a revisão" }).click();
	await page.getByRole("button", { name: "Salvar Candidatos" }).first().click();

	await page.getByRole("button", { name: "Processar Avaliadores" }).click();
	await page.getByRole("button", { name: "Salvar avaliadores" }).click();
	// dois avaliadores com o mesmo nome: nada é salvo e a tela avisa
	await expect(page.getByText(/avaliador "Ana Souza" \(sigla "AS2"\) não foi salvo: nome ou email repetido/)).toBeVisible();
	await expect(page).toHaveURL(/\/mapping-avaliador$/);
});
