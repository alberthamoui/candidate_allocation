---
id: documentation-update
title: Atualização da documentação
summary: Contrato para a ferramenta/LLM manter os documentos sincronizados com commits.
status: active
sources:
  - docs/manifest.yaml
  - AGENTS.md
---

# Atualização da documentação

## Objetivo

Depois de um conjunto de commits, uma ferramenta deve atualizar somente os documentos afetados, com evidência no código e nos testes. `docs/manifest.yaml` faz o roteamento; Markdown continua sendo a documentação consumida.

## Entrada esperada

- referência base ou lista de commits;
- `git diff --name-only` e diff completo;
- working tree atual;
- `docs/manifest.yaml`;
- documentos selecionados e suas fontes/testes.

## Algoritmo

1. liste arquivos alterados;
2. remova paths ignorados pelo manifesto;
3. encontre documentos cujos `sources`, `tests` ou `consumers` combinam com os arquivos;
4. leia os arquivos alterados e as fontes canônicas do documento;
5. leia testes relacionados para confirmar semântica e casos de erro;
6. atualize fatos, fluxos, contratos, limitações e inventários afetados;
7. revise documentos `related` quando a mudança atravessar camadas;
8. valide paths, links, YAML e comandos;
9. execute testes proporcionais e produza relatório de atualização;
10. se nenhum documento cobrir uma mudança relevante, proponha um novo mapeamento em vez de ignorá-la.

## Regras do prompt da IA

- Não use mensagem de commit como única evidência.
- Não invente intenção; descreva o comportamento implementado.
- Não transforme documentação em fonte duplicada de defaults/regras.
- Use paths relativos e nomes de símbolos, não números de linha.
- Preserve IDs e títulos salvo mudança conceitual real.
- Não reescreva ADR aceito; crie um ADR `supersedes`.
- Marque `needs-review` quando código e testes não forem suficientes.
- Não documente artefatos gerados como se fossem fontes.
- Atualize `known-limitations.md` quando encontrar divergência, dívida ou falha silenciosa comprovável.
- Atualize inventários em `generated/` mecanicamente; não coloque explicação arquitetural neles.

## Saída esperada

```text
Documentos atualizados:
- docs/features/...

Documentos revisados sem mudança:
- docs/contracts/...

Cobertura ausente:
- arquivo alterado -> sugestão de novo documento/mapeamento

Validações:
- links
- testes
- lint

Needs review:
- dúvidas que exigem decisão humana
```

## Quando criar documento novo

Crie quando surgir um novo domínio, fluxo de usuário, contrato público, processo operacional ou decisão arquitetural. Não crie um documento por arquivo de código; documente responsabilidades coesas.

## Revisão humana

Pessoas devem conseguir revisar o diff como qualquer código. A IA pode escrever e atualizar, mas mudanças em ADR, contrato público, fonte de verdade ou limitação crítica merecem atenção explícita no relatório.
