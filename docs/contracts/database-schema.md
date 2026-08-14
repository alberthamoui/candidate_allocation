---
id: database-schema
title: Schema SQLite
summary: Banco padrão, tabelas dinâmicas, relações, sincronização e ciclo de vida.
status: active
sources:
  - back/db/default.go
  - back/db/schema.go
  - back/db/record.go
  - back/logic/logic.go
---

# Schema SQLite

## Banco padrão

O arquivo é `./insper.db`, relativo ao diretório de execução. `EnsureDefaultDatabase` abre o banco e executa `EnsureAppSchema`.

No desktop, `App.startup` chama `ClearDatabase`; portanto o banco funciona como armazenamento da sessão, não como histórico permanente.

## Tabelas

### Dinâmicas

`pessoa` e `avaliador` são criadas a partir de `CandidateFields()` e `AvaliadorFields()`:

- coluna `id INTEGER PRIMARY KEY AUTOINCREMENT`;
- campos `Persist=true` viram colunas;
- `required` adiciona `NOT NULL`;
- `unique` cria índice parcial `idx_<tabela>_<coluna>_unique`.

Na sincronização, colunas ausentes são adicionadas e índices únicos gerenciados que ficaram obsoletos são removidos. Colunas antigas não são removidas.

### Relacionais

| Tabela | Papel |
|---|---|
| `opcoes_horario` | valor normalizado de horário |
| `disponibilidade` | candidato, horário e rank de preferência |
| `restricoesNposso` | avaliador proibido para candidato |
| `restricoesPrefiroN` | avaliador evitado para candidato |

`disponibilidade` possui índice único em `(pessoa_id, horario_id, preferencia)`. Foreign keys são ativadas por `PRAGMA foreign_keys = ON`.

## Serialização

`InsertStruct` usa placeholders para valores. Campos INTEGER representados como string, como semestre, são convertidos; erro numérico impede a inserção. Maps em TEXT são serializados como JSON, e map nulo vira `{}`.

## Persistência do fluxo

- candidatos criam `pessoa`, horários e disponibilidades;
- avaliadores criam `avaliador`;
- restrições resolvem candidato por nome e avaliador por sigla antes de criar vínculos.

Valores de horário são lower-case e trimados. Extras preservam as chaves da importação e são JSON.

## Leitura para o solver

`LoadConfiguredAllocationData` reconstrói candidatos, opções ordenadas por preferência, avaliadores, extras e restrições. As funções antigas em `load.go` atendem o solver legado.

## Limitações transacionais

`logic.Save/fillDb` não usa uma transação única e registra algumas falhas individuais sem devolvê-las. A ordem atual da UI salva restrições antes dos avaliadores, embora os vínculos dependam deles. Consulte [limitações conhecidas](../reference/known-limitations.md).

## Testes

`back/db/schema_test.go` cobre criação, migração aditiva e remoção de índices obsoletos. `record_test.go` cobre persistência refletiva dos campos atuais.
