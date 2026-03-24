# Documentação Técnica - Candidate Allocation

## Visão Geral
O fluxo de importação trabalha com três entidades vindas do Excel:
- candidatos
- restrições
- avaliadores

Cada entidade passa por duas etapas separadas:
1. mapeamento de colunas
2. revisão dos dados construídos antes do salvamento

Enquanto o usuário ainda não salvou a etapa atual, ele pode voltar da revisão para o mapeamento e ajustar as colunas. Depois do salvamento, o fluxo segue para a próxima entidade.

## Fluxo Atual do Frontend
Ordem das telas:
1. mapeamento de candidatos
2. revisão de candidatos
3. mapeamento de restrições
4. revisão de restrições
5. mapeamento de avaliadores
6. revisão de avaliadores
7. sucesso

O estado dos mapeamentos fica no `Root` em [`frontend/src/main.tsx`](/Users/joaobresser/Documents/Pessoal/PS/candidate_allocation/frontend/src/main.tsx), então quando a revisão volta para o mapeamento o usuário reencontra exatamente a última configuração usada naquela etapa.

## Metadata de Mapeamento
Os badges exibidos no mapeamento vêm do backend, não de constantes duplicadas no frontend.

Fonte da verdade:
- [`back/type/types.go`](/Users/joaobresser/Documents/Pessoal/PS/candidate_allocation/back/type/types.go)
- [`back/type/metadata.go`](/Users/joaobresser/Documents/Pessoal/PS/candidate_allocation/back/type/metadata.go)

Metadados expostos para a UI:
- `required`: o campo é obrigatório para o domínio
- `unique`: o campo participa da reconstrução por identificador único
- `duplicate`: o campo é usado pela lógica de duplicidade

Os métodos do Wails que entregam isso ao frontend ficam em [`app.go`](/Users/joaobresser/Documents/Pessoal/PS/candidate_allocation/app.go) e os bindings gerados/espelhados ficam em `frontend/wailsjs`.

## Regras dos Mapeamentos
Em todas as telas de mapeamento:
- colunas não mapeadas continuam visíveis na seção de disponíveis
- remover um extra limpa a variável e devolve a coluna para disponível
- apenas itens com `indice >= 0` e `variavel != ""` são enviados ao backend

Diferença importante:
- candidatos e avaliadores aceitam campos extras
- restrições não aceitam extras; colunas fora do schema ficam apenas como disponíveis

## Revisão e Salvamento
O componente compartilhado de revisão é [`frontend/src/EntityVerificationView.tsx`](/Users/joaobresser/Documents/Pessoal/PS/candidate_allocation/frontend/src/EntityVerificationView.tsx).

Ele é usado para:
- editar campos inline
- resolver duplicados
- salvar apenas a entidade da etapa atual
- voltar ao mapeamento da mesma etapa antes do save

As restrições usam esse mesmo componente com `allowExtras={false}`.

## Pontos de Atenção
- Se mudar o schema de candidatos, avaliadores ou restrições, atualize as structs e confirme os testes de metadata em [`back/type/metadata_test.go`](/Users/joaobresser/Documents/Pessoal/PS/candidate_allocation/back/type/metadata_test.go).
- Se mudar métodos exportados do `App`, mantenha `frontend/wailsjs` consistente.
- Se alterar o workflow do CLI em [`back/workflow/import_cli.go`](/Users/joaobresser/Documents/Pessoal/PS/candidate_allocation/back/workflow/import_cli.go), alinhe o fluxo correspondente da aplicação.
