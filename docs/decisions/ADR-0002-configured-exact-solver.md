---
id: adr-0002
title: Solver exato configurado como caminho oficial
summary: O fluxo de produção usa AllocationConfiguration, adapter explícito e solver exato.
status: accepted
---

# ADR-0002: Solver exato configurado como caminho oficial

- Status: accepted
- Supersedes: algoritmo legado como caminho principal

## Contexto

O algoritmo antigo cria mesas aleatoriamente, usa constantes locais e faz alocação gulosa. Ele não consegue provar optimalidade nem explicar de forma estruturada hard constraints e penalidades.

## Decisão

O caminho oficial é:

`AllocationConfiguration → AllocationProblem → SolveAllocation → SolverResult`.

O solver exato usa hard constraints explícitas, score decomponível, branch-and-bound, min-cost flow, poda, paralelismo e métricas. `RunConfiguredAllocation` é o serviço de entrada para CLI e Wails.

## Consequências

- optimalidade é testável contra oráculo em instâncias pequenas;
- configuração e resultado têm contratos JSON estáveis;
- progresso e explicabilidade chegam ao frontend;
- a atribuição de avaliadores ainda é feita pelo adapter, não otimizada pelo solver;
- código legado permanece isolado até remoção segura.

## Alternativas consideradas

- manter heurística aleatória: rejeitada por não determinismo e baixa explicabilidade;
- usar apenas min-cost flow: insuficiente para todos os critérios acoplados;
- solver externo: não adotado para preservar execução local e controle do contrato.
