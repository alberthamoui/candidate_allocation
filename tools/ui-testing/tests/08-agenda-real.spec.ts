import { test, expect } from "@playwright/test";
import { completeImportWizard, reviewSchedules } from "./support/ui";

test("agenda real preserva avaliação, horário manual e Excel após recarregar", async ({
	page,
}) => {
	await completeImportWizard(page);
	await page.getByTestId("configure-allocation-button").click();
	await page.getByTestId("min-people-per-group-input").fill("1");
	await page.getByTestId("max-people-per-group-input").fill("8");
	await page.getByTestId("evaluators-per-group-input").fill("1");
	await reviewSchedules(page);
 await page.getByTestId("start-allocation-button").click();
	await expect(page.getByTestId("allocation-result-page")).toBeVisible({
		timeout: 30000,
	});
	if (await page.getByTestId("stop-search-button").isVisible()) {
		await page.getByTestId("stop-search-button").click();
	}
	await expect
		.poll(
			() =>
				page.evaluate(async () =>
					Boolean(
						(await (window as any).go.main.App.GetAllocationRunState(false))
							.running,
					),
				),
			{ timeout: 15000 },
		)
		.toBe(false);
	const initial = await page.evaluate(
		async () => await (window as any).go.main.App.GetAgenda(),
	);
	const person = initial.mesas[0].candidatos[0];
	await page.getByLabel(`Aprovar ${person.nome}`, { exact: true }).click();
	await expect(
		page.locator(`[data-candidate-id="${person.id}"]`),
	).toContainText("Passou");
	await page.getByLabel(`Realocar ${person.nome}`, { exact: true }).click();
	await page.getByLabel("Novo horário", { exact: true }).fill("sexta 23-24");
	await page.getByRole("button", { name: "Confirmar realocação" }).click();
	await expect(page.getByRole("dialog")).toHaveCount(0);
	await expect(
		page.getByRole("heading", { name: "Horário: sexta 23-24", exact: true }),
	).toBeVisible();
	await page.reload();
	await expect(page.getByTestId("allocation-result-page")).toBeVisible();
	await expect(
		page.locator(`[data-candidate-id="${person.id}"]`),
	).toContainText("Passou");
	await expect(
		page.getByRole("heading", { name: "Horário: sexta 23-24", exact: true }),
	).toBeVisible();
	const [download] = await Promise.all([
		page.waitForEvent("download"),
		page.getByRole("button", { name: "Exportar Excel", exact: true }).click(),
	]);
	expect(download.suggestedFilename()).toBe("agenda.xlsx");
	await page.goto("/");
	await page
		.getByRole("button", { name: "Retomar agenda salva", exact: true })
		.click();
	await expect(page.getByTestId("allocation-result-page")).toBeVisible();
	await expect(
		page.locator(`[data-candidate-id="${person.id}"]`),
	).toContainText("Passou");

	const final = await page.evaluate(
		async () => await (window as any).go.main.App.GetAgenda(),
	);
	expect(final.editado).toBe(true);
	expect(final.hardViolations.length).toBeGreaterThan(0);
});
