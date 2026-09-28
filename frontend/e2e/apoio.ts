import { APIRequestContext, Page, expect } from "@playwright/test";
import { readFileSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

// Planilhas versionadas em Excels/
export const PLANILHA_OFICIAL = fileURLToPath(new URL("../../Excels/teste_oficial.xlsx", import.meta.url));
export const PLANILHA_EXEMPLO = fileURLToPath(new URL("../../Excels/base_exemplo.xlsx", import.meta.url));

/**
 * Passa uma planilha por upload → build → save das 3 abas direto pela API,
 * aceitando as sugestões de mapeamento, e devolve o id da sessão. Serve para
 * testes que começam depois da importação.
 */
export async function prepararSessao(request: APIRequestContext, planilha = PLANILHA_OFICIAL): Promise<string> {
	const ok = async (resp: Awaited<ReturnType<APIRequestContext["post"]>>) => {
		expect(resp.ok(), `${resp.url()} → ${resp.status()} ${await resp.text()}`).toBeTruthy();
		return resp.json();
	};

	const up = await ok(
		await request.post("/api/upload", {
			multipart: {
				file: { name: path.basename(planilha), mimeType: "application/octet-stream", buffer: readFileSync(planilha) },
				nOpcoes: "5",
				emailDomain: "@al.insper.edu.br",
			},
		})
	);
	const headers = { "X-Session-Id": up.sessionId };
	const post = (url: string, data?: unknown) => request.post(url, { headers, data });

	const usuarios = await ok(await post("/api/build-usuarios", up.mapping));
	await ok(await post("/api/save-usuarios", Object.values<any>(usuarios.usuarios).map((u) => u.usuario)));

	const mAv = await ok(await post("/api/suggest-avaliador"));
	await ok(await post("/api/save-avaliadores", await ok(await post("/api/build-avaliadores", mAv))));

	const mRe = await ok(await post("/api/suggest-restricao"));
	await ok(await post("/api/save-restricoes", await ok(await post("/api/build-restricoes", mRe))));

	return up.sessionId;
}

/** Faz a aba usar a sessão dada (como se o upload tivesse sido feito nela). */
export async function usarSessao(page: Page, sessionId: string) {
	await page.addInitScript((id) => sessionStorage.setItem("allocation_session_id", id), sessionId);
}

/** Linha da tabela de mapeamento de uma variável (ex.: "nome", "opcao 1"). */
export function linhaMapeamento(page: Page, variavel: string) {
	return page.getByRole("row").filter({ has: page.getByRole("cell", { name: variavel, exact: true }) });
}

/** Número de um dos cartões da prévia de capacidade (ex.: "mesas por horário"). */
export function cartaoPrevia(page: Page, rotulo: string) {
	return page.locator("div.border.rounded-xl").filter({ hasText: rotulo });
}
