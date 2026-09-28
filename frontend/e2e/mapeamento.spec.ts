import { test, expect } from "@playwright/test";
import { fileURLToPath } from "node:url";
import { PLANILHA_EXEMPLO, PLANILHA_OFICIAL, linhaMapeamento } from "./apoio";

async function enviar(page: import("@playwright/test").Page, planilha: string) {
	await page.goto("/");
	await page.locator('input[type="file"]').setInputFiles(planilha);
	await page.getByRole("button", { name: "Iniciar →" }).click();
	await expect(page).toHaveURL(/\/mapping$/);
}

test("sugestão pelo nome com cabeçalhos diferentes dos campos", async ({ page }) => {
	await enviar(page, PLANILHA_EXEMPLO);
	await expect(linhaMapeamento(page, "timestamp")).toContainText("TimeStamp");
	await expect(linhaMapeamento(page, "email_insper")).toContainText("Email Insper");
	await expect(linhaMapeamento(page, "opcao 1")).toContainText("Primeira Opção");
	await expect(linhaMapeamento(page, "opcao 4")).toContainText("Quarta Opção");
	// a planilha só tem 4 opções: a 5ª fica sem coluna
	await expect(linhaMapeamento(page, "opcao 5").locator("[draggable=true]")).toHaveText("");
});

test("arrastar troca as colunas entre dois campos", async ({ page }) => {
	await enviar(page, PLANILHA_OFICIAL);
	const nome = linhaMapeamento(page, "nome").locator("[draggable=true]");
	const cpf = linhaMapeamento(page, "cpf").locator("[draggable=true]");
	await expect(nome).toHaveText("Nome");
	await expect(cpf).toHaveText("CPF");

	await nome.dragTo(cpf);
	await expect(nome).toHaveText("CPF");
	await expect(cpf).toHaveText("Nome");

	// a troca chega na revisão: o CPF passa a ter o nome da pessoa e dá erro
	await page.getByRole("button", { name: "Confirmar Mudanças" }).click();
	await expect(page).toHaveURL(/\/verify$/);
	await expect(page.getByText("cpf inválido").first()).toBeVisible();
});

test("avaliadores e restrições com cabeçalhos diferentes", async ({ page }) => {
	await enviar(page, PLANILHA_EXEMPLO);
	await page.getByRole("button", { name: "Confirmar Mudanças" }).click();
	await page.getByRole("button", { name: "Salvar Candidatos" }).first().click();

	await page.getByRole("button", { name: "Processar Avaliadores" }).click();
	await expect(linhaMapeamento(page, "nome")).toContainText("Avaliador");
	await page.getByRole("button", { name: "Confirmar Mudanças" }).click();

	await page.getByRole("button", { name: "Processar Restrições" }).click();
	await expect(linhaMapeamento(page, "candidato")).toContainText("CANDIDATOS");
	await expect(linhaMapeamento(page, "naoPosso")).toContainText("NÃO POSSO");
	await expect(linhaMapeamento(page, "prefiroNao")).toContainText("PREFIRO NÃO");
});

test("erro ao salvar avaliadores aparece na tela", async ({ page }) => {
	await enviar(page, fileURLToPath(new URL("./planilhas/avaliador_repetido.xlsx", import.meta.url)));
	await page.getByRole("button", { name: "Confirmar Mudanças" }).click();
	await page.getByRole("button", { name: "Salvar Candidatos" }).first().click();

	await page.getByRole("button", { name: "Processar Avaliadores" }).click();
	await page.getByRole("button", { name: "Confirmar Mudanças" }).click();
	// dois avaliadores com o mesmo nome: nada é salvo e a tela avisa
	await expect(page.getByText(/avaliador "Ana Souza" \(sigla "AS2"\) não foi salvo: nome ou email repetido/)).toBeVisible();
	await expect(page).toHaveURL(/\/mapping-avaliador$/);
});
