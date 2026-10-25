---
id: known-limitations
title: Limitações conhecidas
summary: Divergências e dívidas comprovadas no comportamento atual.
status: active
---

# Limitações conhecidas

Estas são descrições do código atual, não decisões desejadas. Ao corrigir uma delas, atualize os documentos e testes relacionados.

## Workflow visual não deriva integralmente do contrato

`WorkflowDefinition` possui sete etapas e coloca avaliadores antes de restrições. `workflowMeta.ts` possui oito macroetapas, coloca restrições antes de avaliadores e adiciona checkpoint/resultado. A configuração consome defaults/opções do backend, mas a timeline e ajuda ainda são hardcoded.

## Revisão de restrições antes dos avaliadores

O wizard revisa restrições antes dos avaliadores. O `App` mantém essas restrições pendentes e só as persiste depois de salvar os avaliadores, quando as siglas já possuem IDs. A CLI já segue essa ordem de persistência. Candidatos/siglas desconhecidos ainda exigem revisão dos dados importados.

## Banco de importação substituído por uma nova planilha

O startup preserva os dados; uma importação completamente revisada substitui o banco de domínio em uma transação. A agenda final e seus avisos ficam separados em `operations.db`. Não há catálogo de múltiplos processos seletivos na interface.

## Rascunho de importação ainda transitório

A agenda final, seus parâmetros, as avaliações e os canais persistem. Os mapeamentos e correções ainda em revisão dependem do estado React. Persistência do resultado não recupera automaticamente todas as telas anteriores.

## Opções de critério diferem entre domínio e UI

`ListCandidateCriterionColumns` suporta todos os campos centrais e extras. `GetCriteriaOptions`, usado pela tela, consulta apenas `semestre`, `curso` e `extras.*`.

## Avaliadores são atribuídos fora do solver

`problem_builder.go` distribui avaliadores em round-robin antes da busca. O solver otimiza candidatos nos grupos, não a composição dos avaliadores. Se faltarem avaliadores, usa os disponíveis, mesmo abaixo do parâmetro solicitado.

## Solver legado permanece no pacote

`service.go`, `load.go`, `print.go` e tipos legados continuam compilando/testados. Eles usam aleatoriedade e constantes diferentes do workflow oficial, o que aumenta o risco de alguém chamar o caminho errado.

## Tipagem frouxa no frontend

Partes relevantes de `main.tsx`, revisão e resultado usam `any`, reduzindo detecção estática de drift nos contratos.

## Input anuncia formatos além do fluxo testado

O input aceita `.xls` e `.csv`, mas a implementação abre dados com `excelize` e a suíte cobre principalmente `.xlsx`.

## Frontend compilado ainda versionado

`node_modules` e `build/bin` foram retirados do rastreamento e estão no `.gitignore`. `frontend/dist` ainda é versionado, portanto builds locais podem gerar diffs em artefatos compilados. O runner de UI repara Playwright incompleto com `npm ci`.

## Resíduos de template/utilitários

`Greet`, o título nativo `wails-events`, `draft_app.go` e `fix-helphint.js` não representam funcionalidades centrais. Scripts de build assumem execução a partir de `scripts/`, e o instalador do Wails usa `@latest`.

## Agenda e comunicação em mainjoao

A agenda final e as avaliações agora persistem em SQLite e são compartilhadas com os comandos `cli agenda`. O cache de snapshots é limitado e descartável. O rascunho React de importação continua transitório; persistência da agenda não implica recuperação de cada tela de mapeamento. O Google OAuth depende do cliente Desktop configurado para a instalação. O WhatsApp Business exige número e modelo aprovados; não há conexão de WhatsApp pessoal por QR Code ou cadastro integrado da Meta neste aplicativo desktop. Aceitação pelo provedor não é confirmação de entrega. Envios que estavam na chamada de rede durante uma interrupção exigem conferência; somente falhas confirmadas entram na repetição automática solicitada pelo administrador.

## Preparação inicial do solver

O prazo/cancelamento da busca é cooperativo. A preparação inicial por fluxo de custo mínimo pode exceder o prazo antes de a busca observar o cancelamento, principalmente em volumes grandes. As medições debug registram o tempo total; o prazo configurado não é uma garantia de latência. Consulte [validação e desempenho](../operations/debug-validation.md).
