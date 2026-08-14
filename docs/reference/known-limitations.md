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

## Restrição é salva antes do avaliador na UI

O desktop persiste candidatos, depois restrições e só então avaliadores. `fillDb` precisa encontrar avaliador por sigla para inserir a restrição. Como erros individuais são apenas impressos, a UI pode avançar sem vínculos persistidos. A CLI salva avaliadores antes de restrições e não tem essa ordem específica.

## Persistência sem transação e com erros silenciosos

`logic.Save` não usa transação e `fillDb` não retorna erro. Falhas de registros individuais podem produzir persistência parcial enquanto a chamada Wails retorna sucesso.

## Banco limpo no startup

`App.startup` sempre chama `ClearDatabase`. Não há histórico entre sessões nem mecanismo de confirmação/recuperação.

## Configuração e resultado não são persistidos

Parâmetros e resultado passam por React Router `location.state`. Recarregar ou abrir rotas diretamente perde o contexto.

## Mapeamento de horário no desktop é provisório

`BuildConfigurationFromPersistedData` mapeia cada preferência usando o próprio texto como dia e `horario importado` como hora. A UI ainda não permite editar `PreferenceScheduleMapping`; a CLI permite.

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
