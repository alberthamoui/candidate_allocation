package allocation

import (
	"fmt"
	"sort"
	"strings"

	types "candidate_alocator/back/type"
)

const invalidStatePenalty = 1_000_000

func ScoreAllocation(problem types.AllocationProblem, state types.PartialAllocationState) types.SoftScoreBreakdown {
	if !IsCompleteState(problem, state) {
		return types.SoftScoreBreakdown{
			TotalPenalty: invalidStatePenalty,
			Components: []types.SoftScoreComponent{
				{Code: "incomplete_state", Penalty: invalidStatePenalty, Message: "soft scorer requer uma alocacao completa"},
			},
		}
	}

	if viable, violations := IsStateViable(problem, state); !viable {
		return types.SoftScoreBreakdown{
			TotalPenalty: invalidStatePenalty,
			Components: []types.SoftScoreComponent{
				{Code: "invalid_state", Penalty: invalidStatePenalty, Message: formatInvalidStateMessage(violations)},
			},
		}
	}
	return scoreAssignedAllocation(problem, state)
}

// scoreAssignedAllocation evaluates the soft objective for the candidates
// currently assigned. It is used for a hard-safe partial fallback, whose
// missing candidates are reported separately instead of becoming soft points.
func scoreAssignedAllocation(problem types.AllocationProblem, state types.PartialAllocationState) types.SoftScoreBreakdown {
	components := make([]types.SoftScoreComponent, 0)
	components = append(components, scorePreferencePenalties(problem, state)...)
	components = append(components, scoreAvoidEvaluatorPenalties(problem, state)...)

	for _, criterion := range problem.SoftRules.Criteria {
		components = append(components, scoreSoftCriterion(problem, state, criterion)...)
	}

	total := 0
	for _, component := range components {
		total += component.Penalty
	}

	return types.SoftScoreBreakdown{
		TotalPenalty: total,
		Components:   components,
	}
}

func scorePreferencePenalties(problem types.AllocationProblem, state types.PartialAllocationState) []types.SoftScoreComponent {
	candidates := candidateByID(problem)
	candidateIDs := make([]int, 0, len(state.Assignments))
	for candidateID := range state.Assignments {
		candidateIDs = append(candidateIDs, candidateID)
	}
	sort.Ints(candidateIDs)

	components := make([]types.SoftScoreComponent, 0, len(candidateIDs))
	for _, candidateID := range candidateIDs {
		groupID := state.Assignments[candidateID]
		rank := candidatePreferenceRank(candidates[candidateID], groupID)
		penalty := preferencePenaltyForRank(problem.SoftRules.PreferencePenaltyByRank, rank)
		components = append(components, types.SoftScoreComponent{
			Code:    "preference_rank",
			Penalty: penalty,
			Message: fmt.Sprintf("candidato %d alocado na preferencia %d", candidateID, rank+1),
		})
	}

	return components
}

func candidatePreferenceRank(candidate types.SolverCandidate, groupID int) int {
	if rank, ok := candidate.PreferenceRankByGroupID[groupID]; ok {
		return rank
	}
	return indexOfInt(candidate.PreferredGroupIDs, groupID)
}

func scoreAvoidEvaluatorPenalties(problem types.AllocationProblem, state types.PartialAllocationState) []types.SoftScoreComponent {
	candidates := candidateByID(problem)
	groups := groupByID(problem)
	candidateIDs := make([]int, 0, len(state.Assignments))
	for candidateID := range state.Assignments {
		candidateIDs = append(candidateIDs, candidateID)
	}
	sort.Ints(candidateIDs)

	components := make([]types.SoftScoreComponent, 0)
	for _, candidateID := range candidateIDs {
		groupID := state.Assignments[candidateID]
		group := groups[groupID]
		for _, evaluatorID := range group.EvaluatorIDs {
			if !containsInt(candidates[candidateID].EvaluatorRestrictions.AvoidEvaluatorIDs, evaluatorID) {
				continue
			}
			components = append(components, types.SoftScoreComponent{
				Code:    "avoid_evaluator",
				Penalty: problem.SoftRules.AvoidEvaluatorPenalty,
				Message: fmt.Sprintf("candidato %d alocado com avaliador evitado %d no grupo %d", candidateID, evaluatorID, groupID),
			})
		}
	}

	return components
}

func scoreSoftCriterion(problem types.AllocationProblem, state types.PartialAllocationState, criterion types.SoftCriterion) []types.SoftScoreComponent {
	switch criterion.Type {
	case types.SoftCriterionMinValue:
		return scoreMinValueCriterion(problem, state, criterion)
	case types.SoftCriterionMaxValue:
		return scoreMaxValueCriterion(problem, state, criterion)
	case types.SoftCriterionAtLeastOneEach:
		return scoreAtLeastOneEachCriterion(problem, state, criterion)
	case types.SoftCriterionBalancedDistribution:
		return scoreBalancedDistributionCriterion(problem, state, criterion)
	case types.SoftCriterionGroupTogether:
		return scoreGroupTogetherCriterion(problem, state, criterion)
	default:
		return nil
	}
}

func scoreMinValueCriterion(problem types.AllocationProblem, state types.PartialAllocationState, criterion types.SoftCriterion) []types.SoftScoreComponent {
	if len(criterion.SelectedValues) != 1 {
		return nil
	}

	selectedValue := criterion.SelectedValues[0]
	counts := countSelectedValuesByGroup(problem, state, criterion.ColumnKey, criterion.SelectedValues)
	groupIDs := sortedGroupKeysFromProblem(problem)
	components := make([]types.SoftScoreComponent, 0)

	for _, groupID := range groupIDs {
		count := counts[groupID][selectedValue]
		if count == 0 || count >= criterion.Threshold {
			continue
		}
		penalty := criterion.Threshold - count
		components = append(components, types.SoftScoreComponent{
			Code:    "soft_min_value",
			Penalty: penalty,
			Message: fmt.Sprintf("grupo %d ficou com deficit de %s para %q", groupID, criterion.ColumnKey, selectedValue),
		})
	}

	return components
}

func scoreMaxValueCriterion(problem types.AllocationProblem, state types.PartialAllocationState, criterion types.SoftCriterion) []types.SoftScoreComponent {
	if len(criterion.SelectedValues) != 1 {
		return nil
	}

	selectedValue := criterion.SelectedValues[0]
	counts := countSelectedValuesByGroup(problem, state, criterion.ColumnKey, criterion.SelectedValues)
	groupIDs := sortedGroupKeysFromProblem(problem)
	components := make([]types.SoftScoreComponent, 0)

	for _, groupID := range groupIDs {
		count := counts[groupID][selectedValue]
		if count <= criterion.Threshold {
			continue
		}
		penalty := count - criterion.Threshold
		components = append(components, types.SoftScoreComponent{
			Code:    "soft_max_value",
			Penalty: penalty,
			Message: fmt.Sprintf("grupo %d excedeu o maximo de %s para %q", groupID, criterion.ColumnKey, selectedValue),
		})
	}

	return components
}

func scoreAtLeastOneEachCriterion(problem types.AllocationProblem, state types.PartialAllocationState, criterion types.SoftCriterion) []types.SoftScoreComponent {
	counts := countSelectedValuesByGroup(problem, state, criterion.ColumnKey, criterion.SelectedValues)
	groupIDs := sortedGroupKeysFromProblem(problem)
	components := make([]types.SoftScoreComponent, 0)

	for _, groupID := range groupIDs {
		for _, selectedValue := range criterion.SelectedValues {
			if counts[groupID][selectedValue] > 0 {
				continue
			}
			components = append(components, types.SoftScoreComponent{
				Code:    "soft_at_least_one_each",
				Penalty: 1,
				Message: fmt.Sprintf("grupo %d nao contem valor %q em %s", groupID, selectedValue, criterion.ColumnKey),
			})
		}
	}

	return components
}

func scoreBalancedDistributionCriterion(problem types.AllocationProblem, state types.PartialAllocationState, criterion types.SoftCriterion) []types.SoftScoreComponent {
	counts := countSelectedValuesByGroup(problem, state, criterion.ColumnKey, criterion.SelectedValues)
	groupIDs := sortedGroupKeysFromProblem(problem)
	components := make([]types.SoftScoreComponent, 0)

	for _, selectedValue := range criterion.SelectedValues {
		groupCounts := make([]int, 0, len(groupIDs))
		for _, groupID := range groupIDs {
			groupCounts = append(groupCounts, counts[groupID][selectedValue])
		}
		median := medianInt(groupCounts)
		penalty := 0
		for _, count := range groupCounts {
			if count >= median {
				penalty += count - median
			} else {
				penalty += median - count
			}
		}
		if penalty == 0 {
			continue
		}
		components = append(components, types.SoftScoreComponent{
			Code:    "soft_balanced_distribution",
			Penalty: penalty,
			Message: fmt.Sprintf("distribuicao de %q em %s desviou da mediana %d", selectedValue, criterion.ColumnKey, median),
		})
	}

	return components
}

func scoreGroupTogetherCriterion(problem types.AllocationProblem, state types.PartialAllocationState, criterion types.SoftCriterion) []types.SoftScoreComponent {
	selectedValues := make(map[string]struct{}, len(criterion.SelectedValues))
	for _, selectedValue := range criterion.SelectedValues {
		selectedValues[selectedValue] = struct{}{}
	}

	groupMembers := buildStateGroupMembers(state)
	candidates := candidateByID(problem)
	components := make([]types.SoftScoreComponent, 0)
	for groupID, members := range groupMembers {
		selectedCount := 0
		for _, candidateID := range members {
			value := candidates[candidateID].Attributes[criterion.ColumnKey]
			if _, ok := selectedValues[value]; ok {
				selectedCount++
			}
		}
		if selectedCount == 0 || len(members) == selectedCount {
			continue
		}
		components = append(components, types.SoftScoreComponent{
			Code:    "soft_group_together",
			Penalty: selectedCount,
			Message: fmt.Sprintf("grupo %d deixou %d pessoa(s) selecionada(s) em uma mesa misturada por %s", groupID, selectedCount, criterion.ColumnKey),
		})
	}
	return components
}

func medianInt(values []int) int {
	if len(values) == 0 {
		return 0
	}
	sorted := append([]int(nil), values...)
	sort.Ints(sorted)
	return sorted[len(sorted)/2]
}

func countSelectedValuesByGroup(problem types.AllocationProblem, state types.PartialAllocationState, columnKey string, selectedValues []string) map[int]map[string]int {
	selectedSet := make(map[string]struct{}, len(selectedValues))
	for _, selectedValue := range selectedValues {
		selectedSet[selectedValue] = struct{}{}
	}

	counts := make(map[int]map[string]int, len(problem.Groups))
	for _, group := range problem.Groups {
		counts[group.ID] = make(map[string]int, len(selectedValues))
	}

	candidates := candidateByID(problem)
	groupMembers := buildStateGroupMembers(state)
	for groupID, members := range groupMembers {
		for _, candidateID := range members {
			value := candidates[candidateID].Attributes[columnKey]
			if _, ok := selectedSet[value]; !ok {
				continue
			}
			counts[groupID][value]++
		}
	}

	return counts
}

func preferencePenaltyForRank(penalties []int, rank int) int {
	if len(penalties) == 0 {
		return rank
	}
	if rank < 0 {
		return penalties[len(penalties)-1]
	}
	if rank >= len(penalties) {
		return penalties[len(penalties)-1]
	}
	return penalties[rank]
}

func indexOfInt(values []int, target int) int {
	for index, value := range values {
		if value == target {
			return index
		}
	}
	return -1
}

func sortedGroupKeysFromProblem(problem types.AllocationProblem) []int {
	groupIDs := make([]int, 0, len(problem.Groups))
	for _, group := range problem.Groups {
		groupIDs = append(groupIDs, group.ID)
	}
	sort.Ints(groupIDs)
	return groupIDs
}

func formatInvalidStateMessage(violations []types.HardConstraintViolation) string {
	if len(violations) == 0 {
		return "estado invalido"
	}
	messages := make([]string, 0, len(violations))
	for _, violation := range violations {
		messages = append(messages, violation.Code)
	}
	return "estado invalido: " + strings.Join(messages, ", ")
}
