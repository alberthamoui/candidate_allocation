import { test, expect } from "@playwright/test";
import { prepararSessao, usarSessao } from "./apoio";

test("na entrada, preferência e avisos são opcionais", async ({ page }) => {
  await page.goto("/");

  await expect(
    page.getByLabel("Número de opções de horário por candidato"),
  ).toBeVisible();
  await page
    .getByRole("checkbox", { name: /preferência de horário/ })
    .uncheck();
  await expect(
    page.getByLabel("Número de opções de horário por candidato"),
  ).toHaveCount(0);
  await page.getByRole("checkbox", { name: /preferência de horário/ }).check();
  await expect(
    page.getByLabel("Número de opções de horário por candidato"),
  ).toBeVisible();

  await page.getByRole("checkbox", { name: /Configurar avisos/ }).check();
  const mensagem = page.getByRole("textbox", { name: "Mensagem" });
  await expect(mensagem).toHaveValue(/\{\{nome\}\}/);
  await page.getByRole("button", { name: "{{curso}}" }).click();
  await expect(mensagem).toHaveValue(/\{\{curso\}\}$/);

  await page
    .getByRole("checkbox", { name: "Enviar por e-mail (SMTP)" })
    .check();
  await page.getByLabel("Serviço de e-mail").selectOption("smtp");
  await expect(page.getByPlaceholder("servidor SMTP")).toBeVisible();
  await page.getByRole("checkbox", { name: "Enviar por WhatsApp" }).check();
  await expect(page.getByLabel("ID do número na Meta")).toBeVisible();
  await page.getByLabel("Provedor WhatsApp").selectOption("webhook");
  await expect(page.getByPlaceholder(/URL do webhook/)).toBeVisible();

  await page.getByRole("button", { name: "Iniciar →" }).click();
  await expect(
    page.getByText("Por favor, selecione um arquivo .xlsx."),
  ).toBeVisible();
});

test("sem preferência dá para realocar, avaliar e exportar", async ({
  page,
  request,
}) => {
  test.setTimeout(150_000);
  const id = await prepararSessao(request, undefined, {
    usarPreferencia: false,
  });
  await usarSessao(page, id);
  await page.addInitScript(() =>
    sessionStorage.setItem("allocation_usar_preferencia", "false"),
  );
  await page.goto("/parametros");

  const horarios = page.getByLabel("Horários da entrevista");
  await expect(horarios).toBeVisible();
  await horarios.fill("quarta 14-16\nquinta 10-12");
  await expect(
    page.getByRole("button", { name: "Rodar alocação" }),
  ).toBeEnabled({ timeout: 15_000 });
  await page.getByRole("button", { name: "Rodar alocação" }).click();

  await expect(page).toHaveURL(/\/resultado$/);
  await expect(page.locator("body")).toContainText(/\d+ alocados/, {
    timeout: 60_000,
  });
  await expect(
    page.getByRole("button", { name: "Exportar Excel" }),
  ).toBeVisible();

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

test("sugestões orientam sem bloquear um horário manual e persistem ao atualizar", async ({
  page,
  request,
}) => {
  const id = await prepararSessao(request);
  await usarSessao(page, id);
  await page.goto("/resultado");
  await expect(page.getByText(/^\d+ alocados$/)).toBeVisible({
    timeout: 60_000,
  });
  await page.getByRole("button", { name: "Uma pessoa", exact: true }).click();
  await expect(
    page.getByRole("heading", { name: "Melhores destinos sugeridos" }),
  ).toBeVisible();
  await expect(
    page.getByLabel("Mesa de destino").locator("option"),
  ).not.toHaveCount(1);
  await page.getByLabel("Novo horário").fill("sábado 20-22");
  await expect(
    page.getByText(/cria uma mesa manual sem avaliadores/),
  ).toBeVisible();
  await page.getByRole("button", { name: "Trocar horário" }).click();
  await expect(page.getByText(/foi para sábado 20-22/)).toBeVisible();
  await expect(page.getByText(/alerta\(s\) da agenda manual/)).toBeVisible();
  await page.reload();
  await expect(
    page.getByRole("heading", { name: "Resultado da Alocação" }),
  ).toBeVisible();
  await expect(page.locator("body")).toContainText(/sábado 20-22/i);
  const r = await (
    await request.get("/api/resultado", { headers: { "X-Session-Id": id } })
  ).json();
  expect(r.editado).toBe(true);
  expect(r.mesas.some((m: any) => m.dia_nome === "sábado 20-22")).toBe(true);
});

test("configuração completa no resultado, prévia e segredos protegidos", async ({
  page,
  request,
}) => {
  const id = await prepararSessao(request);
  await usarSessao(page, id);
  await page.goto("/resultado");
  await expect(page.getByText(/^\d+ alocados$/)).toBeVisible({
    timeout: 60_000,
  });
  await page.getByRole("button", { name: "Avisos", exact: true }).click();
  await page
    .getByRole("checkbox", { name: "Enviar por e-mail (SMTP)" })
    .check();
  await page.getByLabel("Serviço de e-mail").selectOption("smtp");
  await expect(page.getByLabel("Porta SMTP")).toBeVisible();
  await page.getByLabel("Usuário SMTP").fill("org@exemplo.test");
  await page.getByLabel("Servidor SMTP").fill("smtp.exemplo.test");
  await page.getByLabel("Senha SMTP").fill("segredo-ficticio");
  await page.getByRole("button", { name: "Salvar mensagem" }).click();
  await expect(page.getByText("Configuração de avisos salva.")).toBeVisible();
  const cfg = await (
    await request.get("/api/avisos", { headers: { "X-Session-Id": id } })
  ).json();
  expect(cfg.email.senha).toBe("");
  expect(cfg.email.senha_definida).toBe(true);
  await page.getByRole("button", { name: "Ver prévia da mensagem" }).click();
  await expect(
    page.getByText("Prévia — nenhuma mensagem enviada"),
  ).toBeVisible();
  await page.getByRole("checkbox", { name: "Enviar por WhatsApp" }).check();
  await page.getByLabel("Provedor WhatsApp").selectOption("meta");
  await expect(page.getByLabel("ID do número na Meta")).toBeVisible();
  await expect(page.getByLabel("Modelo aprovado na Meta")).toBeVisible();
});
