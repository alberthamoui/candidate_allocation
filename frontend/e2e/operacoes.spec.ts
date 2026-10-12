import { test, expect } from "@playwright/test";
import { prepararSessao, usarSessao } from "./apoio";

test("na entrada, preferência e avisos são opcionais", async ({ page }) => {
	await page.goto("/");

	await expect(page.getByLabel("Número de opções de horário por candidato")).toBeVisible();
	await page.getByRole("checkbox", { name: /preferência de horário/ }).uncheck();
	await expect(page.getByLabel("Número de opções de horário por candidato")).toHaveCount(0);
	await page.getByRole("checkbox", { name: /preferência de horário/ }).check();
	await expect(page.getByLabel("Número de opções de horário por candidato")).toBeVisible();

	await page.getByRole("checkbox", { name: /Configurar avisos/ }).check();
	const mensagem = page.getByRole("textbox", { name: "Mensagem" });
	await expect(mensagem).toHaveValue(/\{\{nome\}\}/);
	await page.getByRole("button", { name: "{{curso}}" }).click();
	await expect(mensagem).toHaveValue(/\{\{curso\}\}$/);

	await page.getByRole("checkbox", { name: "Enviar por e-mail (SMTP)" }).check();
	await expect(page.getByPlaceholder("servidor SMTP")).toBeVisible();
	await page.getByRole("checkbox", { name: "Enviar por WhatsApp" }).check();
	await expect(page.getByPlaceholder(/URL do webhook/)).toBeVisible();

	await page.getByRole("button", { name: "Iniciar →" }).click();
	await expect(page.getByText("Por favor, selecione um arquivo .xlsx.")).toBeVisible();
});

test("sem preferência dá para realocar, avaliar e exportar", async ({ page, request }) => {
	test.setTimeout(150_000);
	const id = await prepararSessao(request, undefined, { usarPreferencia: false });
	await usarSessao(page, id);
	await page.addInitScript(() => sessionStorage.setItem("allocation_usar_preferencia", "false"));
	await page.goto("/parametros");

	const horarios = page.getByLabel("Horários da entrevista");
	await expect(horarios).toBeVisible();
	await horarios.fill("quarta 14-16\nquinta 10-12");
	await expect(page.getByRole("button", { name: "Rodar alocação" })).toBeEnabled({ timeout: 15_000 });
	await page.getByRole("button", { name: "Rodar alocação" }).click();

	await expect(page).toHaveURL(/\/resultado$/);
	await expect(page.locator("body")).toContainText(/\d+ alocados/, { timeout: 60_000 });
	await expect(page.getByRole("button", { name: "Exportar Excel" })).toBeVisible();

	await page.getByRole("button", { name: "Avaliação" }).click();
	await page.getByRole("button", { name: "Passou" }).first().click();
	await expect(page.getByText(/: passou\./)).toBeVisible();

	await page.getByRole("button", { name: "Uma pessoa" }).click();
	await page.getByRole("checkbox", { name: "Avisar esta pessoa" }).uncheck();
	await page.getByLabel("Novo horário").fill("sexta 8-10");
	await page.getByRole("button", { name: "Trocar horário" }).click();
	await expect(page.getByText(/foi para sexta 8-10\./)).toBeVisible();

	await page.getByRole("button", { name: "Imprevisto" }).click();
	await page.getByRole("checkbox", { name: /Avisar todo mundo/ }).uncheck();
	await expect(page.getByLabel("Dia atual")).not.toHaveValue("");
	await page.getByLabel("Novo dia").fill("sexta 16-18");
	page.once("dialog", (dialog) => dialog.accept());
	await page.getByRole("button", { name: "Mover o dia" }).click();
	await expect(page.getByText(/passaram para sexta 16-18\./)).toBeVisible();

	const [download] = await Promise.all([
		page.waitForEvent("download"),
		page.getByRole("button", { name: "Exportar Excel" }).click(),
	]);
	expect(download.suggestedFilename()).toBe("alocacao.xlsx");
});
