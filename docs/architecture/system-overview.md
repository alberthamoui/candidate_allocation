---
id: system-overview
title: Visão geral do sistema
summary: Componentes, processos e responsabilidades do Candidate Allocation.
status: active
sources:
  - main.go
  - app.go
  - back/
  - frontend/src/
---

# Visão geral do sistema

O sistema possui dois modos de entrada que reutilizam o mesmo domínio Go:

- desktop Wails: React executa dentro de uma WebView e chama métodos de `App` pelos bindings gerados;
- CLI: `go run . cli` conduz importação, configuração e solver pelo terminal.

## Componentes

| Camada | Responsabilidade | Fonte principal |
|---|---|---|
| Bootstrap | Selecionar desktop ou CLI e empacotar o frontend | `main.go`, `cli.go` |
| Ponte Wails | Ciclo de vida, estado transitório, métodos expostos e adaptação para a UI | `app.go` |
| Workflow CLI | Interação de terminal e orquestração da importação | `back/workflow/import_cli.go` |
| Domínio/importação | Excel, mapeamento, validação, duplicidade e persistência | `back/logic` |
| Contratos | Modelos JSON, solver e metadata derivada das structs | `back/type` |
| Persistência | Schema SQLite dinâmico e inserções | `back/db` |
| Alocação oficial | Configuração persistida → problema → solver exato → qualidade | `back/allocation/configured_service.go`, `problem_builder.go`, `exact_solver.go` |
| Frontend | Wizard, configuração, processamento e painel de resultado | `frontend/src` |
| Qualidade | Testes Go, bindings, Playwright e analisador customizado | `*_test.go`, `tools` |

## Processos de execução

No desktop, `main.go` cria um `App`, registra seus métodos no Wails e serve `frontend/dist` no build. Em desenvolvimento, `wails dev` conecta a WebView ao Vite. `App.startup` sincroniza o schema do banco e limpa os dados existentes antes de iniciar uma nova sessão.

Na CLI, `detectCLIMode` reconhece `cli`, `-cli` ou `--cli`. `runCLI` exige `-file`, aceita `-opcoes` e delega para `workflow.RunCLI`.

## Dependências principais

- `excelize`: leitura da planilha em memória;
- `go-sqlite3`: persistência local em `./insper.db`;
- Wails: integração Go/WebView e eventos;
- React Router: navegação do wizard e passagem transitória de configuração/resultado;
- Playwright: teste end-to-end contra o servidor Wails de desenvolvimento.

## Invariantes arquiteturais

- Defaults, tipos de critério soft e otimização base pertencem a `logic.WorkflowDefinition`.
- Tags das structs em `back/type/types.go` alimentam metadata, mapeamento e schema dinâmico.
- O frontend não deve reimplementar regras do solver; relatórios de score e qualidade já chegam calculados pelo backend.
- O caminho oficial de alocação é `RunConfiguredAllocation`. O algoritmo em `back/allocation/service.go` é legado.

## Estado e duração

- bytes do Excel e quantidade de opções ficam em memória no `App`;
- drafts de mapeamento e entidades em revisão ficam no estado do React;
- dados aprovados ficam no SQLite durante a sessão;
- configuração e resultado da alocação passam por `location.state` e não são persistidos;
- encerramento do app limpa o estado transitório; uma nova inicialização também limpa as tabelas.

Consulte [limitações conhecidas](../reference/known-limitations.md) antes de alterar persistência, sequência do workflow ou navegação direta.
