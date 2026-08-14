---
id: frontend-backend-boundary
title: Fronteira frontend e backend
summary: Como React chama Go, mantém estado e recebe eventos pelo Wails.
status: active
sources:
  - app.go
  - frontend/src/main.tsx
  - frontend/src/wailsReady.ts
  - frontend/wailsjs/go/main/App.d.ts
---

# Fronteira frontend e backend

## Binding de chamadas

Métodos públicos de `App` em `app.go` são expostos pelo Wails. Os arquivos em `frontend/wailsjs` são gerados e não devem ser editados manualmente. `App.d.ts` define a assinatura TypeScript e `models.ts` espelha os tipos JSON de Go.

Fluxo típico:

1. um componente importa uma função de `frontend/wailsjs/go/main/App`;
2. o runtime Wails serializa argumentos para Go;
3. `App` delega para `back/logic`, `back/db` ou `back/allocation`;
4. o retorno é serializado para o frontend.

`wails_bindings_test.go` protege campos e métodos críticos contra bindings desatualizados ou inválidos.

## Estado no frontend

`Root`, em `frontend/src/main.tsx`, mantém:

- drafts dos três mapeamentos;
- metadata de campos;
- candidatos, restrições e avaliadores construídos;
- grupos e campos de duplicidade.

Esse estado permite voltar da revisão para o mapeamento sem perder ajustes, mas desaparece ao recarregar a WebView.

`AllocationConfigPage` envia os parâmetros por `navigate(..., {state})`. `AllocationLoadingPage` envia o resultado da mesma forma para `AllocationResultPage`. Abrir essas rotas diretamente não recria o estado anterior.

## Rotas

| Rota | Papel |
|---|---|
| `/` | upload e sugestão dos mapeamentos |
| `/mapping`, `/verify` | candidatos |
| `/mappingRestricoes`, `/verifyRestricoes` | restrições |
| `/mappingAvaliadores`, `/verifyAvaliadores` | avaliadores |
| `/success` | checkpoint após persistência |
| `/allocation-config` | parâmetros e critérios |
| `/allocation-loading` | configuração persistida e solver |
| `/allocation-result` | análise do resultado |

`WorkflowLayout` fornece header, ajuda contextual, conteúdo principal e timeline. `ScrollToTop` reposiciona a tela quando a rota muda; abrir a ajuda não muda a rota.

## Disponibilidade dos bindings

Rotas de configuração e processamento podem montar antes de `window.go.main.App` existir. `waitForWailsBindings` faz polling por até cinco segundos. Em testes, os bindings podem ser substituídos por mocks antes das chamadas.

## Eventos

Durante o solver, `App.RunAllocation` publica `allocation:progress`. `AllocationLoadingPage` assina o evento por `EventsOn` e remove o listener ao desmontar. Contagens de branches são strings porque podem exceder inteiros JavaScript seguros.

## Regra de alteração

Ao mudar uma assinatura pública de `App`:

1. atualize o tipo/função Go e seus testes;
2. regenere os bindings Wails;
3. atualize consumidores React;
4. execute `go test ./...`, build do frontend e Playwright;
5. atualize [API Wails](../contracts/wails-api.md) e o inventário gerado.
