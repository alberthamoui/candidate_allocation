---
id: allocation-execution
title: Execução da alocação
summary: Adapter, hard constraints, score, solver exato, progresso e caminho legado.
status: active
sources:
  - back/allocation/
  - back/type/types.go
  - app.go
---

# Execução da alocação

## Caminho oficial

```text
BuildAllocationConfigurationFromDatabase
  -> RunAllocation
  -> RunConfiguredAllocation
  -> LoadConfiguredAllocationData
  -> BuildAllocationProblem
  -> SolveAllocation
  -> BuildAllocationQualityReport
  -> UIAllocationResult
```

`RunConfiguredAllocation` rejeita conexão nula e configurações com `Diagnostics.HasErrors`. O desktop usa quatro workers e profundidade paralela dois.

## Adapter

`BuildAllocationProblem` cria grupos para cada mapping de preferência. IDs são sequenciais; o label segue `dia hora grupo N`. Avaliadores são ordenados e distribuídos circularmente entre grupos. Se o número solicitado exceder o disponível, todos os avaliadores são usados.

Cada candidato recebe:

- ID sequencial pela ordem carregada;
- grupos permitidos e rank por grupo;
- atributos necessários aos critérios;
- IDs de avaliadores proibidos e evitados.

Nomes de candidatos e siglas de avaliadores precisam coincidir com as restrições persistidas.

## Hard constraints

O problema oficial ativa:

- todos os candidatos devem ser atribuídos;
- somente grupos das preferências do candidato;
- capacidade máxima;
- proibição de avaliadores em `NaoPosso`;
- mínimo de candidatos em grupos usados no estado completo.

Violações têm códigos estáveis, incluindo `candidate_not_found`, `group_not_found`, `candidate_duplicate`, `group_out_of_preference`, `forbidden_evaluator`, `missing_assignment`, `group_below_min_candidates` e testemunhas de inviabilidade futura.

## Score

Quanto menor a penalidade, melhor. A política base sempre aplica:

- penalidades de rank `[0,1,2,3,4]`;
- penalidade 3 para cada conflito com `PrefiroNao`.

Depois são somados os componentes dos critérios configurados. Estados incompletos ou inválidos recebem penalidade sentinela de 1.000.000.

## Solver exato

O solver usa branch-and-bound e prova optimalidade. Principais mecanismos:

- incumbente inicial por min-cost flow;
- escolha do próximo candidato pela quantidade de opções viáveis;
- opções ordenadas por penalidade imediata e conflitos;
- poda por hard constraint futura;
- lower bound de score e min-cost completion;
- eliminação de grupos simétricos;
- fronteira paralela com incumbente compartilhado;
- decomposição de componentes independentes quando não há critérios adicionais e preferências são hard.

O resultado é `optimal` ou `infeasible`. Métricas registram nós, estados completos, podas hard/bound/flow, simetrias, atualizações do melhor resultado e tarefas paralelas.

## Progresso

O tracker estima o total como produto das opções estáticas por candidato, usa `big.Int`, contabiliza subárvores resolvidas/podadas e limita emissões a cada 100 ms. O evento Wails é `allocation:progress`.

## Caminho legado

`back/allocation/service.go`, `load.go` e `print.go` mantêm um algoritmo antigo, aleatório e guloso, com constantes próprias. Ele não é chamado por `RunConfiguredAllocation` e não deve receber novas regras do workflow. Consulte [limitações conhecidas](../reference/known-limitations.md).

## Evidência de correção

Testes comparam o solver a um oráculo exaustivo em problemas pequenos, validam optimalidade conhecida, inviabilidade, equivalência paralelo/sequencial, decomposição, flow, hard constraints, score e integração completa.
