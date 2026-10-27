# Instruções do repositório

A branch principal é `main`. Abra PRs com destino a `main`; não faça merge sem pedido explícito. Use worktree e branch próprias e preserve alterações existentes.

## Arquitetura

- Esta branch é o app web Go HTTP + React, com frontend embarcado. Preserve o solver modular atual; não substitua pela arquitetura Wails da `mainjoao`.
- App e CLI devem usar o mesmo serviço Go, DTOs, regras e estado de sessão. `cli.go` é cliente HTTP da API; não implemente solver, validações de negócio ou armazenamento paralelos na CLI ou no React.
- Centralize operações em `Session`; handlers e componentes só adaptam entradas e apresentam respostas. Toda nova operação precisa estar acessível pela API e pela CLI.
- Alterações na agenda e autorização de avisos devem ser atômicas. Retornos dos provedores são gravados por entrega, sem regravar o histórico inteiro.
- Envio é opcional. Realocações só enviam com autorização do administrador. Nunca use matrícula como telefone; CPF e preferências de horário podem ser omitidos.
- Sugestões orientam a realocação; escolhas manuais continuam livres, com alertas.
- Senhas/tokens nunca entram em logs, respostas, fixtures, commits ou armazenamento do navegador. Não faça testes de envio para destinatários reais sem autorização.

## Validação

- Compile o frontend antes de testar Go, por causa do embed: `npm --prefix frontend ci`, `npm --prefix frontend run build`.
- Execute `go test -race ./...`, lint pelo Makefile e `npm --prefix frontend run test:e2e` para mudanças na interface/API.
- O Playwright sobe o servidor isolado com diretório temporário de dados. Se necessário, `PLAYWRIGHT_CHROMIUM_EXECUTABLE` seleciona o Chromium instalado.
- Para preview manual, use porta e `ALOCACAO_DADOS` exclusivos da worktree. Mantenha o preview disponível e entregue o link validado; nunca substitua uma instância de produção.
