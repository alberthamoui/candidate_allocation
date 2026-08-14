---
id: workflow-definition
title: Contrato do workflow
summary: Fonte única de etapas, defaults, critérios soft e política base.
status: active
sources:
  - back/logic/workflow_definition.go
  - back/type/types.go
tests:
  - back/logic/workflow_definition_test.go
  - app_test.go
---

# Contrato do workflow

## Fonte de verdade

`logic.WorkflowDefinition`, em `back/logic/workflow_definition.go`, é a fonte oficial. CLI, Wails e frontend devem consumir esse retorno. Este documento descreve o contrato, mas não o substitui.

## Estrutura

`WorkflowDefinition` possui:

- `steps`: etapas oficiais com `key`, `label` e `required`;
- `defaultAllocationParams`: quatro limites e critérios vazios;
- `softCriterionOptions`: catálogo de regras selecionáveis;
- `baseOptimization`: penalidades sempre ativas.

## Etapas oficiais atuais

| Key | Label | Obrigatória |
|---|---|---|
| `import` | Importacao | sim |
| `candidate_mapping` | Mapeamento de candidatos | sim |
| `evaluator_mapping` | Mapeamento de avaliadores | sim |
| `restriction_mapping` | Mapeamento de restricoes | sim |
| `verification` | Verificacao | sim |
| `allocation_config` | Configuracao de alocacao | sim |
| `allocation_run` | Execucao | sim |

A timeline visual em `frontend/src/workflowMeta.ts` atualmente usa oito macroetapas e ordem diferente para restrições/avaliadores. Essa divergência é conhecida e não muda a autoridade do backend.

## Defaults

- grupos por horário: 2;
- mínimo por grupo: 4;
- máximo por grupo: 8;
- avaliadores por grupo: 3;
- critérios soft: lista vazia.

## Otimização base

- `preferencePenaltyByRank`: `[0, 1, 2, 3, 4]`;
- `avoidEvaluatorPenalty`: `3`.

Ela continua ativa mesmo sem critérios adicionais.

## Catálogo de critérios

O catálogo contém tipo estável, label, descrição e `requiresThreshold`. Os tipos válidos são `min_value`, `at_least_one_each`, `balanced_distribution`, `group_together` e `max_value`.

## Alteração segura

1. edite `workflow_definition.go` e, se necessário, os tipos em `back/type/types.go`;
2. atualize normalização, validação e score do critério;
3. faça CLI e frontend consumirem o contrato, sem novas constantes duplicadas;
4. regenere bindings quando o shape mudar;
5. atualize testes de definição, app, bindings, solver e UI;
6. atualize os documentos mapeados no manifesto.
