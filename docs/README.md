---
id: documentation-index
title: Índice da documentação
summary: Ponto de entrada para humanos e LLMs localizarem a documentação canônica do projeto.
status: active
---

# Documentação do Candidate Allocation

Esta documentação descreve o comportamento existente. Código e testes continuam sendo as fontes de evidência; os documentos explicam responsabilidades, fluxos, contratos e decisões.

## Leitura recomendada

1. [Visão geral do sistema](architecture/system-overview.md)
2. [Fluxo de dados](architecture/data-flow.md)
3. [Mapa do código](reference/code-map.md)
4. Documento da feature ou contrato que será alterado
5. [Limitações conhecidas](reference/known-limitations.md)

## Arquitetura

- [Visão geral do sistema](architecture/system-overview.md): componentes, processos e responsabilidades.
- [Fronteira frontend/backend](architecture/frontend-backend-boundary.md): React, bindings Wails e estado.
- [Fluxo de dados](architecture/data-flow.md): planilha → domínio → SQLite → solver → UI.

## Funcionalidades

- [Importação](features/import-workflow.md)
- [Mapeamento e revisão](features/mapping-and-review.md)
- [Configuração da alocação](features/allocation-configuration.md)
- [Execução da alocação](features/allocation-execution.md)
- [Resultados e explicabilidade](features/allocation-results.md)

## Contratos

- [Definição oficial do workflow](contracts/workflow-definition.md)
- [Modelos de domínio](contracts/domain-models.md)
- [API Wails](contracts/wails-api.md)
- [Schema SQLite](contracts/database-schema.md)

## Operação

- [Desenvolvimento local e builds](operations/development.md)
- [Estratégia e comandos de teste](operations/testing.md)
- [Troubleshooting](operations/troubleshooting.md)
- [Atualização automática da documentação](operations/documentation-update.md)
- [Prompt do atualizador](prompts/documentation-updater.md)

## Decisões e referência

- [ADRs](decisions/README.md)
- [Glossário](reference/glossary.md)
- [Mapa do código](reference/code-map.md)
- [Limitações conhecidas](reference/known-limitations.md)
- [Inventário de bindings](generated/wails-bindings.md)
- [Inventário de testes](generated/test-inventory.md)

## Roteamento para LLMs

Uma LLM deve ler primeiro este índice e depois consultar [`manifest.yaml`](manifest.yaml). O manifesto relaciona arquivos alterados aos documentos que precisam ser revisados; ele não contém regras de negócio.

O marcador em [`documentation-state.yaml`](documentation-state.yaml) registra o último estado de código coberto pelo atualizador automático.
