---
id: generated-test-inventory
title: Inventário de testes
summary: Mapa mecânico dos arquivos de teste e responsabilidades cobertas.
status: generated
sources:
  - '**/*_test.go'
  - tools/ui-testing/tests/
  - tools/ui-testing/agent_inspect.test.js
---

# Inventário de testes

Este inventário orienta seleção de testes. Os próprios testes são a fonte dos casos exatos.

## Raiz

| Arquivo | Cobertura |
|---|---|
| `cli_test.go` | detecção do modo CLI |
| `app_test.go` | delegação, workflow, ciclo de vida e adaptação para UI |
| `wails_bindings_test.go` | modelos/métodos gerados e JSON fields |

## Tipos e banco

| Arquivo | Cobertura |
|---|---|
| `back/type/types_test.go` | `NullableString` em JSON |
| `back/type/metadata_test.go` | invariantes e infos de mapping |
| `back/db/schema_test.go` | criação, adição de coluna e índices obsoletos |
| `back/db/record_test.go` | inserção de candidatos/avaliadores |

## Importação e configuração

| Arquivo | Cobertura |
|---|---|
| `back/logic/logic_test.go` | Excel, builders, validação, duplicidade, extras e save |
| `back/logic/mapping_suggester_test.go` | headers embaralhados e números |
| `back/logic/avaliador_test.go` | validação/duplicidade de avaliadores |
| `back/logic/preference_logic_test.go` | preferências, colunas e valores únicos |
| `back/logic/allocation_config_logic_test.go` | defaults, normalização, validação e configuração |
| `back/logic/allocation_quantities_test.go` | distribuições e contagem |
| `back/logic/workflow_definition_test.go` | contrato oficial e catálogo soft |
| `back/workflow/import_cli_test.go` | CLI, prompts, duplicados e critérios |

## Solver

| Arquivo | Cobertura |
|---|---|
| `hard_constraints_test.go` | violações parciais/completas |
| `soft_score_test.go` | política base e cinco critérios |
| `problem_builder_test.go` | groups, ranks, atributos, restrições e avaliadores |
| `min_cost_flow_test.go` | capacidade e mínimo por grupo |
| `exact_solver_test.go` | optimalidade, inviabilidade, poda, progresso e paralelo |
| `exact_solver_oracle_test.go` | comparação exaustiva e decomposição |
| `configured_service_test.go` | pipeline persistido oficial |
| `quality_test.go` | características, labels, tons e IDs |
| `solver_integration_test.go` | integração do núcleo |
| `service_test.go` | caminho legado |

## UI Playwright

| Spec | Cobertura |
|---|---|
| `01-home.spec.ts` | home, erro e ajuda |
| `02-mapping-erros.spec.ts` | validação de extras |
| `03-verification.spec.ts` | duplicidade e edição inline |
| `04-full-journey.spec.ts` | wizard completo |
| `05-allocation.spec.ts` | configuração, progresso, resultado e qualidade |
| `06-layout.spec.ts` | header, scroll e ajuda |

`agent_inspect.test.js` cobre parsing e screenshot opt-in do inspector. `tests/support/ui.ts` contém helpers de jornada, não testes independentes.
