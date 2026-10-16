import { test, expect } from "@playwright/test";
import { PLANILHA_OFICIAL, linhaMapeamento } from "./apoio";

const HORARIOS = ["segunda 14-16", "terça 10-12", "quarta 16-18", "quinta 8-10", "sexta 14-16"];

// Sem preferência: quem organiza digita os horários e as colunas de opção da
// planilha são ignoradas. Todo candidato pode ir em qualquer horário.
test("fluxo com horários definidos na tela", async ({ page }) => {
	await page.goto("/");
	await page.locator('input[type="file"]').setInputFiles(PLANILHA_OFICIAL);
	await page.getByRole("radio", { name: "Eu defino" }).click();
	await expect(page.getByText("Opções de horário")).toHaveCount(0);

	// sem horários, não deixa seguir
	await page.getByRole("button", { name: "Iniciar →" }).click();
	await expect(page.getByText("Informe pelo menos um horário")).toBeVisible();

	// repetidos e caixa diferente contam uma vez só
	await page.getByRole("textbox", { name: /Horários das mesas/ }).fill([...HORARIOS, "Segunda 14-16"].join("\n"));
	await expect(page.getByText("5 horário(s)")).toBeVisible();
	await page.getByRole("button", { name: "Iniciar →" }).click();

	// mapeamento sem as colunas de opção
	await expect(page).toHaveURL(/\/mapping$/);
	await expect(linhaMapeamento(page, "nome")).toHaveCount(1);
	await expect(linhaMapeamento(page, "opcao 1")).toHaveCount(0);
	await page.getByRole("button", { name: "Continuar para a revisão" }).click();

	await expect(page.getByTestId("cartao-candidato")).toHaveCount(98);
	await page.getByRole("button", { name: "Salvar Candidatos" }).first().click();
	await page.getByRole("button", { name: "Processar Avaliadores" }).click();
	await page.getByRole("button", { name: "Salvar avaliadores" }).click();
	await page.getByRole("button", { name: "Processar Restrições" }).click();
	await page.getByRole("button", { name: "Salvar restrições" }).click();

	// prévia: os 5 horários digitados, sem a coluna de 1ª opção
	await expect(page).toHaveURL(/\/parametros$/);
	for (const h of HORARIOS) {
		await expect(page.getByRole("cell", { name: h, exact: true })).toBeVisible();
	}
	await expect(page.getByRole("columnheader", { name: "1ª opção" })).toHaveCount(0);
	await expect(page.getByText("nos horários definidos")).toBeVisible();
	await page.getByRole("button", { name: "Rodar alocação" }).click();

	// resultado: todos alocados, sem "Nª opção" em lugar nenhum
	await expect(page).toHaveURL(/\/resultado$/);
	await expect(page.getByText("98 alocados")).toBeVisible({ timeout: 30_000 });
	await expect(page.getByText(/ª opção/)).toHaveCount(0);
	for (const nome of await page.getByTestId("nome-mesa").allTextContents()) {
		expect(HORARIOS.some((h) => nome.startsWith(`${h} - mesa`)), nome).toBeTruthy();
	}

	// detalhes de um candidato: só a mesa, sem a lista de horários escolhidos
	await page.getByTestId("mesa").first().getByTestId("candidatos").getByRole("button").first().click();
	const detalhes = page.getByRole("dialog");
	await expect(detalhes.getByText("Alocado em")).toBeVisible();
	await expect(detalhes.getByText("Horários que escolheu")).toHaveCount(0);
});
