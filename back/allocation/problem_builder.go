package allocation

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"candidate_alocator/back/logic"
	types "candidate_alocator/back/type"
)

// BuildAllocationProblem transforma a configuração normalizada e os dados de
// domínio no contrato único e explícito consumido pelo solver.
func BuildAllocationProblem(
	config types.AllocationConfiguration,
	candidatos []types.Candidato,
	avaliadores []types.Avaliador,
	restricoes []types.Restricao,
) (types.AllocationProblem, error) {
	if err := logic.ValidatePreferenceScheduleMappings(config.Normalized.PreferenceMappings); err != nil {
		return types.AllocationProblem{}, err
	}
	if err := logic.ValidateAllocationParams(config.Normalized.Params, candidatos); err != nil {
		return types.AllocationProblem{}, err
	}

	groups, preferenceToGroupIDs := buildSolverGroups(config.Normalized.PreferenceMappings, config.Normalized.Params, avaliadores)
	candidateIDs := buildCandidateIDsByName(candidatos)
	evaluatorIDs := buildEvaluatorIDsBySigla(avaliadores)
	restrictionsByCandidate, err := buildCandidateRestrictions(restricoes, candidateIDs, evaluatorIDs)
	if err != nil {
		return types.AllocationProblem{}, err
	}

	criteriaColumns := collectSoftCriterionColumns(config.Normalized.Params.SoftCriteria)
	problemCandidates := make([]types.SolverCandidate, 0, len(candidatos))
	for idx, candidato := range candidatos {
		candidateID := idx + 1
		preferredGroupIDs, preferenceRanks := buildCandidatePreferenceData(candidato, preferenceToGroupIDs)
		problemCandidates = append(problemCandidates, types.SolverCandidate{
			ID:                      candidateID,
			Name:                    strings.TrimSpace(candidato.Nome),
			PreferredGroupIDs:       preferredGroupIDs,
			PreferenceRankByGroupID: preferenceRanks,
			Attributes:              buildCandidateAttributes(candidato, criteriaColumns),
			EvaluatorRestrictions:   restrictionsByCandidate[candidateID],
		})
	}

	baseOptimization := logic.WorkflowDefinition().BaseOptimization
	return types.AllocationProblem{
		Candidates: problemCandidates,
		Groups:     groups,
		HardRestrictions: types.SolverHardRestrictions{
			AllCandidatesMustBeAssigned:         true,
			RespectCandidatePreferences:         true,
			EnforceGroupCapacity:                true,
			EnforceForbiddenEvaluators:          true,
			EnforceMinCandidatesOnCompleteState: true,
		},
		SoftRules: types.SolverSoftRules{
			PreferencePenaltyByRank: append([]int(nil), baseOptimization.PreferencePenaltyByRank...),
			AvoidEvaluatorPenalty:   baseOptimization.AvoidEvaluatorPenalty,
			Criteria:                append([]types.SoftCriterion(nil), config.Normalized.Params.SoftCriteria...),
		},
	}, nil
}

func buildSolverGroups(
	mappings []types.PreferenceScheduleMapping,
	params types.AllocationParams,
	avaliadores []types.Avaliador,
) ([]types.SolverGroup, map[string][]int) {
	sortedEvaluators := append([]types.Avaliador(nil), avaliadores...)
	sort.SliceStable(sortedEvaluators, func(i, j int) bool {
		if sortedEvaluators[i].ID != sortedEvaluators[j].ID {
			return sortedEvaluators[i].ID < sortedEvaluators[j].ID
		}
		return strings.TrimSpace(sortedEvaluators[i].Sigla) < strings.TrimSpace(sortedEvaluators[j].Sigla)
	})

	groups := make([]types.SolverGroup, 0, len(mappings)*params.GruposPorHorario)
	preferenceToGroupIDs := make(map[string][]int, len(mappings))
	groupIndex := 0

	for _, mapping := range mappings {
		key := normalizeSolverText(mapping.ValorPreferencia)
		for slot := 0; slot < params.GruposPorHorario; slot++ {
			groupIndex++
			group := types.SolverGroup{
				ID:            groupIndex,
				Label:         buildSolverGroupLabel(mapping, slot+1),
				ScheduleKey:   key,
				EvaluatorIDs:  selectEvaluatorIDs(sortedEvaluators, groupIndex-1, params.AvaliadoresPorGrupo),
				MinCandidates: params.MinPessoasPorGrupo,
				MaxCandidates: params.MaxPessoasPorGrupo,
			}
			groups = append(groups, group)
			preferenceToGroupIDs[key] = append(preferenceToGroupIDs[key], group.ID)
		}
	}

	return groups, preferenceToGroupIDs
}

func buildSolverGroupLabel(mapping types.PreferenceScheduleMapping, groupNumber int) string {
	return strings.TrimSpace(mapping.Dia) + " " + strings.TrimSpace(mapping.Hora) + " grupo " + strconv.Itoa(groupNumber)
}

func selectEvaluatorIDs(avaliadores []types.Avaliador, start, count int) []int {
	if len(avaliadores) == 0 || count <= 0 {
		return []int{}
	}
	if count > len(avaliadores) {
		count = len(avaliadores)
	}

	selected := make([]int, 0, count)
	for i := 0; i < count; i++ {
		index := (start + i) % len(avaliadores)
		selected = append(selected, avaliadores[index].ID)
	}
	return selected
}

func buildCandidateIDsByName(candidatos []types.Candidato) map[string]int {
	result := make(map[string]int, len(candidatos))
	for idx, candidato := range candidatos {
		result[strings.TrimSpace(candidato.Nome)] = idx + 1
	}
	return result
}

func buildEvaluatorIDsBySigla(avaliadores []types.Avaliador) map[string]int {
	result := make(map[string]int, len(avaliadores))
	for _, avaliador := range avaliadores {
		result[strings.TrimSpace(avaliador.Sigla)] = avaliador.ID
	}
	return result
}

func buildCandidateRestrictions(
	restricoes []types.Restricao,
	candidateIDs map[string]int,
	evaluatorIDs map[string]int,
) (map[int]types.SolverCandidateRestrictions, error) {
	result := make(map[int]types.SolverCandidateRestrictions, len(candidateIDs))

	for _, restricao := range restricoes {
		candidateName := strings.TrimSpace(restricao.Candidato)
		candidateID, ok := candidateIDs[candidateName]
		if !ok {
			return nil, fmt.Errorf("candidato %q nao encontrado para restricao", candidateName)
		}

		restrictions := result[candidateID]
		forbidden, err := parseRestrictionEvaluatorIDs(restricao.NaoPosso, evaluatorIDs)
		if err != nil {
			return nil, fmt.Errorf("erro ao resolver NaoPosso de %q: %w", candidateName, err)
		}
		avoid, err := parseRestrictionEvaluatorIDs(restricao.PrefiroNao, evaluatorIDs)
		if err != nil {
			return nil, fmt.Errorf("erro ao resolver PrefiroNao de %q: %w", candidateName, err)
		}

		restrictions.ForbiddenEvaluatorIDs = mergeUniqueInts(restrictions.ForbiddenEvaluatorIDs, forbidden)
		restrictions.AvoidEvaluatorIDs = mergeUniqueInts(restrictions.AvoidEvaluatorIDs, avoid)
		result[candidateID] = restrictions
	}

	return result, nil
}

func parseRestrictionEvaluatorIDs(raw string, evaluatorIDs map[string]int) ([]int, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}

	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || unicode.IsSpace(r)
	})

	result := make([]int, 0, len(parts))
	for _, part := range parts {
		sigla := strings.TrimSpace(part)
		if sigla == "" {
			continue
		}
		id, ok := evaluatorIDs[sigla]
		if !ok {
			return nil, fmt.Errorf("avaliador %q nao encontrado", sigla)
		}
		result = append(result, id)
	}

	return mergeUniqueInts(nil, result), nil
}

func mergeUniqueInts(base []int, values []int) []int {
	seen := make(map[int]struct{}, len(base)+len(values))
	result := make([]int, 0, len(base)+len(values))

	for _, value := range base {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}

	sort.Ints(result)
	return result
}

func collectSoftCriterionColumns(criteria []types.SoftCriterion) []string {
	seen := make(map[string]struct{}, len(criteria))
	columns := make([]string, 0, len(criteria))
	for _, criterion := range criteria {
		columnKey := strings.TrimSpace(criterion.ColumnKey)
		if columnKey == "" {
			continue
		}
		if _, ok := seen[columnKey]; ok {
			continue
		}
		seen[columnKey] = struct{}{}
		columns = append(columns, columnKey)
	}
	sort.Strings(columns)
	return columns
}

func buildCandidatePreferenceData(candidato types.Candidato, preferenceToGroupIDs map[string][]int) ([]int, map[int]int) {
	result := make([]int, 0)
	seen := make(map[int]struct{})
	ranks := make(map[int]int)

	for preferenceRank, option := range candidato.Opcoes {
		groupIDs := preferenceToGroupIDs[normalizeSolverText(option)]
		for _, groupID := range groupIDs {
			if _, ok := seen[groupID]; ok {
				continue
			}
			seen[groupID] = struct{}{}
			result = append(result, groupID)
			ranks[groupID] = preferenceRank
		}
	}

	return result, ranks
}

func buildCandidateAttributes(candidato types.Candidato, columns []string) map[string]string {
	attributes := make(map[string]string, len(columns))
	for _, column := range columns {
		if value, ok := readCandidateAttribute(candidato, column); ok {
			attributes[column] = normalizeSolverText(value)
		}
	}
	return attributes
}

func readCandidateAttribute(candidato types.Candidato, column string) (string, bool) {
	switch strings.TrimSpace(column) {
	case "timestamp":
		return candidato.Timestamp, true
	case "nome":
		return candidato.Nome, true
	case "cpf":
		return candidato.CPF, true
	case "numero":
		return candidato.Numero, true
	case "semestre":
		return candidato.Semestre, true
	case "curso":
		return candidato.Curso, true
	case "email_secundario":
		return candidato.EmailSecundario, true
	case "email_pessoal":
		return candidato.EmailPessoal, true
	default:
		if candidato.Extras == nil {
			return "", false
		}
		value, ok := candidato.Extras[column]
		if !ok || value == nil {
			return "", false
		}
		return string(*value), true
	}
}

func normalizeSolverText(value string) string {
	return strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(value)), " "))
}
