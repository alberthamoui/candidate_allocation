# Regras para agentes

## Fontes de verdade

- O workflow, seus defaults, critérios soft e otimização base têm uma única fonte de verdade em `back/logic/workflow_definition.go`.
- Os modelos e metadados de persistência/mapeamento vêm de `back/type/types.go` e `back/type/metadata.go`.
- Não duplique essas regras no frontend, no CLI ou na documentação. Consumidores devem usar o contrato do backend.

## Implementação e testes

- Toda função nova deve ter teste.
- Sempre execute `go test ./...` e `make lint` depois de mudanças.
- Toda criação ou alteração de UI deve criar ou atualizar testes em `tools/ui-testing`.
- Para testes de UI, mantenha `wails dev` ativo, execute `tools/ui-testing/runner.sh` e examine `tools/ui-testing/test-results/report.json` em caso de falha.
