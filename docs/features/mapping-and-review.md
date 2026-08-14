---
id: mapping-and-review
title: Mapeamento e revisão
summary: Edição dos mappings, extras, validação, duplicidade e salvamento.
status: active
sources:
  - back/type/metadata.go
  - back/logic/logic.go
  - frontend/src/MappingEditorPage.tsx
  - frontend/src/EntityVerificationView.tsx
---

# Mapeamento e revisão

## Metadata dos campos

Badges e regras estruturais vêm das tags das structs:

- `db:"required"`: obrigatório no schema;
- `db:"unique"`: índice único persistente;
- `app:"duplicate"`: participa da detecção de duplicados;
- `db:"-"`: não vira coluna dinâmica.

`CandidateMappingFieldInfos`, `AvaliadorMappingFieldInfos` e `RestricaoMappingFieldInfos` convertem essa metadata para a UI.

## Drafts

`MappingDraft` estende o contrato persistível com:

- `clientId`: identidade estável de interface;
- `manualExtra`: diferencia extras criados manualmente;
- `includeWhenUnmapped`: preserva um extra sem coluna como `null`.

Esses campos de UI não alteram o domínio; `buildPayload` devolve somente `MappingItem`.

## Editor

O editor permite trocar colunas por drag-and-drop, remapear, criar/remover extras e fazer auto-scroll durante arraste. Campos extras são normalizados para validação de chave: acentos são removidos, o texto fica minúsculo e separadores viram `_`.

Antes de confirmar, a UI bloqueia:

- extra manual sem nome;
- extra com chave igual à de um campo central;
- dois extras que geram a mesma chave normalizada;
- colisões de destinos relevantes.

Restrições usam `allowExtraFields=false`.

## Revisão compartilhada

`EntityVerificationView` atende candidatos, restrições e avaliadores. Ele:

- mantém cópia editável das entidades;
- permite edição inline de campos;
- mostra erros associados ao registro;
- permite aceitar um registro de um grupo duplicado ou rejeitar alternativas;
- impede salvamento enquanto houver duplicidade pendente;
- volta ao mapping sem descartar o draft mantido por `Root`;
- remove `undefined` antes do save e trata extras conforme `allowExtras`.

As chaves usadas para comparar duplicados vêm de `duplicateFields` do backend, não de listas hardcoded no React.

## Sequência do desktop

| Revisão | Método de save | Próxima rota |
|---|---|---|
| candidatos | `SaveUsuariosFromMaps` | `/mappingRestricoes` |
| restrições | `SaveRestricoesFromMaps` | `/mappingAvaliadores` |
| avaliadores | `SaveAvaliadoresFromMaps` | `/success` |

## Persistência

O payload genérico é serializado para JSON e desserializado na struct Go correspondente. `Save` sincroniza o schema, abre o banco e delega as inserções. Inserções individuais que falham dentro de `fillDb` são atualmente registradas no stdout e podem não propagar erro ao frontend; isso está registrado como limitação.

## Testes essenciais

- metadata acompanha o schema atual;
- mappings suportam extras e índices ignorados;
- extras sem coluna viram `null` quando solicitado;
- restrições rejeitam extras;
- Playwright cobre conflitos de extras, duplicidade e edição inline.
