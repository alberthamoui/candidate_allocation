---
id: domain-models
title: Modelos de domínio
summary: Entidades importadas, configuração, solver, resultado e metadata derivada.
status: active
sources:
  - back/type/types.go
  - back/type/metadata.go
---

# Modelos de domínio

## Entidades importadas

### Candidato

Campos centrais: `timestamp`, `nome`, `cpf`, `numero`, `semestre`, `curso`, `email_secundario`, `email_pessoal`, `opcoes` e `extras`.

- obrigatórios no SQLite: nome, CPF, número, semestre, curso e os dois emails;
- único: CPF;
- duplicidade de revisão: CPF e os dois emails;
- `opcoes`: não é coluna de `pessoa`; vira disponibilidade relacionada;
- `extras`: JSON em TEXT, com valores `string | null`.

### Avaliador

Campos: `id`, `nome`, `email`, `sigla` e `extras`. Nome, email e sigla são obrigatórios, únicos e usados na duplicidade. `id` não é persistido pelo inserter de struct porque o SQLite o gera.

### Restrição

Campos: `candidato`, `naoPosso` e `prefiroNao`. O texto de avaliadores aceita valores separados por vírgula ou espaço e é convertido em vínculos por sigla.

## Mapeamento e detecção

- `MappingItem`: coluna, índice, variável e flag para extra nulo;
- `MappingFieldInfo`: required/unique/duplicate para a UI;
- `UniqueValueDetection`: valor original, normalizado e ocorrências;
- `CandidateCriterionColumn`: coluna central ou extra elegível.

## Configuração

- `PreferenceScheduleMapping`: preferência → dia/hora;
- `SoftCriterion`: tipo, coluna, valores e threshold;
- `AllocationParams`: limites e critérios;
- `AllocationConfiguration`: summary, normalized, diagnostics e result.

## Solver

- `AllocationProblem`: candidatos, grupos, hard restrictions e soft rules;
- `SolverCandidate`: preferências/ranks, atributos e restrições de avaliador;
- `SolverGroup`: schedule, avaliadores e limites;
- `PartialAllocationState`: assignments e membros;
- `SolverResult`: status, assignments, score, violações, métricas e debug.

## Explicabilidade

`SoftScoreBreakdown` lista componentes de penalidade. `AllocationQualityReport` lista características humanas com IDs dos sujeitos envolvidos. Esses contratos permitem explicar a decisão sem duplicar o solver na UI.

## Tipos legados

`Mesa`, `ResultadoAlocacao`, `Horario` e `HorarioInfo` atendem o algoritmo antigo de `service.go`. Não fazem parte do caminho configurado oficial.

## Metadata

`DescribeStruct` interpreta tags `json`, `db` e `app`. Para adicionar/alterar campo:

1. atualize a struct e tags;
2. verifique metadata e schema dinâmico;
3. ajuste builders/loaders explícitos;
4. regenere bindings;
5. atualize testes de metadata, banco, app e bindings.

`NullableString` é um tipo nomeado para manter JSON `string | null` sem gerar uma classe TypeScript chamada `string`.
