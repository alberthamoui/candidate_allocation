package logic

import (
	"fmt"
	"sort"
	"strings"

	types "candidate_alocator/back/type"
)

func normalizeDetectedValue(value string) string {
	return strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(value)), " "))
}

func humanizeCriterionColumn(key string) string {
	return strings.ReplaceAll(strings.TrimSpace(key), "_", " ")
}

func detectUniqueValues(values []string) []types.UniqueValueDetection {
	detections := make([]types.UniqueValueDetection, 0)
	seen := make(map[string]int)

	for _, rawValue := range values {
		valorOriginal := strings.TrimSpace(rawValue)
		if valorOriginal == "" {
			continue
		}

		valorNormalizado := normalizeDetectedValue(valorOriginal)
		if valorNormalizado == "" {
			continue
		}

		if idx, ok := seen[valorNormalizado]; ok {
			detections[idx].Ocorrencias++
			continue
		}

		seen[valorNormalizado] = len(detections)
		detections = append(detections, types.UniqueValueDetection{
			ValorOriginal:    valorOriginal,
			ValorNormalizado: valorNormalizado,
			Ocorrencias:      1,
		})
	}

	return detections
}

// DetectUniquePreferenceValues coleta todos os valores presentes nas colunas de
// preferência dos candidatos e delega a deduplicação ao detector genérico.
func DetectUniquePreferenceValues(candidatos []types.Candidato) []types.UniqueValueDetection {
	values := make([]string, 0)
	for _, candidato := range candidatos {
		values = append(values, candidato.Opcoes...)
	}
	return detectUniqueValues(values)
}

// ListCandidateCriterionColumns lista todas as colunas não-opção disponíveis
// para configurar critérios soft, incluindo campos extras importados.
func ListCandidateCriterionColumns(candidatos []types.Candidato) []types.CandidateCriterionColumn {
	columns := make([]types.CandidateCriterionColumn, 0)
	seen := make(map[string]struct{})

	for _, field := range types.CandidateFields() {
		switch field.JSONName {
		case "opcoes", "extras":
			continue
		default:
			columns = append(columns, types.CandidateCriterionColumn{
				Key:   field.JSONName,
				Label: humanizeCriterionColumn(field.JSONName),
			})
			seen[field.JSONName] = struct{}{}
		}
	}

	extraKeys := make([]string, 0)
	for _, candidato := range candidatos {
		for key := range candidato.Extras {
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			extraKeys = append(extraKeys, key)
		}
	}
	sort.Strings(extraKeys)
	for _, key := range extraKeys {
		columns = append(columns, types.CandidateCriterionColumn{
			Key:     key,
			Label:   humanizeCriterionColumn(key),
			IsExtra: true,
		})
	}

	return columns
}

func isCandidateCoreCriterionColumn(columnKey string) bool {
	for _, field := range types.CandidateFields() {
		if field.JSONName == columnKey && field.JSONName != "opcoes" && field.JSONName != "extras" {
			return true
		}
	}
	return false
}

func candidateCoreFieldValue(candidate types.Candidato, columnKey string) string {
	switch columnKey {
	case "timestamp":
		return candidate.Timestamp
	case "nome":
		return candidate.Nome
	case "cpf":
		return candidate.CPF
	case "numero":
		return candidate.Numero
	case "semestre":
		return candidate.Semestre
	case "curso":
		return candidate.Curso
	case "email_secundario":
		return candidate.EmailSecundario
	case "email_pessoal":
		return candidate.EmailPessoal
	default:
		return ""
	}
}

// DetectUniqueCandidateColumnValues retorna os valores únicos de qualquer
// coluna não-opção do candidato, incluindo campos extras.
func DetectUniqueCandidateColumnValues(candidatos []types.Candidato, columnKey string) ([]types.UniqueValueDetection, error) {
	columnKey = strings.TrimSpace(columnKey)
	if columnKey == "" {
		return nil, fmt.Errorf("columnKey nao pode ser vazio")
	}
	if columnKey == "opcoes" {
		return nil, fmt.Errorf("columnKey %q nao e elegivel para criterio soft", columnKey)
	}

	values := make([]string, 0, len(candidatos))
	if isCandidateCoreCriterionColumn(columnKey) {
		for _, candidato := range candidatos {
			values = append(values, candidateCoreFieldValue(candidato, columnKey))
		}
		return detectUniqueValues(values), nil
	}

	foundExtraColumn := false
	for _, candidato := range candidatos {
		if candidato.Extras == nil {
			continue
		}
		value, ok := candidato.Extras[columnKey]
		if !ok {
			continue
		}
		foundExtraColumn = true
		if value == nil {
			values = append(values, "")
			continue
		}
		values = append(values, string(*value))
	}

	if !foundExtraColumn {
		return nil, fmt.Errorf("columnKey %q nao foi encontrado nos candidatos", columnKey)
	}

	return detectUniqueValues(values), nil
}

// ValidatePreferenceScheduleMappings garante que cada valor detectado foi
// associado a um dia e hora válidos, sem duplicar a mesma preferência.
func ValidatePreferenceScheduleMappings(mappings []types.PreferenceScheduleMapping) error {
	seen := make(map[string]struct{}, len(mappings))
	for _, mapping := range mappings {
		valorPreferencia := strings.TrimSpace(mapping.ValorPreferencia)
		dia := strings.TrimSpace(mapping.Dia)
		hora := strings.TrimSpace(mapping.Hora)

		if valorPreferencia == "" {
			return fmt.Errorf("valor de preferencia nao pode ser vazio")
		}
		if dia == "" {
			return fmt.Errorf("dia nao pode ser vazio para a preferencia %q", valorPreferencia)
		}
		if hora == "" {
			return fmt.Errorf("hora nao pode ser vazia para a preferencia %q", valorPreferencia)
		}

		key := normalizeDetectedValue(valorPreferencia)
		if _, ok := seen[key]; ok {
			return fmt.Errorf("preferencia %q foi mapeada mais de uma vez", valorPreferencia)
		}
		seen[key] = struct{}{}
	}

	return nil
}
