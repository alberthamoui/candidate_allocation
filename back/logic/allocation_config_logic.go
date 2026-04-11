package logic

import (
	"fmt"
	"strings"

	types "candidate_alocator/back/type"
)

// DefaultAllocationParams devolve a configuração base usada pelo CLI e pelo
// futuro fluxo do Wails.
func DefaultAllocationParams() types.AllocationParams {
	return types.AllocationParams{
		GruposPorHorario:    2,
		MinPessoasPorGrupo:  4,
		MaxPessoasPorGrupo:  8,
		AvaliadoresPorGrupo: 3,
		SoftCriteria:        []types.SoftCriterion{},
	}
}

func normalizeSoftCriterionType(raw types.SoftCriterionType) types.SoftCriterionType {
	return types.SoftCriterionType(strings.TrimSpace(strings.ToLower(string(raw))))
}

// NormalizePreferenceScheduleMappings padroniza o valor da preferência e remove
// espaços redundantes de dia e hora.
func NormalizePreferenceScheduleMappings(mappings []types.PreferenceScheduleMapping) []types.PreferenceScheduleMapping {
	normalized := make([]types.PreferenceScheduleMapping, 0, len(mappings))
	for _, mapping := range mappings {
		normalized = append(normalized, types.PreferenceScheduleMapping{
			ValorPreferencia: normalizeDetectedValue(mapping.ValorPreferencia),
			Dia:              strings.Join(strings.Fields(strings.TrimSpace(mapping.Dia)), " "),
			Hora:             strings.Join(strings.Fields(strings.TrimSpace(mapping.Hora)), " "),
		})
	}
	return normalized
}

// NormalizeSoftCriteria padroniza tipo, coluna e valores selecionados para uso
// consistente entre CLI, frontend e futuras regras de alocação.
func NormalizeSoftCriteria(criteria []types.SoftCriterion) []types.SoftCriterion {
	normalized := make([]types.SoftCriterion, 0, len(criteria))

	for _, criterion := range criteria {
		values := make([]string, 0, len(criterion.SelectedValues))
		seen := make(map[string]struct{}, len(criterion.SelectedValues))
		for _, rawValue := range criterion.SelectedValues {
			value := normalizeDetectedValue(rawValue)
			if value == "" {
				continue
			}
			if _, ok := seen[value]; ok {
				continue
			}
			seen[value] = struct{}{}
			values = append(values, value)
		}

		normalized = append(normalized, types.SoftCriterion{
			Type:           normalizeSoftCriterionType(criterion.Type),
			ColumnKey:      strings.TrimSpace(criterion.ColumnKey),
			SelectedValues: values,
			Threshold:      criterion.Threshold,
		})
	}

	return normalized
}

// NormalizeAllocationParams padroniza a estrutura de parâmetros para o contrato
// estável consumido pelo algoritmo.
func NormalizeAllocationParams(params types.AllocationParams) types.AllocationParams {
	normalized := params
	normalized.SoftCriteria = NormalizeSoftCriteria(params.SoftCriteria)
	return normalized
}

func isSupportedSoftCriterionType(value types.SoftCriterionType) bool {
	switch value {
	case types.SoftCriterionMinValue,
		types.SoftCriterionAtLeastOneEach,
		types.SoftCriterionBalancedDistribution,
		types.SoftCriterionGroupTogether,
		types.SoftCriterionMaxValue:
		return true
	default:
		return false
	}
}

// ValidateSoftCriteria garante que cada regra usa uma coluna válida, aponta
// para valores existentes e respeita a semântica do tipo selecionado.
func ValidateSoftCriteria(criteria []types.SoftCriterion, candidatos []types.Candidato) error {
	columns := ListCandidateCriterionColumns(candidatos)
	columnMap := make(map[string]types.CandidateCriterionColumn, len(columns))
	for _, column := range columns {
		columnMap[column.Key] = column
	}

	for _, criterion := range NormalizeSoftCriteria(criteria) {
		if !isSupportedSoftCriterionType(criterion.Type) {
			return fmt.Errorf("tipo de criterio soft invalido: %q", criterion.Type)
		}
		if criterion.ColumnKey == "" {
			return fmt.Errorf("columnKey do criterio soft nao pode ser vazio")
		}
		if _, ok := columnMap[criterion.ColumnKey]; !ok {
			return fmt.Errorf("columnKey %q nao e elegivel para criterio soft", criterion.ColumnKey)
		}

		detections, err := DetectUniqueCandidateColumnValues(candidatos, criterion.ColumnKey)
		if err != nil {
			return err
		}

		availableValues := make(map[string]struct{}, len(detections))
		for _, detection := range detections {
			availableValues[detection.ValorNormalizado] = struct{}{}
		}

		if len(criterion.SelectedValues) == 0 {
			return fmt.Errorf("criterio soft %q precisa de pelo menos um valor selecionado", criterion.Type)
		}
		for _, value := range criterion.SelectedValues {
			if _, ok := availableValues[value]; !ok {
				return fmt.Errorf("valor %q nao existe na coluna %q", value, criterion.ColumnKey)
			}
		}

		switch criterion.Type {
		case types.SoftCriterionMinValue, types.SoftCriterionMaxValue:
			if len(criterion.SelectedValues) != 1 {
				return fmt.Errorf("criterio soft %q exige exatamente um valor selecionado", criterion.Type)
			}
			if criterion.Threshold <= 0 {
				return fmt.Errorf("criterio soft %q exige threshold maior que zero", criterion.Type)
			}
		case types.SoftCriterionAtLeastOneEach, types.SoftCriterionBalancedDistribution, types.SoftCriterionGroupTogether:
			// Nessas regras o threshold é ignorado nesta etapa.
		}
	}

	return nil
}

// ValidateAllocationParams confere se os limites de alocação estão coerentes e
// se os critérios soft estruturados são válidos para os candidatos atuais.
func ValidateAllocationParams(params types.AllocationParams, candidatos []types.Candidato) error {
	normalized := NormalizeAllocationParams(params)

	if normalized.GruposPorHorario <= 0 {
		return fmt.Errorf("grupos por horario deve ser maior que zero")
	}
	if normalized.MinPessoasPorGrupo <= 0 {
		return fmt.Errorf("minimo de pessoas por grupo deve ser maior que zero")
	}
	if normalized.MaxPessoasPorGrupo <= 0 {
		return fmt.Errorf("maximo de pessoas por grupo deve ser maior que zero")
	}
	if normalized.MinPessoasPorGrupo > normalized.MaxPessoasPorGrupo {
		return fmt.Errorf("minimo de pessoas por grupo nao pode ser maior que o maximo")
	}
	if normalized.AvaliadoresPorGrupo <= 0 {
		return fmt.Errorf("avaliadores por grupo deve ser maior que zero")
	}

	if err := ValidateSoftCriteria(normalized.SoftCriteria, candidatos); err != nil {
		return err
	}

	return nil
}

func buildValidationMessages(mappings []types.PreferenceScheduleMapping, params types.AllocationParams, candidatos []types.Candidato) ([]types.ValidationMessage, error) {
	messages := make([]types.ValidationMessage, 0)

	if err := ValidatePreferenceScheduleMappings(mappings); err != nil {
		messages = append(messages, types.ValidationMessage{
			Level:   types.ValidationMessageLevelError,
			Code:    "invalid_preference_mappings",
			Message: err.Error(),
		})
		return messages, err
	}

	if err := ValidateAllocationParams(params, candidatos); err != nil {
		messages = append(messages, types.ValidationMessage{
			Level:   types.ValidationMessageLevelError,
			Code:    "invalid_allocation_params",
			Message: err.Error(),
		})
		return messages, err
	}

	messages = append(messages, types.ValidationMessage{
		Level:   types.ValidationMessageLevelWarning,
		Code:    "validation_ok",
		Message: "configuracao validada com sucesso",
	})

	return messages, nil
}

// DescribeSoftCriterion devolve uma descrição humana e estável do critério
// soft para logs, debug e resumos do CLI/UI.
func DescribeSoftCriterion(criterion types.SoftCriterion) string {
	values := joinSoftCriterionValues(criterion.SelectedValues)
	switch criterion.Type {
	case types.SoftCriterionMinValue:
		value := values
		return fmt.Sprintf("Se houver pelo menos uma pessoa de %s em um grupo, tente manter pelo menos %d %s de %s nesse grupo.", value, criterion.Threshold, personNoun(criterion.Threshold), value)
	case types.SoftCriterionAtLeastOneEach:
		return fmt.Sprintf("Garanta representacao de todos os valores selecionados em %s: %s.", humanizeCriterionColumn(criterion.ColumnKey), values)
	case types.SoftCriterionBalancedDistribution:
		return fmt.Sprintf("Distribua os valores selecionados em %s de forma equilibrada entre os grupos: %s.", humanizeCriterionColumn(criterion.ColumnKey), values)
	case types.SoftCriterionGroupTogether:
		return fmt.Sprintf("Se possivel, mantenha no mesmo grupo as pessoas com os valores selecionados em %s: %s.", humanizeCriterionColumn(criterion.ColumnKey), values)
	case types.SoftCriterionMaxValue:
		value := values
		return fmt.Sprintf("Se houver pessoas de %s em um grupo, tente manter no maximo %d %s de %s nesse grupo.", value, criterion.Threshold, personNoun(criterion.Threshold), value)
	default:
		return fmt.Sprintf("%s [%s]", criterion.Type, values)
	}
}

func buildHumanSummary(
	detections []types.UniqueValueDetection,
	normalizedMappings []types.PreferenceScheduleMapping,
	normalizedParams types.AllocationParams,
	validationMessages []types.ValidationMessage,
) types.HumanSummary {
	summary := types.HumanSummary{
		DetectedPreferences:    make([]string, 0, len(detections)),
		MappedPreferences:      make([]string, 0, len(normalizedMappings)),
		AllocationParameters:   make([]string, 0, 4),
		SoftCriteria:           make([]string, 0, len(normalizedParams.SoftCriteria)),
		NormalizedValues:       make([]string, 0, len(detections)+len(normalizedMappings)+len(normalizedParams.SoftCriteria)),
		ValidationObservations: make([]string, 0, len(validationMessages)),
	}

	for _, detection := range detections {
		summary.DetectedPreferences = append(summary.DetectedPreferences,
			fmt.Sprintf("%s (%s, %d ocorrencias)", detection.ValorOriginal, detection.ValorNormalizado, detection.Ocorrencias))
		summary.NormalizedValues = append(summary.NormalizedValues,
			fmt.Sprintf("preferencia %q -> %q", detection.ValorOriginal, detection.ValorNormalizado))
	}

	for _, mapping := range normalizedMappings {
		summary.MappedPreferences = append(summary.MappedPreferences,
			fmt.Sprintf("%s -> %s %s", mapping.ValorPreferencia, mapping.Dia, mapping.Hora))
	}

	summary.AllocationParameters = append(summary.AllocationParameters,
		fmt.Sprintf("Grupos por horario: %d", normalizedParams.GruposPorHorario),
		fmt.Sprintf("Minimo de pessoas por grupo: %d", normalizedParams.MinPessoasPorGrupo),
		fmt.Sprintf("Maximo de pessoas por grupo: %d", normalizedParams.MaxPessoasPorGrupo),
		fmt.Sprintf("Avaliadores por grupo: %d", normalizedParams.AvaliadoresPorGrupo),
	)

	for _, criterion := range normalizedParams.SoftCriteria {
		summary.SoftCriteria = append(summary.SoftCriteria, DescribeSoftCriterion(criterion))
		summary.NormalizedValues = append(summary.NormalizedValues,
			fmt.Sprintf("criterio soft %s em %s com valores [%s]", criterion.Type, criterion.ColumnKey, strings.Join(criterion.SelectedValues, ", ")))
	}

	for _, message := range validationMessages {
		summary.ValidationObservations = append(summary.ValidationObservations,
			fmt.Sprintf("%s: %s", strings.ToUpper(string(message.Level)), message.Message))
	}

	return summary
}

// BuildAllocationConfiguration monta o contrato em camadas usado pelo CLI e
// pela UI para depuração, validação e futura execução do algoritmo.
func BuildAllocationConfiguration(
	detections []types.UniqueValueDetection,
	mappings []types.PreferenceScheduleMapping,
	params types.AllocationParams,
	candidatos []types.Candidato,
) (types.AllocationConfiguration, error) {
	normalizedMappings := NormalizePreferenceScheduleMappings(mappings)
	normalizedParams := NormalizeAllocationParams(params)

	validationMessages, validationErr := buildValidationMessages(normalizedMappings, normalizedParams, candidatos)

	diagnostics := types.AllocationDiagnostics{
		DetectedPreferences: append([]types.UniqueValueDetection(nil), detections...),
		OriginalMappings:    append([]types.PreferenceScheduleMapping(nil), mappings...),
		NormalizedMappings:  append([]types.PreferenceScheduleMapping(nil), normalizedMappings...),
		OriginalParams:      params,
		NormalizedParams:    normalizedParams,
		PreferenceMappings:  make([]types.PreferenceMappingDiagnostic, 0, len(normalizedMappings)),
		SoftCriteria:        make([]types.SoftCriterionDiagnostic, 0, len(normalizedParams.SoftCriteria)),
		ValidationMessages:  append([]types.ValidationMessage(nil), validationMessages...),
		HasErrors:           validationErr != nil,
	}

	for index, normalizedMapping := range normalizedMappings {
		originalMapping := normalizedMapping
		if index < len(mappings) {
			originalMapping = mappings[index]
		}

		detected := types.UniqueValueDetection{}
		for _, detection := range detections {
			if detection.ValorNormalizado == normalizedMapping.ValorPreferencia {
				detected = detection
				break
			}
		}

		diagnostics.PreferenceMappings = append(diagnostics.PreferenceMappings, types.PreferenceMappingDiagnostic{
			DetectedValue:     detected,
			OriginalMapping:   originalMapping,
			NormalizedMapping: normalizedMapping,
		})
	}

	for index, normalizedCriterion := range normalizedParams.SoftCriteria {
		originalCriterion := normalizedCriterion
		if index < len(params.SoftCriteria) {
			originalCriterion = params.SoftCriteria[index]
		}

		diagnostics.SoftCriteria = append(diagnostics.SoftCriteria, types.SoftCriterionDiagnostic{
			OriginalCriterion:   originalCriterion,
			NormalizedCriterion: normalizedCriterion,
			Summary:             DescribeSoftCriterion(normalizedCriterion),
		})
	}

	config := types.AllocationConfiguration{
		Summary: buildHumanSummary(detections, normalizedMappings, normalizedParams, validationMessages),
		Normalized: types.NormalizedAllocationInput{
			PreferenceMappings: normalizedMappings,
			Params:             normalizedParams,
		},
		Diagnostics: diagnostics,
		Result: types.AllocationExecutionResult{
			Status: "not_run",
			Notes: []string{
				"algoritmo de alocacao ainda nao foi executado",
			},
		},
	}

	if validationErr != nil {
		return config, validationErr
	}

	return config, nil
}

func personNoun(count int) string {
	if count == 1 {
		return "pessoa"
	}
	return "pessoas"
}

func joinSoftCriterionValues(values []string) string {
	if len(values) == 0 {
		return ""
	}

	displayValues := make([]string, 0, len(values))
	for _, value := range values {
		displayValues = append(displayValues, formatSoftCriterionValue(value))
	}

	switch len(displayValues) {
	case 1:
		return displayValues[0]
	case 2:
		return displayValues[0] + " e " + displayValues[1]
	default:
		return strings.Join(displayValues[:len(displayValues)-1], ", ") + " e " + displayValues[len(displayValues)-1]
	}
}

func formatSoftCriterionValue(value string) string {
	cleaned := strings.TrimSpace(value)
	if cleaned == "" {
		return cleaned
	}

	parts := strings.Fields(cleaned)
	for i, part := range parts {
		if part == strings.ToLower(part) && len(part) <= 4 {
			parts[i] = strings.ToUpper(part)
			continue
		}
		if len(part) == 1 {
			parts[i] = strings.ToUpper(part)
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
	}
	return strings.Join(parts, " ")
}
