---
id: allocation-configuration
title: Configuração da alocação
summary: Defaults, mapeamento de preferências, critérios soft, normalização e validação.
status: active
sources:
  - back/logic/workflow_definition.go
  - back/logic/preference_logic.go
  - back/logic/allocation_config_logic.go
  - frontend/src/AllocationConfigPage.tsx
---

# Configuração da alocação

## Defaults oficiais

`logic.WorkflowDefinition` referencia `DefaultAllocationParams`:

| Parâmetro | Default |
|---|---:|
| grupos por horário | 2 |
| mínimo de pessoas por grupo | 4 |
| máximo de pessoas por grupo | 8 |
| avaliadores por grupo | 3 |
| critérios adicionais | vazio |

O frontend carrega esses valores via `GetWorkflowDefinition`; a CLI usa a mesma função.

## Preferências

Valores das opções de todos os candidatos são trimados, convertidos para minúsculas e têm espaços internos colapsados. `UniqueValueDetection` mantém o primeiro valor original, o normalizado e a contagem.

Cada preferência normalizada deve ter exatamente um `PreferenceScheduleMapping`. Dias aceitam nomes da semana ou datas válidas (AAAA-MM-DD / DD/MM/AAAA); horas aceitam um início ou um intervalo no mesmo dia, como 14:00-16:00. Intervalos invertidos, horas inválidas e preferências sem mapeamento são rejeitados. A realocação manual livre continua separada dessa validação da rodada.

Na CLI, dia e hora são perguntados ao usuário. No desktop, `GetPreferenceScheduleMappings` sugere dias e intervalos reconhecíveis e deixa valores ambíguos vazios para revisão. A tela envia `params.scheduleMappings` ao backend, que valida a cobertura dos dados atuais. Rodadas sem preferência informam os mesmos campos separadamente e atribuem igual prioridade a todos os horários. Preferências diferentes mapeadas para o mesmo dia/intervalo compartilham mesas e capacidade.

## Critérios soft disponíveis

| Tipo | Semântica | Threshold |
|---|---|---|
| `min_value` | quando o valor aparece no grupo, prefere pelo menos N ocorrências | obrigatório |
| `at_least_one_each` | prefere representação de todos os valores escolhidos | não usa |
| `balanced_distribution` | soma as diferenças absolutas entre cada grupo e a mediana das quantidades | não usa |
| `group_together` | penaliza cada pessoa escolhida que esteja em uma mesa misturada com outros valores | não usa |
| `max_value` | prefere no máximo N ocorrências por grupo | obrigatório |

Critérios são soft: adicionam penalidade, mas não tornam uma solução estruturalmente inválida.

## Colunas elegíveis

A lógica de domínio pode listar todos os campos centrais do candidato, exceto `opcoes`/`extras`, além das chaves extras encontradas. Na tela desktop, `GetCriteriaOptions` expõe valores persistidos de `semestre`, `curso` e `extras.*`.

Valores selecionados são normalizados, vazios removidos e duplicados eliminados. A validação confirma tipo, coluna, valores existentes e threshold.

## Contrato em camadas

`AllocationConfiguration` contém:

- `summary`: texto humano para inspeção;
- `normalized`: mappings e parâmetros prontos para o adapter;
- `diagnostics`: original, normalizado, mensagens e flag de erro;
- `result`: status de execução, inicialmente `not_run`.

A configuração é devolvida mesmo quando há erro, permitindo inspecionar diagnósticos; a chamada também retorna o erro.

## Contagem combinatória

As funções de quantidade estimam distribuições de pessoas distintas em grupos, respeitando mínimo/máximo. A versão `AcrossSchedules` permite horários vazios e considera todos os slots. É uma estimativa de combinações de capacidade, não a contagem exata do espaço de busca após preferências e restrições.

## Tela

`AllocationConfigPage` mostra capacidade mínima/máxima por horário, edita quatro parâmetros e permite múltiplos critérios. Ao iniciar, valida a configuração no backend e mostra os erros na própria tela. Somente uma configuração válida segue para a busca.
