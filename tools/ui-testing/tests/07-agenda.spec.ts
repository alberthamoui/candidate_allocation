import { test, expect, type Page } from "@playwright/test";

async function agenda(page: Page) {
	await page.goto("/");
	await page.waitForFunction(() => Boolean((window as any).go?.main?.App));
	await page.evaluate(() => {
		const w = window as any,
			app = w.go.main.App;
		const candidate = (id: number, nome: string) => ({
			id,
			nome,
			curso: "ADM",
			semestre: 1,
			opcoes: ["quarta 14-16"],
			naoPosso: [],
			prefiroNao: [],
			extras: {},
			avaliacao: "",
		});
		w.agendaFixture = {
			status: "Sucesso!",
			solverStatus: "optimal",
			mesas: [
				{
					id: 1,
					horario: "quarta 14-16",
					descricao: "Mesa 1",
					candidatos: [candidate(1, "Ana"), candidate(2, "Bia")],
					avaliadores: [],
				},
				{
					id: 2,
					horario: "sexta 10-12",
					descricao: "Mesa 2",
					candidatos: [candidate(3, "Caio")],
					avaliadores: [],
				},
			],
			naoAlocados: [],
			hardViolations: [],
			quality: { characteristics: [] },
			score: { totalPenalty: 0, components: [] },
			metrics: {},
		};
		w.historyCalls = 0;
		w.detailCalls = 0;
		w.sendCalls = 0;
		app.GetNoticeStates = async () => ({});
		app.GetAllocationRunState = async () => ({
			running: false,
			hasResult: true,
			result: w.agendaFixture,
			progress: { percent: 100, totalBranches: "1", branchesResolved: "1" },
			error: "",
		});
		app.EvaluateCandidates = async (ids: number[], value: string) => {
			w.agendaFixture.mesas.forEach((m: any) =>
				m.candidatos.forEach((c: any) => {
					if (ids.includes(c.id)) c.avaliacao = value;
				}),
			);
			return structuredClone(w.agendaFixture);
		};
		app.GetMoveSuggestions = async () => [
			{
				groupId: 2,
				description: "sexta 10-12 · Mesa 2",
				schedule: "sexta 10-12",
				vacancies: 3,
				penalty: 0,
				alerts: [],
			},
		];
		app.MoveAgenda = async (p: any) => {
			w.lastMove = p;
			const c = w.agendaFixture.mesas[0].candidatos.shift();
			w.agendaFixture.mesas[1].candidatos.push(c);
			return { result: structuredClone(w.agendaFixture), batchId: 0 };
		};
		app.PreviewAgendaNotices = async () => [
			{
				name: "Ana",
				channel: "email",
				destination: "ana@example.test",
				message: "Olá Ana, sexta 10-12",
				error: "",
			},
		];
		app.SendAgendaNotices = async () => {
			w.sendCalls++;
			return 9;
		};
		app.GetChannelConfig = async () => ({
			ativo: true,
			mensagem: "Olá {{nome}}",
			email: {
				ativo: true,
				provedor: "gmail",
				host: "smtp.gmail.com",
				porta: 587,
				usuario: "org@example.test",
				remetente: "org@example.test",
				senha: "",
				senha_definida: true,
			},
			whatsapp: {
				ativo: false,
				provedor: "meta",
				phone_id: "",
				token: "",
				template: "",
				idioma: "pt_BR",
				variaveis_template: ["nome"],
				url: "",
			},
		});
		app.GetGoogleConnectionStatus = async () => ({
			available: false,
			status: "idle",
			message: "",
		});
		app.SaveChannelConfig = async (c: any) => {
			w.savedChannels = c;
		};
		app.TestChannels = async () => ({ email: "Conexão verificada" });
		app.GetNoticeHistory = async (before: number) => {
			w.historyCalls++;
			return {
				items: [
					{
						id: 7,
						created: new Date().toISOString(),
						accepted: 1,
						failed: 1,
						pending: 0,
						unknown: 0,
						repeatedBy: 0,
					},
				],
				next: before ? 0 : 3,
			};
		};
		app.GetNoticeDetails = async () => {
			w.detailCalls++;
			return {
				items: [
					{
						id: 1,
						name: "Ana",
						channel: "email",
						status: "failed",
						error: "Conexão temporariamente indisponível",
					},
				],
				next: 0,
			};
		};
		w.runtime.EventsEmit("allocation:complete", w.agendaFixture);
		history.pushState({}, "", "/allocation-result");
		dispatchEvent(new PopStateEvent("popstate"));
	});
	await expect(page.getByTestId("allocation-result-page")).toBeVisible();
}

test("ações da pessoa avaliam, filtram, realocam e exigem confirmação para avisar", async ({
	page,
}) => {
	await agenda(page);
	await page.getByLabel("Aprovar Ana", { exact: true }).click();
	await expect(page.locator('[data-candidate-id="1"]')).toContainText("Passou");
	await page.getByRole("button", { name: "Sem avaliação: 2" }).click();
	await expect(page.locator('[data-candidate-id="1"]')).toBeHidden();
	await expect(page.locator('[data-candidate-id="2"]')).toBeVisible();
	await page.getByRole("button", { name: "Mostrar todos" }).click();
	await page.getByLabel("Realocar Ana", { exact: true }).click();
	await expect(page.getByRole("dialog")).toContainText("Melhores destinos");
	await page.getByRole("button", { name: /1º.*sexta/ }).click();
	await page.getByRole("button", { name: "Confirmar realocação" }).click();
	await expect(
		page.locator('[data-group-id="2"] [data-candidate-id="1"]'),
	).toBeVisible();
	expect(
		await page.evaluate(() => (window as any).lastMove.destinationId),
	).toBe(2);
	await page.getByLabel("Avisar Ana", { exact: true }).click();
	await expect(
		page.getByRole("dialog", { name: "Confirmar envio" }),
	).toContainText("ana@example.test");
	expect(await page.evaluate(() => (window as any).sendCalls)).toBe(0);
	await page.getByLabel("Fechar ação").click();
	expect(await page.evaluate(() => (window as any).sendCalls)).toBe(0);
	await page.getByLabel("Avisar Ana", { exact: true }).click();
	await page
		.getByRole("button", { name: "Confirmar envio", exact: true })
		.click();
	expect(await page.evaluate(() => (window as any).sendCalls)).toBe(1);
});

test("histórico não faz polling nem carrega destinatários ao abrir o painel", async ({
	page,
}) => {
	await agenda(page);
	await page.getByRole("button", { name: "Canais e mensagens" }).click();
	expect(await page.evaluate(() => (window as any).historyCalls)).toBe(0);
	await page.getByRole("button", { name: "Histórico", exact: true }).click();
	await expect
		.poll(() => page.evaluate(() => (window as any).historyCalls))
		.toBe(1);
	expect(await page.evaluate(() => (window as any).detailCalls)).toBe(0);
	await page.waitForTimeout(3200);
	expect(await page.evaluate(() => (window as any).historyCalls)).toBe(1);
	await page.getByRole("button", { name: "Ver destinatários" }).click();
	await expect(page.getByRole("dialog")).toContainText(
		"Conexão temporariamente indisponível",
	);
	expect(await page.evaluate(() => (window as any).detailCalls)).toBe(1);
	await page
		.getByRole("button", { name: "Carregar envios anteriores" })
		.click();
	await expect
		.poll(() => page.evaluate(() => (window as any).historyCalls))
		.toBe(2);
	await page.getByRole("button", { name: "1. Conectar canais" }).click();
	await page.evaluate(() =>
		(window as any).runtime.EventsEmit("notices:changed"),
	);
	await page.waitForTimeout(200);
	expect(await page.evaluate(() => (window as any).historyCalls)).toBe(2);
});

test("canais oferecem configuração guiada e modelos de mensagem", async ({
	page,
}) => {
	await agenda(page);
	await page.getByRole("button", { name: "Canais e mensagens" }).click();
	await expect(page.getByLabel("Seu e-mail de envio")).toHaveValue(
		"org@example.test",
	);
	await expect(
		page.getByLabel("Senha de aplicativo", { exact: true }),
	).toHaveAttribute("type", "password");
	await page.getByRole("button", { name: "Salvar e testar conexão" }).click();
	await expect(page.getByRole("status")).toContainText("Conexão verificada");
	await page.getByRole("button", { name: "2. Mensagem" }).click();
	await page
		.getByRole("button", { name: "Mudança de horário", exact: true })
		.click();
	await expect(page.getByLabel("Mensagem", { exact: true })).toHaveValue(
		/remarcada/,
	);
	await expect(
		page.getByText("Maria foi remarcada", { exact: false }),
	).toHaveCount(0);
	await page.getByRole("button", { name: "Salvar configuração" }).click();
	expect(
		await page.evaluate(() => (window as any).savedChannels.mensagem),
	).toContain("{{nome}}");
});

test("ações de mesa e realocação permanecem acessíveis no celular", async ({
	page,
}) => {
	await page.setViewportSize({ width: 390, height: 844 });
	await agenda(page);
	await page
		.locator('[data-group-id="1"]')
		.getByRole("button", { name: "Realocar mesa" })
		.click();
	await expect(page.getByRole("dialog")).toBeVisible();
	await page.keyboard.press("Escape");
	await expect(page.getByRole("dialog")).toHaveCount(0);
	expect(
		await page.evaluate(
			() => document.documentElement.scrollWidth > innerWidth,
		),
	).toBe(false);
});
