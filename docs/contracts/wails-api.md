---
id: wails-api
title: API Wails
summary: Métodos públicos de App agrupados por responsabilidade e regras de compatibilidade.
status: active
sources:
  - app.go
  - frontend/wailsjs/go/main/App.d.ts
  - frontend/wailsjs/go/models.ts
---

# API Wails

O contrato consumido pelo frontend é gerado a partir dos métodos públicos de `App`. Consulte [inventário completo](../generated/wails-bindings.md) para assinaturas.

## Importação e mapeamento

- `SuggestMapping`, `SuggestMappingAvaliador`, `SuggestMappingRestricao`;
- `GetCandidateMappingFieldInfos`, `GetAvaliadorMappingFieldInfos`, `GetRestricaoMappingFieldInfos`;
- `BuildUsuariosWithMapping`, `BuildAvaliadoresWithMapping`, `BuildRestricoesWithMapping`;
- `SaveUsuariosFromMaps`, `SaveAvaliadoresFromMaps`, `SaveRestricoesFromMaps`.

`SuggestMapping` também guarda os bytes da planilha e `nOpcoes` no estado transitório de `App`. As outras sugestões/builders dependem desse estado.

## Configuração

- `GetWorkflowDefinition`, `DefaultAllocationParams`, `GetSoftCriterionOptions`;
- `DetectUniquePreferenceValues`, `ListCandidateCriterionColumns`, `DetectUniqueCandidateColumnValues`;
- `NormalizePreferenceScheduleMappings`, `NormalizeSoftCriteria`;
- `ValidatePreferenceScheduleMappings`, `ValidateSoftCriteria`, `ValidateAllocationParams`;
- `BuildAllocationConfiguration`, `BuildAllocationConfigurationFromDatabase`;
- funções de contagem combinatória.

## Execução

- `GetCriteriaOptions`: valores persistidos para a tela;
- `RunAllocation`: executa o caminho oficial e retorna `UIAllocationResult`;
- evento `allocation:progress`: snapshots assíncronos do solver.

## Utilitário

`Greet` é remanescente do template Wails e não participa do fluxo de negócio.

## Ciclo de vida

- `startup`: guarda contexto, sincroniza e limpa o banco, escreve sentinel opcional;
- `beforeClose`: limpa bytes/contexto;
- `shutdown`: repete a limpeza e remove sentinel.

`CANDIDATE_ALLOCATOR_WAILS_SMOKE_FILE` permite que testes externos observem a inicialização por um arquivo `ok` removido no shutdown.

## Compatibilidade

Alterações de nome, argumento, retorno ou tag JSON exigem regenerar `frontend/wailsjs`. `wails_bindings_test.go` verifica que classes e métodos essenciais ainda existem e que `NullableString` não produz modelo inválido.

Arquivos gerados nunca são a fonte original: corrija Go primeiro.
