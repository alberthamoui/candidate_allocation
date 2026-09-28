import { test, expect } from "@playwright/test";

test("sem arquivo, pede para selecionar um", async ({ page }) => {
	await page.goto("/");
	await page.getByRole("button", { name: "Iniciar →" }).click();
	await expect(page.getByText("Por favor, selecione um arquivo .xlsx.")).toBeVisible();
	await expect(page).toHaveURL(/\/$/);
});

test("arquivo que não é Excel mostra erro", async ({ page }) => {
	await page.goto("/");
	await page.locator('input[type="file"]').setInputFiles({
		name: "nao-e-excel.xlsx",
		mimeType: "application/octet-stream",
		buffer: Buffer.from("isto não é uma planilha"),
	});
	await page.getByRole("button", { name: "Iniciar →" }).click();
	await expect(page.getByText(/Erro ao processar o arquivo/)).toBeVisible();
	await expect(page).toHaveURL(/\/$/);
});

test("planilha de exemplo pode ser baixada", async ({ page }) => {
	await page.goto("/");
	const [download] = await Promise.all([
		page.waitForEvent("download"),
		page.getByRole("link", { name: "Baixar planilha de exemplo" }).click(),
	]);
	expect(download.suggestedFilename()).toMatch(/\.xlsx$/);
});

test("selo mostra o commit que o servidor está rodando", async ({ page, request }) => {
	const versao = await (await request.get("/api/versao")).json();
	expect(versao.commit).toMatch(/^[0-9a-f]{7}$/);

	await page.goto("/");
	const selo = page.locator("div.fixed.bottom-3.right-3");
	await expect(selo).toBeVisible();
	await expect(selo).toContainText(versao.commit);
	if (versao.branch) await expect(selo).toContainText(versao.branch);
});
