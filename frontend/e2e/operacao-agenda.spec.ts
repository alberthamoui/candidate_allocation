import { test, expect } from "@playwright/test";
import { createServer } from "node:http";
import { fileURLToPath } from "node:url";
import { prepararSessao, usarSessao } from "./apoio";

const semCPF = fileURLToPath(
  new URL("./planilhas/sem_cpf_sem_opcoes.xlsx", import.meta.url),
);

test("entrada oferece configurar ou adiar os canais e insere variáveis", async ({
  page,
}) => {
  await page.goto("/");
  await expect(
    page.getByRole("button", { name: "Agora não", exact: true }),
  ).toHaveAttribute("aria-pressed", "true");
  await page
    .getByRole("button", { name: "Configurar avisos", exact: true })
    .click();
  const mensagem = page.getByRole("textbox", { name: "Mensagem", exact: true });
  await mensagem.fill("Olá ");
  await mensagem.click();
  await mensagem.press("End");
  await page.getByRole("button", { name: "{{nome}}", exact: true }).click();
  await expect(mensagem).toHaveValue("Olá {{nome}}");
  await page.getByRole("button", { name: "Agora não", exact: true }).click();
  await expect(mensagem).toHaveCount(0);
});

test("realocação e avaliação junto à pessoa persistem ao recarregar sem nova alocação", async ({
  page,
  request,
}) => {
  const id = await prepararSessao(request);
  await usarSessao(page, id);
  await page.goto("/resultado");
  await expect(page.getByText("98 alocados", { exact: true })).toBeVisible();
  const avaliacao = page
    .getByRole("combobox", { name: /^Avaliação de / })
    .first();
  const nome = (await avaliacao.getAttribute("aria-label"))!.replace(
    "Avaliação de ",
    "",
  );
  await avaliacao.selectOption("aprovado");
  await expect(
    page.getByRole("status").filter({ hasText: `${nome}: passou.` }),
  ).toBeVisible();
  await page
    .getByRole("button", { name: `Realocar ${nome}`, exact: true })
    .click();
  const dialog = page.getByRole("dialog");
  await expect(dialog.getByText(/Melhor opção:/)).toBeVisible();
  await dialog
    .getByLabel("Novo horário", { exact: true })
    .fill("29/10/2026 14:00-16:00");
  await dialog.getByRole("button", { name: "Confirmar realocação" }).click();
  await expect(dialog).toHaveCount(0);
  let recalculou = false;
  page.on("request", (req) => {
    if (req.url().includes("/api/alocar?")) recalculou = true;
  });
  await page.reload();
  await expect(page.getByText("98 alocados", { exact: true })).toBeVisible();
  expect(recalculou).toBeFalsy();
  await expect(
    page.getByRole("combobox", { name: `Avaliação de ${nome}`, exact: true }),
  ).toHaveValue("aprovado");
  const resultado = await (
    await request.get("/api/resultado", { headers: { "X-Session-Id": id } })
  ).json();
  expect(
    resultado.mesas.find((m: any) =>
      m.candidatos.some((c: any) => c.nome === nome),
    ).dia_nome,
  ).toBe("29/10/2026 14:00-16:00");
});

test("mesa e dia inteiro podem ser realocados pela interface", async ({
  page,
  request,
}) => {
  const id = await prepararSessao(request);
  await usarSessao(page, id);
  await page.goto("/resultado");
  await expect(page.getByText("98 alocados", { exact: true })).toBeVisible();
  const mesa = page.getByTestId("mesa").first();
  await mesa
    .getByRole("button", { name: "Realocar mesa", exact: true })
    .click();
  let dialog = page.getByRole("dialog");
  await dialog
    .getByLabel("Novo horário", { exact: true })
    .fill("28/10/2026 10:00-12:00");
  await dialog.getByRole("button", { name: "Confirmar realocação" }).click();
  await expect(dialog).toHaveCount(0);
  const horario = page.getByRole("region", {
    name: "Mesas de 28/10/2026 10:00-12:00",
    exact: true,
  });
  await horario
    .getByRole("button", { name: "Realocar horário ou dia" })
    .click();
  dialog = page.getByRole("dialog");
  await dialog
    .getByLabel("Origem: horário completo ou somente o dia")
    .fill("28/10/2026");
  await dialog.getByLabel("Novo dia ou horário").fill("30/10/2026");
  await dialog.getByRole("button", { name: "Confirmar realocação" }).click();
  await expect(
    page.getByRole("region", {
      name: "Mesas de 30/10/2026 10:00-12:00",
      exact: true,
    }),
  ).toBeVisible();
});

test("planilha sem CPF e sem opções usa horários do administrador", async ({
  page,
}) => {
  await page.goto("/");
  await page
    .getByRole("checkbox", {
      name: "Usar preferências de horário dos candidatos",
      exact: true,
    })
    .uncheck();
  await page.locator("input[type=file]").setInputFiles(semCPF);
  await page.getByRole("button", { name: "Iniciar →" }).click();
  await expect(page.locator('[data-variavel="opcao 1"]')).toHaveCount(0);
  await page.getByRole("button", { name: "Continuar para a revisão" }).click();
  await page
    .getByRole("button", { name: "Salvar Candidatos", exact: true })
    .click();
  await page.getByRole("button", { name: "Processar Avaliadores" }).click();
  await page
    .getByRole("button", { name: "Salvar avaliadores", exact: true })
    .click();
  await page.getByRole("button", { name: "Processar Restrições" }).click();
  await page
    .getByRole("button", { name: "Salvar restrições", exact: true })
    .click();
  await page
    .getByLabel("Horários disponíveis (um por linha)")
    .fill("28/10/2026 10:00-12:00");
  await page.getByLabel(/^Avaliadores por mesa/).fill("1");
  await page
    .getByRole("button", { name: "Rodar alocação", exact: true })
    .click();
  await expect(page.getByText("6 alocados", { exact: true })).toBeVisible();
  await expect(
    page.getByText("Sem preferência de horário", { exact: false }).first(),
  ).toBeVisible();
});

test("RA não vira telefone e o acompanhamento consulta só o lote ativo", async ({
  page,
  request,
}) => {
  const recebidas: any[] = [];
  const servidor = createServer((req, res) => {
    let body = "";
    req.on("data", (c) => (body += c));
    req.on("end", () => {
      recebidas.push(JSON.parse(body));
      res.writeHead(200, { "Content-Type": "application/json" });
      res.end("{}");
    });
  });
  await new Promise<void>((resolve) =>
    servidor.listen(0, "127.0.0.1", resolve),
  );
  const endereco = servidor.address() as { port: number };
  try {
    const id = await prepararSessao(request);
    const headers = { "X-Session-Id": id };
    // A planilha oficial tem RA, sem telefone: o envio deve apontar o contato ausente.
    const cfg = {
      ativo: true,
      mensagem: "Olá {{nome}}, dia {{data}} às {{horario}}.",
      email: { ativo: false },
      whatsapp: {
        ativo: true,
        provedor: "webhook",
        url: `http://127.0.0.1:${endereco.port}`,
        token: "",
      },
    };
    await request.post("/api/avisos", { headers, data: cfg });
    await usarSessao(page, id);
    await page.goto("/resultado");
    await expect(page.getByText("98 alocados", { exact: true })).toBeVisible();
    let historicos = 0;
    page.on("request", (r) => {
      if (r.url().includes("/api/avisos/historico")) historicos++;
    });
    await page
      .getByTestId("mesa")
      .first()
      .getByRole("button", { name: "Avisar", exact: true })
      .first()
      .click();
    await page
      .getByRole("dialog")
      .getByRole("button", { name: "Autorizar envio", exact: true })
      .click();
    await expect(
      page.getByRole("status").filter({ hasText: /Envio:.*falhas/ }),
    ).toBeVisible();
    expect(historicos).toBe(0);
    expect(recebidas).toHaveLength(0); // RA jamais vira telefone
    await page
      .getByRole("button", { name: "Histórico de envios", exact: true })
      .click();
    await page
      .getByRole("button", { name: "Ver mensagens", exact: true })
      .first()
      .click();
    await expect(page.getByRole("dialog")).toContainText(
      "informe um telefone com DDD",
    );
  } finally {
    await new Promise<void>((resolve, reject) =>
      servidor.close((e) => (e ? reject(e) : resolve())),
    );
  }
});

test("aviso automático usa o telefone próprio e as variáveis da agenda", async ({
  page,
  request,
}) => {
  const recebidas: any[] = [];
  const servidor = createServer((req, res) => {
    let body = "";
    req.on("data", (c) => (body += c));
    req.on("end", () => {
      recebidas.push(JSON.parse(body));
      res.writeHead(200, { "Content-Type": "application/json" });
      res.end("{}");
    });
  });
  await new Promise<void>((resolve) =>
    servidor.listen(0, "127.0.0.1", resolve),
  );
  const endereco = servidor.address() as { port: number };
  try {
    const id = await prepararSessao(request, semCPF);
    const headers = { "X-Session-Id": id };
    const config = await request.post("/api/avisos", {
      headers,
      data: {
        ativo: true,
        enviar_ao_alocar: true,
        mensagem: "Olá {{nome}}, sua entrevista é em {{data}} às {{horario}}.",
        email: { ativo: false },
        whatsapp: {
          ativo: true,
          provedor: "webhook",
          url: `http://127.0.0.1:${endereco.port}`,
          token: "",
        },
      },
    });
    expect(config.ok()).toBeTruthy();
    const q = new URLSearchParams({
      sessionId: id,
      sem_preferencia: "true",
      horarios_livres: JSON.stringify(["28/10/2026 10:00-12:00"]),
      avaliadores_por_mesa: "1",
    });
    const r = await request.get("/api/alocar?" + q);
    expect(await r.text()).toContain('"done":true');
    await expect.poll(() => recebidas.length).toBe(6);
    expect(recebidas[0].to).toBe("5511999990000");
    expect(
      recebidas.every((m) => m.message.includes("28/10/2026 às 10:00-12:00")),
    ).toBeTruthy();
    await usarSessao(page, id);
    await page.goto("/resultado");
    await expect(page.getByText("6 alocados", { exact: true })).toBeVisible();
    await page
      .getByRole("button", { name: "Histórico de envios", exact: true })
      .click();
    await expect(
      page.getByText("6 aceitos · 0 falhas · 0 incertos · 0 pendentes", {
        exact: true,
      }),
    ).toBeVisible();
  } finally {
    await new Promise<void>((resolve, reject) =>
      servidor.close((e) => (e ? reject(e) : resolve())),
    );
  }
});

test("cancelar a configuração preserva os canais salvos", async ({
  page,
  request,
}) => {
  const id = await prepararSessao(request);
  await usarSessao(page, id);
  await page.goto("/resultado");
  await expect(page.getByText("98 alocados", { exact: true })).toBeVisible();
  await page
    .getByRole("button", { name: "Configurar canais", exact: true })
    .click();
  const dialog = page.getByRole("dialog");
  await dialog
    .getByRole("checkbox", { name: "Avisos ligados", exact: true })
    .check();
  await dialog
    .getByRole("checkbox", { name: "Enviar por e-mail", exact: true })
    .check();
  await dialog.getByRole("button", { name: "Cancelar", exact: true }).click();
  await expect(dialog).toHaveCount(0);
  await expect(
    page.getByText("Avisos desligados. Configure um canal para enviar.", {
      exact: true,
    }),
  ).toBeVisible();
  const cfg = await (
    await request.get("/api/avisos", { headers: { "X-Session-Id": id } })
  ).json();
  expect(cfg.ativo).toBeFalsy();
});
