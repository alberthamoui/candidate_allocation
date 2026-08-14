---
id: import-workflow
title: Fluxo de importação
summary: Leitura da planilha, sugestão de colunas e diferenças entre desktop e CLI.
status: active
sources:
  - back/logic/logic.go
  - back/logic/mapping_suggester.go
  - back/workflow/import_cli.go
  - frontend/src/App.tsx
tests:
  - back/logic/logic_test.go
  - back/workflow/import_cli_test.go
  - tools/ui-testing/tests/01-home.spec.ts
---

# Fluxo de importação

## Entrada

O desktop aceita `.xlsx`, `.xls` e `.csv` no input, mas o backend usa `excelize` e o fluxo documentado/testado é o de planilhas Excel. O frontend sempre solicita cinco colunas virtuais de preferência. A CLI aceita `-opcoes` e usa cinco como default.

## Sugestão de mapeamento

`SuggestMappingForSheet` lê o header e compara cada campo esperado com as colunas. A normalização:

- ignora caixa, espaços, pontuação, `_` e `-`;
- separa camelCase e transições entre letras e números;
- normaliza ordinais e números por extenso de 1 a 10;
- pontua coincidência exata, sequência de tokens e tokens compartilhados;
- resolve conflitos para que uma coluna não seja sugerida para dois campos.

Quando há menos colunas que variáveis, campos restantes ficam sem coluna (`indice` negativo ou nome vazio) e podem ser corrigidos na UI/CLI.

## Construção das entidades

- candidatos: aba 0, campos centrais, `opcao 1..N` e extras;
- avaliadores: aba 1, linhas completamente vazias ignoradas;
- restrições: aba 2, somente campos `candidato`, `naoPosso` e `prefiroNao`.

Índices negativos e mappings com variável vazia são ignorados. Um extra manual sem coluna pode ser preservado como `null` quando `includeWhenUnmapped` estiver ativo.

## Validação

Candidatos normalizam CPF, número e emails. As validações atuais exigem:

- CPF com 11 dígitos;
- número com 9 dígitos;
- semestre entre 1 e 10;
- email pessoal em formato básico;
- email secundário no domínio `al.insper.edu.br`.

Avaliadores exigem nome, email e sigla não vazios. Duplicidades são derivadas da metadata do domínio e podem conectar registros transitivamente.

## Desktop

O upload gera os três drafts e as três listas de metadata de uma vez. O fluxo navega candidatos → restrições → avaliadores, com revisão e persistência após cada mapeamento.

## CLI

`RunCLI` valida o arquivo, limpa/sincroniza o banco, oferece edição interativa dos mappings, escolhe o primeiro registro válido de cada grupo duplicado, coleta configuração e executa a alocação. Os prompts aceitam defaults ao pressionar Enter.

## Falhas esperadas

- arquivo ausente ou ilegível;
- aba esperada inexistente;
- planilha sem dados além do header;
- payload de mapping inválido;
- tipos incompatíveis na conversão para structs.

Consulte [mapeamento e revisão](mapping-and-review.md) para regras de edição e [limitações](../reference/known-limitations.md) para a ordem atual de persistência das restrições.
