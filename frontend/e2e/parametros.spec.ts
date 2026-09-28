import { test, expect } from "@playwright/test";
import { cartaoPrevia, prepararSessao, usarSessao } from "./apoio";

// Todos começam na tela de parâmetros com a planilha oficial já importada
// (19 avaliadores, 98 candidatos, 9 horários).
test.beforeEach(async ({ page, request }) => {
	await usarSessao(page, await prepararSessao(request));
	await page.goto("/parametros");
	await expect(page.getByRole("heading", { name: "Parâmetros da Alocação" })).toBeVisible();
});

test("prévia com os valores padrão", async ({ page }) => {
	await expect(page.getByLabel("Mesas por horário")).toHaveValue("5");
	await expect(page.getByLabel("Avaliadores por mesa")).toHaveValue("5");
	await expect(page.getByLabel("Mínimo de candidatos por mesa")).toHaveValue("5");
	await expect(page.getByLabel("Máximo de candidatos por mesa")).toHaveValue("8");

	// 19 avaliadores / 5 por mesa = 3 mesas por horário, das 5 pedidas
	await expect(cartaoPrevia(page, "mesas por horário")).toContainText("3");
	await expect(cartaoPrevia(page, "mesas por horário")).toContainText("de 5 pedidas");
	await expect(cartaoPrevia(page, "vagas no total")).toContainText("216");
	await expect(cartaoPrevia(page, "cabem no máximo")).toContainText("98");
	await expect(page.getByText("Com 19 avaliadores cabem só 3 mesas por horário")).toBeVisible();

	// tabela por horário, em ordem de dia da semana
	const linhas = page.locator("tbody tr");
	await expect(linhas).toHaveCount(9);
	await expect(linhas.first()).toContainText("segunda 8-10");
	await expect(linhas.last()).toContainText("sexta 8-10");

	// com os padrões, não há o que restaurar
	await expect(page.getByRole("button", { name: "Restaurar valores padrão" })).toHaveCount(0);
});

test("mínimo maior que o máximo bloqueia a alocação", async ({ page }) => {
	await page.getByLabel("Mínimo de candidatos por mesa").fill("9");
	await expect(page.getByText("o mínimo de candidatos por mesa (9) não pode ser maior que o máximo (8)")).toBeVisible();
	await expect(page.getByRole("button", { name: "Rodar alocação" })).toBeDisabled();

	await page.getByLabel("Mínimo de candidatos por mesa").fill("");
	await expect(page.getByText("Preencha todos os campos com números inteiros.")).toBeVisible();
	await expect(page.getByRole("button", { name: "Rodar alocação" })).toBeDisabled();

	await page.getByLabel("Mínimo de candidatos por mesa").fill("5");
	await expect(page.getByRole("button", { name: "Rodar alocação" })).toBeEnabled();
});

test("prévia acompanha as mudanças e restaura os padrões", async ({ page }) => {
	// com 3 avaliadores por mesa cabem as 5 mesas: some o aviso
	await page.getByLabel("Avaliadores por mesa").fill("3");
	await expect(cartaoPrevia(page, "mesas por horário")).not.toContainText("pedidas");
	await expect(cartaoPrevia(page, "vagas no total")).toContainText("360");
	await expect(page.getByText("Há mesas e vagas para todos os candidatos")).toBeVisible();

	// 1 mesa de até 5 por horário: 45 vagas para 98 candidatos
	await page.getByLabel("Mesas por horário").fill("1");
	await page.getByLabel("Máximo de candidatos por mesa").fill("5");
	await expect(cartaoPrevia(page, "vagas no total")).toContainText("45");
	await expect(page.getByText(/no máximo \d+ dos 98 candidatos cabem nas mesas/)).toBeVisible();

	await page.getByRole("button", { name: "Restaurar valores padrão" }).click();
	await expect(page.getByLabel("Mesas por horário")).toHaveValue("5");
	await expect(page.getByLabel("Avaliadores por mesa")).toHaveValue("5");
	await expect(page.getByLabel("Máximo de candidatos por mesa")).toHaveValue("8");
	await expect(page.getByRole("button", { name: "Restaurar valores padrão" })).toHaveCount(0);
});

test("alocação usa os parâmetros escolhidos e Voltar os mantém", async ({ page }) => {
	await page.getByLabel("Avaliadores por mesa").fill("3");
	await page.getByLabel("Mínimo de candidatos por mesa").fill("4");
	await expect(page.getByText("Há mesas e vagas para todos os candidatos")).toBeVisible();
	await page.getByRole("button", { name: "Rodar alocação" }).click();

	await expect(page.getByText("98 alocados")).toBeVisible({ timeout: 30_000 });
	// cada mesa com 3 avaliadores e 4 a 8 candidatos
	const mesas = page.getByTestId("mesa");
	const n = await mesas.count();
	expect(n).toBeGreaterThan(0);
	for (let i = 0; i < n; i++) {
		const mesa = mesas.nth(i);
		const [candidatos, avaliadores] = [mesa.getByTestId("candidatos").locator("li"), mesa.getByTestId("avaliadores").locator("li")];
		await expect(avaliadores).toHaveCount(3);
		const c = await candidatos.count();
		expect(c).toBeGreaterThanOrEqual(4);
		expect(c).toBeLessThanOrEqual(8);
	}

	await page.getByRole("button", { name: "Voltar" }).click();
	await expect(page).toHaveURL(/\/parametros$/);
	await expect(page.getByLabel("Avaliadores por mesa")).toHaveValue("3");
	await expect(page.getByLabel("Mínimo de candidatos por mesa")).toHaveValue("4");
});

test("critérios adicionais: validação, prévia, resultado e Voltar", async ({ page }) => {
	const adicionar = page.getByRole("button", { name: "Adicionar critério" });
	const criterios = page.getByTestId("criterio");

	// "pelo menos um de cada" sem valores não roda
	await adicionar.click();
	await criterios.nth(0).getByLabel("Regra").selectOption("um_de_cada");
	await expect(page.getByText(/escolha os valores de curso/)).toBeVisible();
	await expect(page.getByRole("button", { name: "Rodar alocação" })).toBeDisabled();

	// escolher Direito libera e a prévia avisa que não há Direito para todas as mesas
	await criterios.nth(0).getByRole("button", { name: /^Direito/ }).click();
	await expect(criterios.nth(0).getByRole("button", { name: /^Direito/ })).toHaveAttribute("aria-pressed", "true");
	await expect(page.getByText(/têm curso "Direito"/)).toBeVisible();
	await expect(page.getByRole("button", { name: "Rodar alocação" })).toBeEnabled();

	// segundo critério: misturar semestre com importância alta
	await adicionar.click();
	await criterios.nth(1).getByLabel("Coluna").selectOption("semestre");
	await criterios.nth(1).getByLabel("Importância").selectOption("10");
	await expect(criterios.nth(1)).toContainText("Evita pôr na mesma mesa candidatos com o mesmo semestre");

	await page.getByRole("button", { name: "Rodar alocação" }).click();
	await expect(page.getByText("98 alocados")).toBeVisible({ timeout: 30_000 });
	await expect(page.getByTestId("qualidade-criterio_1")).toContainText("Mesas sem todos os cursos escolhidos");
	await expect(page.getByTestId("qualidade-criterio_2")).toContainText("Mesas com semestre repetido");

	// o item de um critério destaca os candidatos dele, como os outros
	const item = page.getByTestId("qualidade-criterio_1");
	const n = parseInt((await item.innerText()).trim());
	if (n > 0) {
		await item.click();
		await expect(page.getByTestId("contagem-filtro")).toContainText("Mesas sem todos os cursos escolhidos");
		await expect(page.getByTestId("mesa")).toHaveCount(n);
	}

	// voltar mantém os critérios
	await page.getByRole("button", { name: "Voltar" }).click();
	await expect(criterios).toHaveCount(2);
	await expect(criterios.nth(0).getByLabel("Regra")).toHaveValue("um_de_cada");
	await expect(criterios.nth(1).getByLabel("Importância")).toHaveValue("10");
});
