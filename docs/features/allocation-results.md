---
id: allocation-results
title: Resultados e explicabilidade
summary: Adaptação do solver para a UI, painel, filtros, qualidade e diagnóstico.
status: active
sources:
  - app.go
  - back/allocation/quality.go
  - frontend/src/AllocationResultPage.tsx
---

# Resultados e explicabilidade

## Contrato de UI

`App.RunAllocation` converte `SolverResult` em `UIAllocationResult`:

- `status`: mensagem humana (`Sucesso!`, `Alocação Parcial` ou `Impossível`);
- `solverStatus`: status técnico;
- mesas com candidatos e avaliadores;
- candidatos não alocados;
- score e componentes;
- relatório de qualidade;
- violações, motivo de rejeição, métricas e debug notes.

Os candidatos incluem semestre numérico, curso, emails, opções, restrições já humanizadas e extras. Avaliadores incluem sigla, email, restrições reversas e extras.

## Relatório de qualidade

O backend produz características clicáveis com IDs de candidatos, avaliadores e grupos:

- quantidade por rank de preferência;
- conflitos `PrefiroNao`;
- um item para cada critério soft configurado.

Cada característica possui código, label, descrição, valor, penalidade e tom. O frontend apenas apresenta e destaca os IDs; não recalcula score.

## Painel

O resultado apresenta:

- resumo de mesas, horários, não alocados e score;
- tooltip com componentes positivos do score;
- cards de qualidade que destacam candidatos relacionados;
- busca por nome/curso/semestre;
- filtros por semestre e curso;
- mesas agrupadas por horário;
- popover de detalhes de candidato ou avaliador;
- seção de não alocados;
- diagnóstico técnico recolhível com métricas do solver.

Busca, filtros e destaque de qualidade são mutuamente exclusivos para deixar claro qual regra visual está ativa.

## Ausência de resultado

O resultado chega por `location.state`. Quando a rota é aberta diretamente ou recarregada, a tela mostra um estado vazio e orienta voltar à configuração. Nenhum histórico de rodadas é persistido.

## Ordenação

Mesas são ordenadas por ID; candidatos dentro das mesas e não alocados são ordenados por nome. Mesas vazias não aparecem no resultado final.
