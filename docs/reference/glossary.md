---
id: glossary
title: Glossário
summary: Vocabulário do domínio, UI, persistência e solver.
status: active
---

# Glossário

| Termo | Significado |
|---|---|
| Candidato | Pessoa a ser distribuída em um grupo/mesa. A tabela histórica chama a entidade de `pessoa`. |
| Avaliador | Pessoa associada a uma mesa para avaliar candidatos. |
| Restrição `NaoPosso` | Proibição hard entre candidato e avaliador. |
| Restrição `PrefiroNao` | Preferência negativa soft, com penalidade quando violada. |
| Opção/preferência | Valor de horário informado pelo candidato, ordenado por rank. |
| Horário/schedule | Valor normalizado que origina um ou mais grupos. |
| Grupo/mesa | Unidade final com capacidade, schedule e avaliadores. `SolverGroup` usa “grupo”; a UI apresenta “Mesa”. |
| Mapping | Associação de uma coluna do Excel a um campo do domínio. |
| Campo central | Campo previsto na struct Go e descrito pela metadata. |
| Extra | Coluna específica do cliente armazenada como JSON. |
| Draft | Estado editável de UI ainda não enviado ao backend. |
| Duplicidade | Grupo de registros conectado por um ou mais campos marcados `duplicate`. |
| WorkflowDefinition | Contrato oficial de etapas, defaults e opções. |
| AllocationConfiguration | Envelope normalizado/diagnóstico antes do solver. |
| AllocationProblem | Modelo explícito consumido pelo solver. |
| Hard constraint | Regra cuja violação invalida a solução. |
| Soft criterion | Regra desejável que adiciona penalidade. |
| Score | Soma de penalidades; menor é melhor. |
| Incumbente | Melhor solução completa conhecida durante branch-and-bound. |
| Lower bound | Limite otimista usado para podar uma subárvore incapaz de melhorar o incumbente. |
| Infeasible | Nenhuma solução satisfaz todas as hard constraints. |
| Quality characteristic | Explicação clicável de um aspecto do resultado com IDs relacionados. |
| Binding Wails | Código gerado que permite ao TypeScript chamar métodos Go. |
| Sentinel | Arquivo opcional que sinaliza startup do Wails em smoke tests. |
