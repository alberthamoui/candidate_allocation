---
id: testing
title: Testes e lint
summary: Pirâmide de testes, comandos obrigatórios, Playwright e diagnóstico.
status: active
sources:
  - Makefile
  - tools/ui-testing/
  - tools/unexported/
  - '**/*_test.go'
---

# Testes e lint

## Validação obrigatória

Na raiz:

```bash
go test ./...
make lint
```

`make lint` executa `golangci-lint` e o analisador local `tools/unexported`, que encontra funções públicas não consumidas fora do pacote. A allowlist cobre pacotes com APIs públicas intencionais.

Para validar o frontend compilável:

```bash
cd frontend
npm run build
```

## Cobertura Go por camada

- `back/type`: JSON nullable e invariantes de metadata;
- `back/db`: schema dinâmico, migração aditiva, índices e inserção;
- `back/logic`: Excel, sugestão, mappings, validação, duplicidade, configuração e quantidades;
- `back/workflow`: CLI, prompts e seleção de duplicados;
- `back/allocation`: adapter, hard constraints, score, flow, solver, oráculo, qualidade e integração;
- raiz: modo CLI, delegação de `App`, ciclo de vida e integridade dos bindings.

## UI com Playwright

Mantenha o servidor ativo:

```bash
wails dev
```

Em outro terminal:

```bash
cd tools/ui-testing
./runner.sh
./runner.sh tests/05-allocation.spec.ts
```

O runner:

- valida se `@playwright/test` carrega e usa `npm ci` se a instalação estiver incompleta;
- instala Chromium apenas se o executável estiver ausente;
- espera `UI_BASE_URL` (default `http://localhost:34115`) por até 60 segundos;
- executa um worker, sem paralelismo, pois Wails/SQLite compartilham estado;
- grava `test-results/report.json` e screenshots somente em falha.

Variáveis: `UI_BASE_URL` e `UI_WAIT_TIMEOUT_SECONDS`.

## Suítes UI

1. home, erro sem arquivo e ajuda;
2. conflitos de campos extras;
3. duplicidade e edição inline;
4. jornada completa;
5. configuração, progresso, resultados e explicabilidade;
6. header, scroll e ajuda.

Os helpers em `tests/support/ui.ts` concentram importação e avanço do wizard. Bindings do backend são mockados em testes específicos quando o objetivo é isolar UI de configuração/resultado.

## Inspector local

```bash
cd tools/ui-testing
node --test agent_inspect.test.js
node agent_inspect.js --route=/ --out-prefix=/tmp/agent_inspect
node agent_inspect.js --route=/allocation-config --screenshot=true
```

Por padrão, o inspector salva HTML e árvore de acessibilidade; PNG somente com `--screenshot=true`.

## Fixtures

`Execelteste/Base4Restricao.xlsx` e `BaseRestricao.xlsx` são planilhas de teste. Mudanças nelas podem alterar mappings, duplicidades, persistência e jornada completa e devem ser tratadas como mudanças de comportamento.

Veja [inventário de testes](../generated/test-inventory.md) para o mapa arquivo → responsabilidade.
