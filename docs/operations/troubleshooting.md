---
id: troubleshooting
title: Troubleshooting
summary: Diagnóstico de Wails, bindings, Playwright, banco e rotas transitórias.
status: active
sources:
  - tools/ui-testing/runner.sh
  - tools/ui-testing/agent_inspect.js
  - frontend/src/wailsReady.ts
  - app.go
---

# Troubleshooting

## Playwright: connection refused

Confirme que `wails dev` está ativo e que `curl http://localhost:34115` responde. A porta 5173 é do Vite interno; testes devem usar a base configurável 34115.

## Playwright: módulo ausente

Se aparecer `Cannot find module` dentro de `node_modules/playwright`, rode o runner atualizado ou:

```bash
cd tools/ui-testing
npm ci
```

O runner detecta `@playwright/test` incompleto. Para inspecionar browsers:

```bash
npx playwright install --list
```

## Bindings ainda indisponíveis

Rotas diretas podem montar antes do runtime. Componentes que chamam Wails durante o mount devem usar `waitForWailsBindings`. Se o erro persistir, confirme que a página está dentro do Wails ou que o teste instalou mocks em `window.go.main.App`.

## Relatório de UI parece antigo

`test-results/report.json` descreve a última execução concluída. Compare os paths/linhas do relatório com os specs atuais e execute `./runner.sh` novamente antes de diagnosticar seletores.

## Resultado vazio ao recarregar

Configuração e resultado usam `location.state`. Recarregar `/allocation-loading` ou `/allocation-result` perde esse state. Refaça o caminho a partir de `/allocation-config`.

## Banco inesperadamente vazio

O desktop executa `ClearDatabase` em todo startup. Verifique também o diretório corrente, pois `./insper.db` é relativo ao processo.

## Restrições não aparecem no solver

Atualmente a UI salva restrições antes dos avaliadores, mas a persistência resolve siglas consultando `avaliador`. Erros individuais são impressos e não necessariamente propagados. Veja [limitações conhecidas](../reference/known-limitations.md).

## Alocação inviável

Leia, nesta ordem:

1. `rejectionReason`;
2. `hardViolations` e seus códigos;
3. capacidade total e preferências;
4. avaliadores proibidos dos grupos;
5. métricas/debug notes.

## Inspeção visual

- OpenCode: `ui_inspector` está permitido em `opencode.json`;
- script do projeto: `agent_inspect.js` abre um Chromium separado;
- Computer Use: observa a janela nativa Wails, mas não faz parte do repositório.
