package allocation

import (
	"fmt"
	"sort"
	"strings"

	"candidate_alocator/back/logic"
	types "candidate_alocator/back/type"
)

// BuildAllocationQualityReport translates the solver result into clickable
// result characteristics. The report is produced in the backend so the UI does
// not need to know how preference ranks or additional criteria are scored.
func BuildAllocationQualityReport(problem types.AllocationProblem, result types.SolverResult) types.AllocationQualityReport {
	state := types.PartialAllocationState{Assignments: result.Assignments}
	characteristics := buildPreferenceQualityCharacteristics(problem, state)
	characteristics = append(characteristics, buildAvoidEvaluatorQualityCharacteristic(problem, state))
	characteristics = append(characteristics, buildConfiguredQualityCharacteristics(problem, state)...)
	return types.AllocationQualityReport{Characteristics: characteristics}
}

func buildPreferenceQualityCharacteristics(problem types.AllocationProblem, state types.PartialAllocationState) []types.AllocationQualityCharacteristic {
	rankCount := len(problem.SoftRules.PreferencePenaltyByRank)
	if rankCount < 1 {
		rankCount = 1
	}

	byRank := make([][]int, rankCount)
	penaltyByRank := make([]int, rankCount)
	groupsByRank := make([][]int, rankCount)
	candidates := candidateByID(problem)
	for candidateID, groupID := range state.Assignments {
		candidate, ok := candidates[candidateID]
		if !ok {
			continue
		}
		rank := candidatePreferenceRank(candidate, groupID)
		if rank < 0 {
			continue
		}
		for rank >= len(byRank) {
			byRank = append(byRank, nil)
			penaltyByRank = append(penaltyByRank, 0)
			groupsByRank = append(groupsByRank, nil)
		}
		byRank[rank] = append(byRank[rank], candidateID)
		groupsByRank[rank] = append(groupsByRank[rank], groupID)
		penaltyByRank[rank] += preferencePenaltyForRank(problem.SoftRules.PreferencePenaltyByRank, rank)
	}

	result := make([]types.AllocationQualityCharacteristic, 0, len(byRank))
	for rank := range byRank {
		sort.Ints(byRank[rank])
		result = append(result, types.AllocationQualityCharacteristic{
			Code:         fmt.Sprintf("preference_rank_%d", rank+1),
			Label:        fmt.Sprintf("Preferência %d de horário", rank+1),
			Description:  fmt.Sprintf("Candidatos alocados na %dª opção de horário informada.", rank+1),
			Value:        len(byRank[rank]),
			ValueLabel:   pluralizeQualityValue(len(byRank[rank]), "candidato", "candidatos"),
			Penalty:      penaltyByRank[rank],
			Tone:         preferenceQualityTone(rank),
			CandidateIDs: byRank[rank],
			GroupIDs:     uniqueSortedQualityIDs(groupsByRank[rank]),
		})
	}
	return result
}

func buildAvoidEvaluatorQualityCharacteristic(problem types.AllocationProblem, state types.PartialAllocationState) types.AllocationQualityCharacteristic {
	candidates := candidateByID(problem)
	groups := groupByID(problem)
	candidateIDs := make([]int, 0)
	evaluatorIDs := make([]int, 0)
	groupIDs := make([]int, 0)
	penalty := 0
	for candidateID, groupID := range state.Assignments {
		candidate, ok := candidates[candidateID]
		if !ok {
			continue
		}
		for _, evaluatorID := range groups[groupID].EvaluatorIDs {
			if !containsInt(candidate.EvaluatorRestrictions.AvoidEvaluatorIDs, evaluatorID) {
				continue
			}
			candidateIDs = append(candidateIDs, candidateID)
			evaluatorIDs = append(evaluatorIDs, evaluatorID)
			groupIDs = append(groupIDs, groupID)
			penalty += problem.SoftRules.AvoidEvaluatorPenalty
		}
	}
	candidateIDs = uniqueSortedQualityIDs(candidateIDs)
	return types.AllocationQualityCharacteristic{
		Code:         "avoid_evaluator",
		Label:        "Alocações em ‘Prefiro não’",
		Description:  "Candidatos alocados com um avaliador marcado como preferência negativa.",
		Value:        len(candidateIDs),
		ValueLabel:   pluralizeQualityValue(len(candidateIDs), "candidato", "candidatos"),
		Penalty:      penalty,
		Tone:         zeroIsSuccessQualityTone(len(candidateIDs)),
		CandidateIDs: candidateIDs,
		EvaluatorIDs: uniqueSortedQualityIDs(evaluatorIDs),
		GroupIDs:     uniqueSortedQualityIDs(groupIDs),
	}
}

func buildConfiguredQualityCharacteristics(problem types.AllocationProblem, state types.PartialAllocationState) []types.AllocationQualityCharacteristic {
	labels := make(map[types.SoftCriterionType]string)
	for _, option := range logic.WorkflowDefinition().SoftCriterionOptions {
		labels[option.Type] = option.Label
	}

	result := make([]types.AllocationQualityCharacteristic, 0, len(problem.SoftRules.Criteria))
	for index, criterion := range problem.SoftRules.Criteria {
		components := scoreSoftCriterion(problem, state, criterion)
		penalty := 0
		for _, component := range components {
			penalty += component.Penalty
		}

		candidateIDs, groupIDs := qualitySubjectsForCriterion(problem, state, criterion)
		label := labels[criterion.Type]
		if label == "" {
			label = string(criterion.Type)
		}
		result = append(result, types.AllocationQualityCharacteristic{
			Code:         fmt.Sprintf("configured_%d_%s", index+1, criterion.Type),
			Label:        label,
			Description:  configuredQualityDescription(criterion),
			Value:        len(components),
			ValueLabel:   pluralizeQualityValue(len(components), "desvio", "desvios"),
			Penalty:      penalty,
			Tone:         zeroIsSuccessQualityTone(len(components)),
			CandidateIDs: candidateIDs,
			GroupIDs:     groupIDs,
		})
	}
	return result
}

func qualitySubjectsForCriterion(problem types.AllocationProblem, state types.PartialAllocationState, criterion types.SoftCriterion) ([]int, []int) {
	selected := make(map[string]struct{}, len(criterion.SelectedValues))
	for _, value := range criterion.SelectedValues {
		selected[normalizeSolverText(value)] = struct{}{}
	}
	candidates := candidateByID(problem)
	candidateIDs := make([]int, 0)
	groupIDs := make([]int, 0)
	for candidateID, groupID := range state.Assignments {
		value := normalizeSolverText(candidates[candidateID].Attributes[criterion.ColumnKey])
		if _, ok := selected[value]; !ok {
			continue
		}
		candidateIDs = append(candidateIDs, candidateID)
		groupIDs = append(groupIDs, groupID)
	}
	return uniqueSortedQualityIDs(candidateIDs), uniqueSortedQualityIDs(groupIDs)
}

func configuredQualityDescription(criterion types.SoftCriterion) string {
	values := strings.Join(criterion.SelectedValues, ", ")
	if criterion.Threshold > 0 {
		return fmt.Sprintf("Campo %s; valores %s; limite %d.", criterion.ColumnKey, values, criterion.Threshold)
	}
	return fmt.Sprintf("Campo %s; valores %s.", criterion.ColumnKey, values)
}

func uniqueSortedQualityIDs(values []int) []int {
	seen := make(map[int]struct{}, len(values))
	result := make([]int, 0, len(values))
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

func pluralizeQualityValue(value int, singular, plural string) string {
	if value == 1 {
		return singular
	}
	return plural
}

func preferenceQualityTone(rank int) string {
	if rank == 0 {
		return "success"
	}
	if rank == 1 {
		return "neutral"
	}
	return "warning"
}

func zeroIsSuccessQualityTone(value int) string {
	if value == 0 {
		return "success"
	}
	return "warning"
}
