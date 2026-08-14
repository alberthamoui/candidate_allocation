---
id: generated-wails-bindings
title: Inventário dos bindings Wails
summary: Lista mecânica dos métodos expostos ao frontend.
status: generated
sources:
  - app.go
  - frontend/wailsjs/go/main/App.d.ts
---

# Inventário dos bindings Wails

Este arquivo é um inventário; a explicação canônica está em [API Wails](../contracts/wails-api.md). Regenere/revise quando `App.d.ts` mudar.

## Importação e persistência

- `SuggestMapping(bytes, optionCount) -> MappingItem[]`
- `SuggestMappingAvaliador() -> MappingItem[]`
- `SuggestMappingRestricao() -> MappingItem[]`
- `GetCandidateMappingFieldInfos() -> MappingFieldInfo[]`
- `GetAvaliadorMappingFieldInfos() -> MappingFieldInfo[]`
- `GetRestricaoMappingFieldInfos() -> MappingFieldInfo[]`
- `BuildUsuariosWithMapping(items) -> UsuariosResponse`
- `BuildAvaliadoresWithMapping(items) -> AvaliadoresResponse`
- `BuildRestricoesWithMapping(items) -> Restricao[]`
- `SaveUsuariosFromMaps(maps) -> void`
- `SaveAvaliadoresFromMaps(maps) -> void`
- `SaveRestricoesFromMaps(maps) -> void`

## Workflow e configuração

- `GetWorkflowDefinition() -> WorkflowDefinition`
- `DefaultAllocationParams() -> AllocationParams`
- `GetSoftCriterionOptions() -> SoftCriterionOption[]`
- `DetectUniquePreferenceValues(candidates) -> UniqueValueDetection[]`
- `ListCandidateCriterionColumns(candidates) -> CandidateCriterionColumn[]`
- `DetectUniqueCandidateColumnValues(candidates, key) -> UniqueValueDetection[]`
- `NormalizePreferenceScheduleMappings(mappings) -> PreferenceScheduleMapping[]`
- `NormalizeSoftCriteria(criteria) -> SoftCriterion[]`
- `ValidatePreferenceScheduleMappings(mappings) -> void`
- `ValidateSoftCriteria(criteria, candidates) -> void`
- `ValidateAllocationParams(params, candidates) -> void`
- `BuildAllocationConfiguration(detections, mappings, params, candidates) -> AllocationConfiguration`
- `BuildAllocationConfigurationFromDatabase(params) -> AllocationConfiguration`
- `CountPossibleAllocationQuantities(params, people) -> number`
- `CountPossibleAllocationQuantitiesAcrossSchedules(params, people, schedules) -> number`

## Execução e UI

- `GetCriteriaOptions() -> Record<string, string[]>`
- `RunAllocation(config) -> UIAllocationResult`
- `Greet(name) -> string`

Evento fora de `App.d.ts`: `allocation:progress` com percent, branches como strings, nós e subárvores podadas.
