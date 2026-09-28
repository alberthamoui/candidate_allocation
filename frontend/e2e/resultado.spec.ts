import { test, expect, Page } from "@playwright/test";
import { prepararSessao, usarParametros, usarSessao } from "./apoio";

// Candidatos/avaliadores destacados pelos filtros e pela qualidade
const destacados = (page: Page, lista: "candidatos" | "avaliadores") =>
	page.getByTestId(lista).locator("button.bg-yellow-100");
const contagem = (page: Page) => page.getByTestId("contagem-filtro");
const valorDoItem = async (page: Page, codigo: string) =>
	parseInt((await page.getByTestId(`qualidade-${codigo}`).innerText()).trim());

async function abrirResultado(page: Page) {
	await page.goto("/resultado");
	await expect(page.getByText(/^\d+ alocados$/)).toBeVisible({ timeout: 30_000 });
}

test.describe("com os parâmetros padrão", () => {
	test.beforeEach(async ({ page, request }) => {
		await usarSessao(page, await prepararSessao(request));
		await abrirResultado(page);
	});

	test("resumo e relatório de qualidade somam os 98 candidatos", async ({ page }) => {
		await expect(page.getByTestId("resumo")).toContainText("98");
		let soma = 0;
		for (const item of await page.locator('[data-testid^="qualidade-opcao_"]').all()) {
			soma += parseInt((await item.innerText()).trim());
		}
		expect(soma + (await valorDoItem(page, "nao_alocados"))).toBe(98);
		// itens com valor 0 não são clicáveis
		await expect(page.getByTestId("qualidade-nao_alocados")).toBeDisabled();
	});

	test("clicar num item de qualidade destaca os candidatos dele", async ({ page }) => {
		const todasAsMesas = await page.getByTestId("mesa").count();
		const n = await valorDoItem(page, "opcao_2");
		test.skip(n === 0, "ninguém ficou na 2ª opção");

		const item = page.getByTestId("qualidade-opcao_2");
		await item.click();
		await expect(item).toHaveAttribute("aria-pressed", "true");
		await expect(destacados(page, "candidatos")).toHaveCount(n);
		for (const c of await destacados(page, "candidatos").all()) {
			await expect(c).toContainText("2ª opção");
		}
		await expect(contagem(page)).toContainText(`${n} candidato`);
		expect(await page.getByTestId("mesa").count()).toBeLessThan(todasAsMesas);

		// clicar de novo desliga o destaque
		await item.click();
		await expect(item).toHaveAttribute("aria-pressed", "false");
		await expect(page.getByTestId("mesa")).toHaveCount(todasAsMesas);
		await expect(contagem(page)).toHaveCount(0);
	});

	test("busca encontra candidato e avaliador", async ({ page }) => {
		const primeiraMesa = page.getByTestId("mesa").first();
		const candidato = (await primeiraMesa.getByTestId("candidatos").locator("li").first().innerText()).split("\n")[0].trim();
		const avaliadorBotao = primeiraMesa.getByTestId("avaliadores").locator("li").first();
		const avaliador = (await avaliadorBotao.innerText()).trim().split(/\s+(?=\S+$)/)[0]; // sem a sigla
		const mesasDoAvaliador = await page
			.getByTestId("mesa")
			.filter({ has: page.getByTestId("avaliadores").filter({ hasText: avaliador }) })
			.count();
		expect(mesasDoAvaliador).toBeGreaterThan(0);

		const busca = page.getByLabel("Buscar");
		await busca.fill(candidato);
		await expect(contagem(page)).toHaveText(/^1 candidato encontrado/);
		await expect(destacados(page, "candidatos")).toHaveCount(1);
		await expect(destacados(page, "candidatos")).toContainText(candidato);
		await expect(page.getByTestId("mesa")).toHaveCount(1);

		// acha o avaliador e todas as mesas em que ele está
		await busca.fill(avaliador);
		await expect(contagem(page)).toContainText("1 avaliador");
		await expect(page.getByTestId("mesa")).toHaveCount(mesasDoAvaliador);
		for (const a of await destacados(page, "avaliadores").all()) {
			await expect(a).toContainText(avaliador);
		}

		await page.getByRole("button", { name: "Limpar filtros" }).click();
		await expect(busca).toHaveValue("");
		await expect(destacados(page, "candidatos")).toHaveCount(0);
	});

	test("filtros de horário, curso e semestre", async ({ page }) => {
		// horário: só as mesas dele
		const horario = page.getByLabel("Horário");
		await horario.selectOption({ index: 1 });
		const escolhido = await horario.inputValue();
		const secoes = page.locator("section[aria-label^='Mesas de ']");
		await expect(secoes).toHaveCount(1);
		await expect(secoes.first()).toHaveAttribute("aria-label", `Mesas de ${escolhido}`);
		await expect(contagem(page)).toContainText("no horário");
		await horario.selectOption("");

		// curso: todo destacado é do curso (conferido no painel de detalhes)
		const curso = page.getByLabel("Curso");
		await curso.selectOption({ index: 1 });
		const cursoEscolhido = await curso.inputValue();
		const n = await destacados(page, "candidatos").count();
		expect(n).toBeGreaterThan(0);
		await expect(contagem(page)).toContainText(`${n} candidato`);
		await destacados(page, "candidatos").first().click();
		await expect(page.getByRole("dialog")).toContainText(cursoEscolhido);
		await page.keyboard.press("Escape");

		// curso + semestre: nunca mais que só o curso
		const semestre = page.getByLabel("Semestre");
		await semestre.selectOption({ index: 1 });
		const semestreEscolhido = await semestre.inputValue();
		const m = await destacados(page, "candidatos").count();
		expect(m).toBeLessThanOrEqual(n);
		if (m > 0) {
			await destacados(page, "candidatos").first().click();
			await expect(page.getByRole("dialog")).toContainText(cursoEscolhido);
			await expect(page.getByRole("dialog")).toContainText(`${semestreEscolhido}º`);
		}
	});

	test("detalhes do candidato e do avaliador", async ({ page }) => {
		const mesa = page.getByTestId("mesa").first();
		// textContent: o texto sem a capitalização do CSS
		const nomeMesa = ((await mesa.locator("span.font-bold").first().textContent()) ?? "").trim();

		await mesa.getByTestId("candidatos").locator("button").first().click();
		const dialogo = page.getByRole("dialog");
		await expect(dialogo).toContainText("Candidato");
		await expect(dialogo).toContainText(`Alocado em ${nomeMesa}`);
		await expect(dialogo).toContainText("← alocado");
		await expect(dialogo).toContainText("Horários que escolheu");
		await page.keyboard.press("Escape");
		await expect(dialogo).toHaveCount(0);

		await mesa.getByTestId("avaliadores").locator("button").first().click();
		await expect(dialogo).toContainText("Avaliador");
		await expect(dialogo).toContainText("Sigla");
		await expect(dialogo).toContainText(/Mesas \([1-9]\d*\)/);
		await expect(dialogo).toContainText(nomeMesa);
		await page.getByRole("button", { name: "Fechar detalhes" }).click();
		await expect(dialogo).toHaveCount(0);
	});

	test("exportação continua funcionando", async ({ page }) => {
		const [download] = await Promise.all([
			page.waitForEvent("download"),
			page.getByRole("button", { name: "Exportar Excel" }).click(),
		]);
		expect(download.suggestedFilename()).toBe("alocacao.xlsx");
	});
});

test("com pouca capacidade, destaca e detalha quem ficou sem mesa", async ({ page, request }) => {
	await usarSessao(page, await prepararSessao(request));
	await usarParametros(page, { mesas_por_horario: 1, avaliadores_por_mesa: 5, min_pessoas_por_mesa: 5, max_pessoas_por_mesa: 5 });
	await abrirResultado(page);

	const semMesa = await valorDoItem(page, "nao_alocados");
	expect(semMesa).toBeGreaterThan(0);
	const tabela = page.getByRole("region", { name: "Candidatos não alocados" });
	await expect(tabela.locator("tbody tr")).toHaveCount(semMesa);

	await page.getByTestId("qualidade-nao_alocados").click();
	await expect(page.getByTestId("mesa")).toHaveCount(0);
	await expect(tabela.locator("tbody tr.bg-yellow-50")).toHaveCount(semMesa);
	await expect(page.getByText("Nenhuma mesa com candidatos")).toHaveCount(0);

	await tabela.locator("tbody tr").first().getByRole("button").click();
	await expect(page.getByRole("dialog")).toContainText("Sem mesa: não coube");
});
