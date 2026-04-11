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

Cada linha do mapeamento é persistida como draft com identidade estável:
- `clientId`: evita remount de inputs ao digitar
- `manualExtra`: separa extra criado pelo usuário de coluna apenas disponível
- `includeWhenUnmapped`: marca extras que devem seguir como `null` mesmo sem coluna

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

Os extras nulos usam o tipo nomeado `types.NullableString` em [`back/type/types.go`](/Users/joaobresser/Documents/Pessoal/PS/candidate_allocation/back/type/types.go). Isso mantém o JSON compatível com `string | null`, mas evita que o gerador do Wails produza um `export class string` inválido em `models.ts` durante o build.

Na revisão de duplicados, o frontend usa apenas `duplicateFields` vindos do backend. Isso evita fallback hardcoded para campos como `cpf`, `nome`, `sigla` ou `email`, então uma mudança de metadata no backend passa a refletir direto na UI.

## Schema do Banco
As tabelas dinâmicas de candidatos e avaliadores são montadas a partir do metadata do backend em [`back/db/schema.go`](/Users/joaobresser/Documents/Pessoal/PS/candidate_allocation/back/db/schema.go).

Comportamento atual:
- cria a tabela com base em `types.CandidateFields()` e `types.AvaliadorFields()`
- adiciona colunas novas automaticamente quando o schema crescer
- cria índices únicos esperados a partir do campo `Unique`
- remove índices únicos obsoletos quando um campo deixa de ser único ou sai do schema persistido

Isso faz com que mudanças como adicionar coluna, renomear campo JSON/coluna persistida ou remover unicidade passem a depender só do metadata do backend para a estrutura nova ser aplicada.

## Regras dos Mapeamentos
Em todas as telas de mapeamento:
- colunas não mapeadas continuam visíveis na seção de disponíveis
- extras manuais ficam estáveis mesmo com nome temporariamente vazio
- remover um extra é a única exclusão real; se ele estava ligado a uma coluna, essa coluna volta para disponíveis
- extras manuais sem coluna seguem para a próxima etapa com valor `null`
- a confirmação bloqueia se existir extra manual sem nome válido ou com chave conflitante

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

Regra importante da revisão:
- o card só deve renderizar campos realmente presentes na entidade; campos opcionais ausentes, como `opcoes` em avaliadores e restrições, não podem ser materializados com `undefined` no clone do estado

Os `EditableCell` também interceptam o teclado da edição para evitar navegação acidental do browser com `Backspace` quando o usuário está digitando.

## Nova Etapa de Configuração
Depois que candidatos, restrições e avaliadores são salvos, o fluxo do CLI passa a preparar uma configuração intermediária de alocação antes de qualquer execução do algoritmo.

Responsabilidades dessa etapa:
- detectar os valores únicos encontrados em todas as colunas de preferência
- mapear cada valor para um `dia + hora` reais
- coletar parâmetros editáveis com defaults
- coletar critérios soft estruturados por tipo, coluna e valores únicos, sem aplicá-los ao algoritmo ainda

Fontes principais dessa regra:
- [`back/logic/preference_logic.go`](/Users/joaobresser/Documents/Pessoal/PS/candidate_allocation/back/logic/preference_logic.go)
- [`back/logic/allocation_config_logic.go`](/Users/joaobresser/Documents/Pessoal/PS/candidate_allocation/back/logic/allocation_config_logic.go)
- [`back/workflow/import_cli.go`](/Users/joaobresser/Documents/Pessoal/PS/candidate_allocation/back/workflow/import_cli.go)

Defaults atuais da configuração:
- `gruposPorHorario = 2`
- `minPessoasPorGrupo = 4`
- `maxPessoasPorGrupo = 8`
- `avaliadoresPorGrupo = 3`
- `softCriteria = []`

### Contrato em Camadas
O contrato antigo `AllocationSetup` foi substituído por `AllocationConfiguration` em [`back/type/types.go`](/Users/joaobresser/Documents/Pessoal/PS/candidate_allocation/back/type/types.go).

Esse envelope agora separa explicitamente 4 camadas:
- `summary`: resumo humano legível para log, console e debug
- `normalized`: payload mínimo e estável pronto para o algoritmo
- `diagnostics`: origem, normalização e validação da configuração
- `result`: placeholder tipado para o resultado final da alocação

Detalhe importante:
- o algoritmo atual ainda não consome `AllocationConfiguration`
- a camada `normalized` foi preparada para isso sem misturar UI, coleta interativa e diagnóstico
- o `result.status = "not_run"` enquanto a execução continuar desativada

### Como a Configuração é Montada
As funções puras de montagem ficam em [`back/logic/allocation_config_logic.go`](/Users/joaobresser/Documents/Pessoal/PS/candidate_allocation/back/logic/allocation_config_logic.go).

Pipeline atual:
1. detectar preferências brutas dos candidatos
2. normalizar mapeamentos de preferência
3. normalizar parâmetros e critérios soft
4. validar o conjunto completo
5. gerar diagnósticos tipados
6. gerar o resumo humano final

Isso evita o problema anterior de misturar em um único objeto:
- dados coletados
- dados já normalizados
- mensagens de validação
- texto de debug

### Camadas em Detalhe
`summary` inclui:
- preferências detectadas
- preferências mapeadas
- parâmetros de alocação
- critérios soft
- valores normalizados
- observações de validação

`normalized` inclui:
- `preferenceMappings` normalizados
- `params` normalizados, incluindo `softCriteria`

`diagnostics` inclui:
- preferências detectadas originalmente
- mappings originais e normalizados
- parâmetros originais e normalizados
- diagnósticos por mapping e por critério soft
- mensagens de validação
- flag `hasErrors`

Detalhe de usabilidade no CLI:
- os prompts de dia e hora usam defaults editáveis, então `Enter` aceita os valores padrão durante testes
- os resumos dos critérios soft são escritos em linguagem natural para deixar explícita a intenção da regra
- o critério `min_value` descreve o comportamento como tentativa de manter pelo menos `N` pessoas do valor selecionado quando ele aparece no grupo
- o critério `max_value` descreve o comportamento como tentativa de limitar a `N` pessoas do valor selecionado quando ele aparece no grupo

O `RunCLI` agora monta `AllocationConfiguration` em memória e deixa a chamada final de alocação comentada/desativada nesta etapa.

Os critérios soft deixaram de ser texto livre e passaram a ser regras estruturadas com:
- tipo fixo (`min_value`, `at_least_one_each`, `balanced_distribution`, `group_together`, `max_value`)
- coluna alvo do candidato
- subconjunto de valores únicos selecionados nessa coluna
- `threshold` apenas para os tipos mínimo e máximo

As colunas elegíveis incluem todos os campos não-opção do candidato e também as chaves presentes em `Extras`. A detecção de valores únicos foi generalizada, então a mesma base atende:
- o passo 5 de preferências
- a seleção de valores para critérios soft por coluna

O contrato do backend para o futuro Wails foi preparado em [`app.go`](/Users/joaobresser/Documents/Pessoal/PS/candidate_allocation/app.go), expondo os helpers de listagem de colunas, detecção de valores únicos, normalização, validação e montagem final de `AllocationConfiguration`.

## Pontos de Atenção
- Se mudar o schema de candidatos, avaliadores ou restrições, atualize as structs e confirme os testes de metadata em [`back/type/metadata_test.go`](/Users/joaobresser/Documents/Pessoal/PS/candidate_allocation/back/type/metadata_test.go).
- Se mudar `required`, `unique` ou `duplicate` nas tags das structs, a UI de mapeamento, a revisão de duplicados e a sincronização de índices do banco devem se ajustar sem precisar de regra nova no frontend.
- Se mudar métodos exportados do `App`, mantenha `frontend/wailsjs` consistente.
- Se alterar o workflow do CLI em [`back/workflow/import_cli.go`](/Users/joaobresser/Documents/Pessoal/PS/candidate_allocation/back/workflow/import_cli.go), alinhe o fluxo correspondente da aplicação.
