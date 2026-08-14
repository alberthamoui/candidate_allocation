---
id: adr-0001
title: Workflow com fonte única no backend
summary: Defaults e opções oficiais pertencem ao contrato Go compartilhado.
status: accepted
---

# ADR-0001: Workflow com fonte única no backend

- Status: accepted
- Supersedes: nenhum

## Contexto

CLI, Wails e frontend precisam exibir as mesmas etapas, defaults, critérios soft e política base. Constantes duplicadas evoluem de forma independente e tornam testes/documentação inconsistentes.

## Decisão

`back/logic/workflow_definition.go` define `WorkflowDefinition`. Consumidores obtêm o contrato por chamada Go direta ou `GetWorkflowDefinition` no Wails. Tipos ficam em `back/type/types.go`.

## Consequências

- novas opções exigem suporte completo em normalização, validação, solver e UI;
- bindings e testes de contrato detectam drift;
- documentação referencia a fonte, sem se tornar fonte concorrente;
- a timeline frontend atual ainda precisa convergir totalmente para esse contrato, registrada como limitação.

## Alternativas consideradas

- manter defaults separados no CLI e React: rejeitado por drift;
- tornar o frontend fonte: rejeitado porque CLI e solver são Go;
- gerar tudo de um JSON externo: não adotado; adicionaria outra camada sem necessidade atual.
