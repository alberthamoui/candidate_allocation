---
id: decisions-index
title: Decisões arquiteturais
summary: Índice e regras dos Architecture Decision Records.
status: active
---

# Decisões arquiteturais

ADRs registram por que a arquitetura tomou uma direção. Depois de aceitos, são históricos e não devem ser reescritos para parecer atuais. Uma decisão posterior cria outro ADR e usa `supersedes`.

## Índice

- [ADR-0001: workflow com fonte única no backend](ADR-0001-workflow-single-source-of-truth.md)
- [ADR-0002: solver exato configurado como caminho oficial](ADR-0002-configured-exact-solver.md)

## Template

```markdown
# ADR-NNNN: Título

- Status: proposed | accepted | superseded
- Supersedes: ADR-NNNN ou nenhum

## Contexto
## Decisão
## Consequências
## Alternativas consideradas
```
