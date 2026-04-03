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
	normalized := params
	normalized.SoftCriteria = NormalizeSoftCriteria(params.SoftCriteria)

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
