# Regras para agentes

## Branch principal e pull requests

- A branch principal deste projeto é **`mainjoao`**. Use `origin/mainjoao` como base de novos trabalhos e `mainjoao` como destino dos pull requests.
- Faça `git fetch origin mainjoao` e trabalhe em uma branch isolada/worktree. Não use `main` como destino padrão, nem substitua a arquitetura de `mainjoao` por implementações de outras branches.
- Preserve o solver, os contratos de `back/logic/workflow_definition.go` e os modelos de `back/type`. Porte funcionalidades para essa estrutura.
- Não faça push direto nem merge em `mainjoao`; entregue um PR revisável.

## Fontes de verdade

- O workflow, seus defaults, critérios soft e otimização base têm uma única fonte de verdade em `back/logic/workflow_definition.go`.
- Os modelos e metadados de persistência/mapeamento vêm de `back/type/types.go` e `back/type/metadata.go`.
- Não duplique essas regras no frontend, no CLI ou na documentação. Consumidores devem usar o contrato do backend.

## Implementação e testes

- Toda função nova deve ter teste.
- Sempre execute `go test ./...` e `make lint` depois de mudanças.
- Toda criação ou alteração de UI deve criar ou atualizar testes em `tools/ui-testing`.
- Para testes de UI, mantenha `wails dev` ativo, execute `tools/ui-testing/runner.sh` e examine `tools/ui-testing/test-results/report.json` em caso de falha.
- Em servidores sem GTK/WebKit, use o adaptador real `go run . preview` após `npm run build` no frontend e execute o mesmo runner com `UI_BASE_URL` apontando para o preview. Registre a falta das dependências nativas; o preview usa os mesmos métodos `App`, não substitui o backend por mocks.
- Agendas/configurações ficam em `operations.db`; destinatários são registros separados. Não volte a embutir o histórico na agenda nem crie polling contínuo para o histórico. Use eventos e paginação.
- O cache de agendas é descartável, limitado e expira por inatividade. Uma recuperação do disco deve respeitar o mesmo limite da criação.
- Atualize README e os documentos afetados junto com o código. O atualizador `make docs-update` é específico de Jujutsu; em worktrees Git, faça essa atualização diretamente.

- Diagnósticos de desempenho só podem existir em builds `debug`/`dev`. Mantenha um teste de release demonstrando que variáveis de ambiente não ativam amostragem ou a CLI de desempenho. Benchmarks usam dados sintéticos e bancos temporários.
- A substituição de importações deve incluir limpeza e todos os vínculos em uma transação; nunca limpe o banco no upload ou silencie erro de um registro.
