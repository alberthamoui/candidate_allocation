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

## Frontend Executivo

O frontend agora usa um shell visual compartilhado para todas as rotas do wizard desktop.

Peças centrais:

- [`frontend/src/workflowShell.tsx`](/Users/joaobresser/Documents/Pessoal/PS/candidate_allocation/frontend/src/workflowShell.tsx): layout global, barra lateral com etapas, cabeçalho premium, drawer fixo de ajuda e componentes visuais reutilizáveis
- [`frontend/src/workflowMeta.ts`](/Users/joaobresser/Documents/Pessoal/PS/candidate_allocation/frontend/src/workflowMeta.ts): fonte única do conteúdo contextual por rota, incluindo título, resumo e ajuda da página
- [`frontend/src/index.css`](/Users/joaobresser/Documents/Pessoal/PS/candidate_allocation/frontend/src/index.css): tokens visuais globais com tipografia, paleta, sombras, superfícies e classes base do design system

Direção adotada:

- estética clara e institucional com tom “luxuoso executivo”
- tipografia serif para títulos e sans humanista para corpo
- paleta baseada em marfim quente, azul-ardósia e acento bronze
- mesma linguagem visual em importação, mapeamento, revisão, configuração, loading e resultado

## Ajuda Contextual

O padrão de ajuda passou a ser híbrido:

- botão fixo `Ajuda da página` sempre no mesmo lugar, abrindo drawer lateral
- ícones `?` inline apenas em elementos com maior chance de gerar dúvida

O conteúdo da ajuda é estático no frontend e mapeado por rota em `workflowMeta.ts`.

Isso reduz duplicação e permite manter:

- objetivo da página
- instruções de uso
- impacto das decisões/configurações
- glossário curto dos elementos mais importantes

## Estrutura das Telas

As telas foram reorganizadas sem alterar contratos com o backend:

- home: hero de importação e bloco técnico do `Greet` rebaixado para utilitário
- mapeamento: três áreas visuais estáveis, com campos principais, extras e colunas disponíveis
- revisão: painel operacional com métricas, duplicados priorizados e barra fixa de salvamento
- sucesso: checkpoint institucional antes da configuração
- configuração: seções separadas para parâmetros base, critérios soft e leitura de impacto
- loading: tela de processamento coerente com o shell global
- resultado: dashboard executivo com filtros, grupos por horário e painel lateral de não alocados

## Bindings do Wails em Rotas Diretas

As telas de configuração e processamento agora usam [`frontend/src/wailsReady.ts`](/Users/joaobresser/Documents/Pessoal/PS/candidate_allocation/frontend/src/wailsReady.ts) para esperar os bindings do Wails antes de chamar métodos do backend.

Isso evita crash quando o usuário ou um teste Playwright abre diretamente rotas como:

- `/allocation-config`
- `/allocation-loading`

Sem essa espera, chamadas imediatas para `window.go.main.App.*` podem falhar antes do runtime terminar de inicializar no browser de desenvolvimento.

## UI Testing

Os testes de UI continuam em `tools/ui-testing`, mas agora com alguns ajustes importantes para convivência com o modo Wails dev:

- a suíte usa `data-testid` nos pontos críticos de navegação para reduzir fragilidade visual
- `playwright.config.ts` grava artefatos temporários em `/tmp/candidate-allocation-playwright`
- os diretórios `test-results/` e `tools/ui-testing/test-results/` possuem `go.mod` local para impedir que artefatos do Playwright contaminem o `go mod tidy` executado pelo `wails dev`

Isso evita que nomes de diretório gerados por falha de teste virem pseudo-pacotes Go inválidos dentro do módulo principal.

O inspector visual agora só gera screenshot quando o parâmetro `screenshot=true` é passado. Por padrão ele salva apenas HTML e árvore de acessibilidade para reduzir o consumo da cota de screenshots na sessão.

O shell visual também ganhou um ajuste de alinhamento no header fixo e no botão de ajuda da página para reduzir o aspecto desalinhado nas rotas principais.

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

Depois do resumo da configuração, o CLI também imprime as quantidades possíveis de alocação calculadas a partir dos parâmetros e do total de candidatos válidos.

Os critérios soft deixaram de ser texto livre e passaram a ser regras estruturadas com:

- tipo fixo (`min_value`, `at_least_one_each`, `balanced_distribution`, `group_together`, `max_value`)
- coluna alvo do candidato
- subconjunto de valores únicos selecionados nessa coluna
- `threshold` apenas para os tipos mínimo e máximo

As colunas elegíveis incluem todos os campos não-opção do candidato e também as chaves presentes em `Extras`. A detecção de valores únicos foi generalizada, então a mesma base atende:

- o passo 5 de preferências
- a seleção de valores para critérios soft por coluna

O contrato do backend para o futuro Wails foi preparado em [`app.go`](/Users/joaobresser/Documents/Pessoal/PS/candidate_allocation/app.go), expondo os helpers de listagem de colunas, detecção de valores únicos, normalização, validação e montagem final de `AllocationConfiguration`.

## Base do Solver

O pacote [`back/allocation`](/Users/joaobresser/Documents/Pessoal/PS/candidate_allocation/back/allocation) agora contém a base determinística do solver separada da coleta do CLI e da persistência em banco.

### Contrato do solver

O contrato único de entrada do núcleo é `types.AllocationProblem`.

Ele representa o problema já pronto para decisão e não replica o contrato de UI/CLI:

- `Candidates`: candidatos com `PreferredGroupIDs`, `Attributes` normalizados e restrições por avaliador
- `Groups`: grupos materializados, com `EvaluatorIDs`, capacidade mínima e máxima
- `HardRestrictions`: flags explícitas das regras obrigatórias da fase 1
- `SoftRules`: penalidades de preferência, penalidade de `PrefiroNao` e critérios soft

### Contagem de Quantidades

O pacote [`back/logic`](/Users/joaobresser/Documents/Pessoal/PS/candidate_allocation/back/logic) expõe helpers para enumerar e contar distribuições possíveis de pessoas entre grupos.

Esses helpers trabalham com `types.AllocationParams` e consideram:

- `GruposPorHorario` como quantidade de grupos por horario
- `MinPessoasPorGrupo` e `MaxPessoasPorGrupo` como limites por grupo
- `totalPeople` como total de pessoas disponíveis para distribuir
- `scheduleCount` quando a mesma configuração precisa ser repetida em varios horarios

Comportamento atualizado para contagem precisa:

- Grupos no **mesmo horário** são considerados indistinguíveis (sem rótulo). Trocar grupos de lugar no mesmo horário representa a mesma opção de alocação.
- Horários diferentes são distinguíveis. Trocar a alocação de pessoas entre horários diferentes representa opções distintas.
- `CountPossibleAllocationQuantities` calcula as distribuições válidas combinatórias considerando grupos indistinguíveis dentro de um único horário usando programação dinâmica.
- `CountPossibleAllocationQuantitiesAcrossSchedules` calcula a distribuição das pessoas sobre os vários horários (distinguíveis), onde cada horário organiza seus candidatos em grupos indistinguíveis, permitindo que existam grupos vazios se não houver candidatos suficientes naquele horário.

Exemplo prático corrigido:

- 4 pessoas sendo distribuídas em 2 horários, com 2 grupos de tamanho 2 por horário, geram **12 alocações diferentes** ao invés de 36, pois as permutações idênticas dentro do mesmo horário foram removidas.

Esses helpers ainda não levam em conta regras de preferência ou restrições de candidato; eles operam somente sobre os parâmetros agregados da configuração.

No CLI, `CountPossibleAllocationQuantitiesAcrossSchedules` passou a ser usado para mostrar o total final. O resumo humano consolidado foi removido da saída intermediária para reduzir ruído.

Quando houver mais candidatos do que a capacidade total dos grupos, o contador considera todas as formas de escolher um subconjunto de candidatos e deixa o excedente de fora.

Diferença importante:

- `AllocationConfiguration` continua sendo o contrato de coleta, validação e debug
- `AllocationProblem` é o contrato de decisão do solver

### Camada adaptadora

A transformação entre configuração normalizada e contrato do solver fica em [`back/allocation/problem_builder.go`](/Users/joaobresser/Documents/Pessoal/PS/candidate_allocation/back/allocation/problem_builder.go).

Responsabilidades do builder:

- ler `config.Normalized`
- materializar grupos de solver a partir de `PreferenceMappings`
- distribuir avaliadores de forma determinística entre grupos
- transformar preferências dos candidatos em `PreferredGroupIDs`
- copiar atributos normalizados usados pelos critérios soft
- converter `NaoPosso` em `ForbiddenEvaluatorIDs`
- converter `PrefiroNao` em `AvoidEvaluatorIDs`

Nesta fase, a distribuição de avaliadores é determinística e serve como base verificável do núcleo. A busca exata e estratégias mais avançadas ficam para fases futuras.

### Hard constraints

As regras hard ficam em [`back/allocation/hard_constraints.go`](/Users/joaobresser/Documents/Pessoal/PS/candidate_allocation/back/allocation/hard_constraints.go).

API principal:

- `CheckHardConstraints`
- `IsStateViable`
- `IsCompleteState`

Regras cobertas na fase 1:

- candidato não pode aparecer duas vezes
- IDs de candidato e grupo precisam existir
- candidato só pode ser alocado em grupo permitido por preferência
- grupo não pode exceder `MaxCandidates`
- candidato não pode cair com avaliador em `ForbiddenEvaluatorIDs`
- estado completo exige todos os candidatos alocados
- estado completo exige que grupos usados respeitem `MinCandidates`

As violações são retornadas de forma estruturada em `HardConstraintViolation`, com `Code` estável para teste e depuração.

### Soft scorer

O scorer fica em [`back/allocation/soft_score.go`](/Users/joaobresser/Documents/Pessoal/PS/candidate_allocation/back/allocation/soft_score.go).

Convenção da fase 1:

- menor penalidade é melhor
- o scorer é determinístico
- a saída inclui breakdown explicável por componente

Componentes atuais:

- rank da preferência do candidato
- penalidade por `PrefiroNao` (`AvoidEvaluatorIDs`)
- penalidades dos critérios soft:
  - `min_value`
  - `max_value`
  - `at_least_one_each`
  - `balanced_distribution`
  - `group_together`

### Solver exato

O solver final desta etapa fica em [`back/allocation/exact_solver.go`](/Users/joaobresser/Documents/Pessoal/PS/candidate_allocation/back/allocation/exact_solver.go).

API principal:

- `SolveAllocation(problem, options)`
- `NormalizeSolverOptions(options)`
- `EstimateLowerBound(problem, state)`

Contrato fechado desta etapa:

- entrada unica: `types.AllocationProblem`
- saida unica: `types.SolverResult`
- a coleta de configuracao continua separada em `AllocationConfiguration`
- o solver nao acessa banco, CLI nem frontend

### Fluxo de execucao do solver

Fluxo completo da busca:

1. recebe `AllocationProblem` ja normalizado
2. precomputa visoes deterministicas de candidatos, grupos e penalidades fixas por candidato/grupo
3. gera a fronteira inicial ate `ParallelDepth` usando apenas poda hard
4. paraleliza apenas essas subarvores externas quando `WorkerCount > 1`
5. cada subarvore segue com busca recursiva sequencial e deterministica

Em cada passo recursivo:

- escolhe a proxima pessoa ainda nao alocada com menor numero de grupos viaveis
- tenta os grupos em ordem deterministica, priorizando menor penalidade imediata
- aplica a alocacao parcial
- roda a poda hard com `IsStateViable` e com checagens de inviabilidade futura hard
- se o estado continuar viavel, calcula um lower bound otimista do soft score
- se esse bound ja for pior ou igual ao melhor score daquela subarvore, corta o ramo
- se o estado estiver completo, calcula o score real com `ScoreAllocation` e tenta atualizar o incumbent da subarvore

No final:

- consolida o melhor resultado de todas as subarvores em ordem deterministica
- retorna a melhor solucao valida com score e metricas
- ou retorna `status = infeasible` com `HardViolations` e `RejectionReason`

### Quantidades possiveis de alocacao

Em [`back/logic/allocation_quantities.go`](/Users/joaobresser/Documents/Pessoal/PS/candidate_allocation/back/logic/allocation_quantities.go) existe uma rotina para enumerar todas as distribuicoes possiveis de quantidade por grupo a partir de `AllocationParams` e do total de candidatos.

Uso esperado:

- entrada: `AllocationParams` + numero total de candidatos
- saida: todas as combinacoes viaveis de pessoas por grupo, respeitando `GruposPorHorario`, `MinPessoasPorGrupo` e `MaxPessoasPorGrupo`
- caso útil para teste: 10 candidatos com 2 grupos travados em 5 e 5 produzem apenas uma distribuicao possivel

### Como a poda funciona

O solver usa branch and bound com duas podas seguras:

- `hard prune`: elimina estados que ja violam regra obrigatoria ou que nao conseguem mais completar as regras hard futuras
- `bound prune`: elimina estados cujo menor score ainda possivel ja e pior ou igual ao incumbent da subarvore

Por que a poda por bound e segura:

- o lower bound soma apenas penalidades irreversiveis do estado parcial
- preferencia e `avoid_evaluator` ja ficam fixas assim que o candidato e alocado
- para criterios soft, so entram no bound componentes que nao podem melhorar depois, como excesso de `max_value` e espalhamento ja consumado de `group_together`
- criterios que ainda podem melhorar ficam com contribuicao `0` no bound, preservando admissibilidade

Isso garante que o solver nunca corta um ramo que ainda poderia produzir uma solucao melhor que o incumbent.

### Metricas e debug

`SolverResult` agora expoe:

- `Status`, `Assignments`, `Score`, `HardViolations` e `RejectionReason`
- `Metrics` com `NodesVisited`, `CompleteStates`, `NodesPrunedByHard`, `NodesPrunedByBound`, `BestUpdates` e `ParallelTasks`
- `DebugNotes` com eventos resumidos de poda e atualizacao da melhor solucao

Detalhe importante para testes:

- o resultado final da alocacao e deterministico entre execucao sequencial e paralela para o mesmo input fixo
- as metricas podem variar entre modos porque as subarvores paralelas nao compartilham poda por incumbent em tempo real

## Pontos de Atenção

- Se mudar o schema de candidatos, avaliadores ou restrições, atualize as structs e confirme os testes de metadata em [`back/type/metadata_test.go`](/Users/joaobresser/Documents/Pessoal/PS/candidate_allocation/back/type/metadata_test.go).
- Se mudar `required`, `unique` ou `duplicate` nas tags das structs, a UI de mapeamento, a revisão de duplicados e a sincronização de índices do banco devem se ajustar sem precisar de regra nova no frontend.
- Se mudar métodos exportados do `App`, mantenha `frontend/wailsjs` consistente.
- Se alterar o workflow do CLI em [`back/workflow/import_cli.go`](/Users/joaobresser/Documents/Pessoal/PS/candidate_allocation/back/workflow/import_cli.go), alinhe o fluxo correspondente da aplicação.

## Ferramentas de Qualidade (Linter)

O repositório agora possui uma configuração explícita e visível dos linters utilizados, garantindo um padrão de código consistente.

A configuração principal fica no `.golangci.yml`, com a listagem de todos os linters habilitados (como `unused`, `gosec`, `revive`, entre outros).

**Analyzer Customizado (Unexported)**
Foi criado um analyzer específico localizado em `tools/unexported/main.go`. A responsabilidade dele é buscar funções exportadas (com inicial maiúscula) que não estão sendo chamadas fora do seu pacote de origem e sugerir que elas se tornem privadas (unexported).
- **Como funciona:** O script varre o módulo inteiro usando `go/packages`, constrói um índice de definições e referências e aponta métodos "vazando" a não ser que estejam em uma allowlist (como os métodos do `App` usados pelo Wails).
- **Como executar:** Ao invés de executar apenas `golangci-lint run`, é recomendado utilizar `make lint`, que cuidará de buildar a ferramenta e executar ambos os checks (o golangci-lint padrão e o nosso analyzer customizado).

### Limpeza Recente

- foram removidos helpers de processo de teste que não eram usados
- o uso de `rand.Seed` foi substituído por geradores locais
- os fluxos de escrita/leitura passaram a tratar `Close` explicitamente
- funções e tipos exportados passaram a ter documentação para manter o `revive` limpo
- `beforeClose` limpa o estado transitório do app e `shutdown` também remove o sentinel temporário usado pelo smoke test do Wails

## Alocação UI

Foi adicionada uma interface de carregamento `AllocationLoadingPage.tsx` e uma tela de resultados `AllocationResultPage.tsx`. O backend foi atualizado com uma função `RunAllocation` em `app.go` para fazer a ponte com o Wails. A interface mostra de forma cronológica os horários e separa claramente os grupos, colocando os avaliadores no final das listas com destaque visual. Também contém áreas para filtros de critérios (Soft/Hard) e lista de candidatos não alocados.

- Adicionado `AllocationConfigPage.tsx` para definir grupos, mínimo/máximo de pessoas e critérios soft. Integrei com o `exact_solver.go` substituindo o antigo simulador.

- Expandido `AllocationConfigPage.tsx` para conter um construtor de Soft Criteria (Criar, Editar, Deletar), passando tipos nativos suportados pelo Go (ex: min_value, balanced_distribution, etc) e a coluna de filtragem com threshold para o exact solver.

- Adicionado sistema de listagem dinâmica das opções e colunas nos soft criterias a partir do BD.
- Adicionado barra de pesquisa livre na tela de resultado.
