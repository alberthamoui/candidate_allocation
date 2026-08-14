# Candidate Allocation

Aplicação desktop e CLI para importar candidatos, restrições e avaliadores de uma planilha, revisar os dados, configurar critérios e executar uma alocação exata em grupos.

## Stack

- Go 1.25
- Wails 2.10
- SQLite
- React, TypeScript, Vite e Tailwind CSS
- Playwright para testes de UI

## Executar

Pré-requisitos: Go, Node.js/npm, Wails CLI e um compilador C compatível com `go-sqlite3`.

```bash
wails dev
```

Modo CLI:

```bash
go run . cli -file caminho/para/arquivo.xlsx
go run . cli -file caminho/para/arquivo.xlsx -opcoes 5
```

## Validar

```bash
go test ./...
make lint
```

Para a UI, mantenha `wails dev` ativo e execute:

```bash
cd tools/ui-testing
./runner.sh
```

## Documentação

- [Índice completo](docs/README.md)
- [Visão geral da arquitetura](docs/architecture/system-overview.md)
- [Fluxo de importação](docs/features/import-workflow.md)
- [Execução da alocação](docs/features/allocation-execution.md)
- [Desenvolvimento local](docs/operations/development.md)
- [Testes](docs/operations/testing.md)
- [Limitações conhecidas](docs/reference/known-limitations.md)

Para atualizar a documentação depois de mudanças no código, comece por [`docs/manifest.yaml`](docs/manifest.yaml) e siga [`docs/operations/documentation-update.md`](docs/operations/documentation-update.md).
