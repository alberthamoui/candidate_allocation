---
id: development
title: Desenvolvimento local
summary: Pré-requisitos, comandos, modos de execução, build e ferramentas auxiliares.
status: active
sources:
  - go.mod
  - wails.json
  - Makefile
  - frontend/package.json
  - scripts/
  - opencode.json
---

# Desenvolvimento local

## Pré-requisitos

- Go 1.25 compatível com `go.mod`;
- Node.js e npm;
- Wails CLI 2.x;
- compilador C para `github.com/mattn/go-sqlite3`;
- `golangci-lint` e `goimports` para o fluxo completo de qualidade.

Instalação do Wails disponível em `scripts/install-wails-cli.sh`; o script usa `@latest`, portanto valide compatibilidade com `wails.json` antes de atualizar a versão local.

## Desktop

Na raiz:

```bash
wails dev
```

O Wails executa `npm install`/`npm run dev` em `frontend`, gera bindings e serve o proxy em `http://localhost:34115`; o Vite costuma usar `http://localhost:5173` internamente.

## CLI

```bash
go run . cli -file Execelteste/Base4Restricao.xlsx
go run . cli -file caminho.xlsx -opcoes 5
```

Aliases aceitos: `cli`, `-cli` e `--cli`. Sem um deles, o processo inicia o desktop.

## Frontend isolado

```bash
cd frontend
npm install
npm run dev
npm run build
```

O build executa TypeScript e Vite, gerando `frontend/dist`, que é embutido pelo Go. O frontend isolado não fornece automaticamente os bindings reais do Wails.

## Build desktop

```bash
wails build
```

Os scripts em `scripts/` fazem builds limpos para plataforma atual, Windows amd64, macOS Intel, ARM64 e universal. Eles começam com `cd ../`, portanto foram escritos para execução a partir do diretório `scripts`.

## Banco

`./insper.db` é criado no diretório corrente (ou no caminho de `CANDIDATE_ALLOCATOR_DATABASE`). Uma nova importação válida limpa esse banco; o startup preserva os dados. A agenda final, as avaliações, os canais e o histórico de avisos ficam em `operations.db` no diretório de configuração do usuário, configurável por `CANDIDATE_ALLOCATOR_DATA_DIR`. Veja [operações da agenda](../features/agenda-operations.md).

## OpenCode e inspeção de UI

`opencode.json` habilita e permite `ui_inspector`. O script local `tools/ui-testing/agent_inspect.js` é independente: abre o Chromium contra a porta 34115 e salva HTML/árvore de acessibilidade, com screenshot opcional.

## Arquivos gerados

- `frontend/dist`: build web;
- `frontend/wailsjs`: bindings e modelos Wails;
- `build`: pacotes Wails;
- `tools/ui-testing/test-results` e `/tmp/candidate-allocation-playwright`: testes de UI.

`node_modules` e `build/bin` são ignorados e não devem voltar ao controle de versão. Não edite artefatos gerados como fonte primária.

## Atualização da documentação

Depois de um change de código, execute:

```bash
make docs-update
```

O comando exige `jj`, `codex` autenticado e um repositório com o marcador `docs/documentation-state.yaml`. Use `make docs-update-dry-run` para inspecionar o intervalo sem criar change nem iniciar o Codex.

A branch principal é `mainjoao`; use `origin/mainjoao` como base e `mainjoao` como destino dos PRs.

Em host sem GTK/WebKit, compile o frontend e execute `go run . preview` (loopback na porta 8128). Esse adaptador de desenvolvimento chama os métodos reais do `App`. O runner pode apontar para ele com `UI_BASE_URL=http://127.0.0.1:8128`; `PLAYWRIGHT_CHROMIUM_PATH` permite usar um Chromium já instalado.
