---
id: data-flow
title: Fluxo de dados
summary: Transformações desde o Excel até o painel de resultados.
status: active
sources:
  - back/logic/
  - back/db/
  - back/allocation/
  - frontend/src/
---

# Fluxo de dados

```text
Excel (3 abas)
  -> sugestão e edição de MappingItem
  -> structs de domínio + erros + grupos duplicados
  -> revisão manual
  -> SQLite
  -> AllocationConfiguration
  -> AllocationProblem
  -> solver exato
  -> SolverResult + AllocationQualityReport
  -> UIAllocationResult
  -> painel React
```

## Importação

O frontend lê o arquivo como bytes e chama `SuggestMapping(data, 5)`. O `App` guarda esses bytes para as chamadas subsequentes das abas 1 e 2. A convenção atual é:

1. índice 0: candidatos;
2. índice 1: avaliadores;
3. índice 2: restrições.

Os `MappingItem` transformam colunas em campos centrais, opções de horário ou extras. Candidatos e avaliadores aceitam extras; restrições não.

## Validação e revisão

Builders retornam structs junto de erros de campo e grupos de duplicidade. O frontend edita cópias em memória, resolve grupos duplicados e envia apenas registros aprovados aos métodos `Save*FromMaps`.

## Persistência

`logic.Save` abre `./insper.db` e usa metadata de `back/type` para inserir campos persistentes. Preferências viram registros em `opcoes_horario` e `disponibilidade`; restrições viram vínculos candidato–avaliador.

## Configuração

Na UI, os parâmetros vêm de `GetWorkflowDefinition` e as opções de coluna/valor vêm do SQLite. No processamento, `BuildAllocationConfigurationFromDatabase` recarrega os dados, detecta preferências e cria `AllocationConfiguration` normalizada.

Na CLI, dia e hora são coletados interativamente para cada preferência e a mesma função de configuração é usada.

## Adaptação para o solver

`BuildAllocationProblem`:

- cria grupos por horário e associa avaliadores em round-robin;
- converte a ordem das opções em ranks por grupo;
- resolve siglas de restrição para IDs;
- copia apenas atributos usados pelos critérios soft;
- ativa hard constraints e a política base oficial.

## Resultado

O solver devolve assignments, score, violações, métricas e debug notes. `App.RunAllocation` combina IDs com os registros originais, ordena mesas/pessoas, separa não alocados e adiciona `AllocationQualityReport`. O frontend usa os IDs do relatório para destacar candidatos e grupos sem recalcular regras.
