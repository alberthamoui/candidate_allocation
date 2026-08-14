---
id: documentation-updater-prompt
title: Prompt do atualizador de documentação
summary: Instruções executadas por codex exec para sincronizar documentação e código.
status: active
---

# Tarefa: sincronizar a documentação com o código

Você está em uma execução não interativa dedicada exclusivamente à documentação deste repositório. O contexto anterior foi calculado pelo comando `scripts/update-docs`.

## Objetivo

Analise todo o intervalo entre a base documentada e o hash de código alvo. Descubra o comportamento efetivamente alterado e atualize a documentação canônica para refletir o estado presente no hash alvo.

## Procedimento obrigatório

1. Leia `AGENTS.md`, `docs/README.md`, `docs/manifest.yaml` e `docs/operations/documentation-update.md` por completo. A seção “Contrato editorial” de `documentation-update.md` é a fonte exclusiva para estilo, estrutura e completude da documentação; não espere essas regras em `AGENTS.md`.
2. Use `jj --ignore-working-copy log`, `jj --ignore-working-copy diff` e os hashes fornecidos no contexto para inspecionar todos os commits do intervalo. O wrapper já fez o snapshot; não permita que leituras do agente tentem escrever em `.git`. Não confie apenas nas mensagens dos commits.
3. Cruze cada arquivo relevante com `sources`, `tests`, `consumers` e `related` do manifesto.
4. Leia o código atual, os testes e os documentos selecionados antes de editar.
5. Atualize somente documentação: `docs/**`, `README.md`, `Docs.md` e, apenas se as regras permanentes realmente mudaram, `AGENTS.md`.
6. Não edite `docs/documentation-state.yaml`; o comando atualiza esse arquivo somente depois que esta execução terminar com sucesso.
7. Não execute comandos mutáveis do Jujutsu, como `jj new`, `jj describe`, `jj commit`, `jj rebase` ou `jj abandon`.
8. Preserve IDs de frontmatter. Não reescreva ADR aceito; crie um novo ADR com `supersedes` quando necessário.
9. Registre divergências comprovadas em `docs/reference/known-limitations.md`. Marque `needs-review` quando o código não for evidência suficiente.
10. Se aparecer uma responsabilidade sem documento, crie o documento apropriado e adicione seu roteamento ao manifesto.
11. Antes de encerrar, aplique o checklist de completude do contrato editorial a cada documento alterado.

## Validação

- valide o YAML do manifesto e do estado;
- valide frontmatter, IDs únicos e links Markdown locais;
- execute testes ou lint proporcionais quando uma afirmação operacional depender deles;
- confirme com uma leitura do working tree ou `git diff` que nenhum arquivo fora do escopo documental foi alterado; o wrapper fará a validação definitiva com `jj` depois que o agente sair.

## Resposta final

Informe de forma concisa:

- documentos atualizados;
- documentos revisados sem mudança;
- validações executadas e resultado;
- limitações ou itens `needs-review`;
- lacunas de cobertura adicionadas ao manifesto.
