---
id: code-map
title: Mapa do código
summary: Roteamento de responsabilidades para arquivos, documentos e testes.
status: active
---

# Mapa do código

## Raiz

| Path | Responsabilidade |
|---|---|
| `main.go` | bootstrap Wails, assets e seleção do modo CLI |
| `cli.go` | flags e delegação para o workflow CLI |
| `app.go` | ponte Wails, ciclo de vida e adaptação do resultado |
| `wails_bindings_test.go` | compatibilidade Go ↔ TypeScript |
| `Makefile` | lint padrão e analisador customizado |
| `wails.json` | comandos de build/dev do frontend |
| `draft_app.go` | rascunho sem comportamento de produção |
| `fix-helphint.js` | utilitário avulso, não integrado ao build principal |

## Backend

| Path | Responsabilidade | Documento |
|---|---|---|
| `back/type/types.go` | modelos JSON, solver e tipos legados | [modelos](../contracts/domain-models.md) |
| `back/type/metadata.go` | reflexão de tags para mapping/schema | [modelos](../contracts/domain-models.md) |
| `back/logic/workflow_definition.go` | fonte oficial do workflow | [workflow](../contracts/workflow-definition.md) |
| `back/logic/mapping_suggester.go` | similaridade e resolução de sugestões | [importação](../features/import-workflow.md) |
| `back/logic/logic.go` | Excel, builders, validação, duplicidade e save | [importação](../features/import-workflow.md) |
| `back/logic/preference_logic.go` | detecção de valores/colunas | [configuração](../features/allocation-configuration.md) |
| `back/logic/allocation_config_logic.go` | defaults, normalização, validação e envelope | [configuração](../features/allocation-configuration.md) |
| `back/logic/allocation_quantities.go` | contagem/distribuições de capacidade | [configuração](../features/allocation-configuration.md) |
| `back/db/default.go` | banco padrão | [SQLite](../contracts/database-schema.md) |
| `back/db/schema.go` | schema estático/dinâmico e limpeza | [SQLite](../contracts/database-schema.md) |
| `back/db/record.go` | inserção refletiva | [SQLite](../contracts/database-schema.md) |
| `back/db/auxfunctions.go` | inserts/queries relacionais | [SQLite](../contracts/database-schema.md) |
| `back/workflow/import_cli.go` | orquestração/prompt CLI | [importação](../features/import-workflow.md) |

## Alocação oficial

| Path | Responsabilidade |
|---|---|
| `configured_service.go` | carrega persistência e executa pipeline oficial |
| `problem_builder.go` | adapter domínio/configuração → solver |
| `hard_constraints.go` | viabilidade e violações hard |
| `soft_score.go` | score base e critérios adicionais |
| `exact_solver.go` | branch-and-bound conectado |
| `decomposition.go` | componentes independentes e progresso agregado |
| `min_cost_flow.go` | incumbente/completion e bounds por fluxo |
| `progress.go` | snapshots de branches |
| `quality.go` | explicabilidade humana |

`service.go`, `load.go` e `print.go` pertencem ao caminho legado.

## Frontend

| Path | Responsabilidade |
|---|---|
| `frontend/src/main.tsx` | rotas e estado transitório do wizard |
| `App.tsx` | upload e drafts iniciais |
| `MappingEditorPage.tsx` | editor compartilhado |
| `Mapping*Page.tsx` | wrappers dos três mappings |
| `EntityVerificationView.tsx` | revisão compartilhada |
| `Verify*.tsx` | wrappers/save/navegação por entidade |
| `AllocationConfigPage.tsx` | parâmetros e critérios |
| `AllocationLoadingPage.tsx` | progresso e execução |
| `AllocationResultPage.tsx` | painel e filtros |
| `workflowShell.tsx` | layout/componentes compartilhados |
| `workflowMeta.ts` | timeline e ajuda contextual da UI |
| `wailsReady.ts` | espera por bindings |
| `index.css` | tokens e estilos globais |
| `frontend/wailsjs` | código gerado pelo Wails |

## Ferramentas

| Path | Responsabilidade |
|---|---|
| `tools/ui-testing/runner.sh` | preflight e Playwright |
| `tools/ui-testing/tests` | testes end-to-end |
| `tools/ui-testing/agent_inspect.js` | DOM, acessibilidade e screenshot opcional |
| `tools/unexported/main.go` | analisador de APIs públicas não consumidas |
| `scripts` | instalação Wails e builds multiplataforma |

Para roteamento automático mais preciso, use [`docs/manifest.yaml`](../manifest.yaml).
