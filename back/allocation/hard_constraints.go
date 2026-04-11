package allocation

import (
	"fmt"
	"sort"

	types "candidate_alocator/back/type"
)

func IsCompleteState(problem types.AllocationProblem, state types.PartialAllocationState) bool {
	if len(problem.Candidates) == 0 {
		return true
	}
	if len(state.Assignments) != len(problem.Candidates) {
		return false
	}

	candidateSet := candidateByID(problem)
	for candidateID := range state.Assignments {
		if _, ok := candidateSet[candidateID]; !ok {
			return false
		}
	}
	return true
}

func IsStateViable(problem types.AllocationProblem, state types.PartialAllocationState) (bool, []types.HardConstraintViolation) {
	violations := CheckHardConstraints(problem, state)
	return len(violations) == 0, violations
}

func CheckHardConstraints(problem types.AllocationProblem, state types.PartialAllocationState) []types.HardConstraintViolation {
	candidates := candidateByID(problem)
	groups := groupByID(problem)
	groupMembers := buildStateGroupMembers(state)
	violations := make([]types.HardConstraintViolation, 0)

	for candidateID, groupID := range state.Assignments {
		if _, ok := candidates[candidateID]; !ok {
			violations = append(violations, types.HardConstraintViolation{
				Code:        "candidate_not_found",
				Message:     fmt.Sprintf("candidato %d nao existe no problema", candidateID),
				CandidateID: candidateID,
				GroupID:     groupID,
			})
		}
		if _, ok := groups[groupID]; !ok {
			violations = append(violations, types.HardConstraintViolation{
				Code:        "group_not_found",
				Message:     fmt.Sprintf("grupo %d nao existe no problema", groupID),
				CandidateID: candidateID,
				GroupID:     groupID,
			})
		}
	}

	seenCandidates := make(map[int]int)
	groupIDs := sortedGroupKeys(groupMembers)
	for _, groupID := range groupIDs {
		members := groupMembers[groupID]
		if _, ok := groups[groupID]; !ok {
			violations = append(violations, types.HardConstraintViolation{
				Code:    "group_not_found",
				Message: fmt.Sprintf("grupo %d nao existe no problema", groupID),
				GroupID: groupID,
			})
			continue
		}

		if problem.HardRestrictions.EnforceGroupCapacity && len(members) > groups[groupID].MaxCandidates {
			violations = append(violations, types.HardConstraintViolation{
				Code:    "group_capacity_exceeded",
				Message: fmt.Sprintf("grupo %d excedeu a capacidade maxima", groupID),
				GroupID: groupID,
			})
		}

		for _, candidateID := range members {
			if _, ok := candidates[candidateID]; !ok {
				violations = append(violations, types.HardConstraintViolation{
					Code:        "candidate_not_found",
					Message:     fmt.Sprintf("candidato %d nao existe no problema", candidateID),
					CandidateID: candidateID,
					GroupID:     groupID,
				})
				continue
			}

			seenCandidates[candidateID]++
			if seenCandidates[candidateID] > 1 {
				violations = append(violations, types.HardConstraintViolation{
					Code:        "candidate_duplicate",
					Message:     fmt.Sprintf("candidato %d aparece mais de uma vez no estado", candidateID),
					CandidateID: candidateID,
					GroupID:     groupID,
				})
			}

			if assignedGroupID, ok := state.Assignments[candidateID]; ok && assignedGroupID != groupID {
				violations = append(violations, types.HardConstraintViolation{
					Code:        "assignment_mismatch",
					Message:     fmt.Sprintf("candidato %d possui assignment inconsistente com groupMembers", candidateID),
					CandidateID: candidateID,
					GroupID:     groupID,
				})
			}

			if problem.HardRestrictions.RespectCandidatePreferences &&
				!containsInt(candidates[candidateID].PreferredGroupIDs, groupID) {
				violations = append(violations, types.HardConstraintViolation{
					Code:        "group_out_of_preference",
					Message:     fmt.Sprintf("grupo %d nao esta nas preferencias do candidato %d", groupID, candidateID),
					CandidateID: candidateID,
					GroupID:     groupID,
				})
			}

			if problem.HardRestrictions.EnforceForbiddenEvaluators {
				for _, evaluatorID := range groups[groupID].EvaluatorIDs {
					if containsInt(candidates[candidateID].EvaluatorRestrictions.ForbiddenEvaluatorIDs, evaluatorID) {
						violations = append(violations, types.HardConstraintViolation{
							Code:        "forbidden_evaluator",
							Message:     fmt.Sprintf("candidato %d nao pode ser avaliado por %d no grupo %d", candidateID, evaluatorID, groupID),
							CandidateID: candidateID,
							GroupID:     groupID,
							EvaluatorID: evaluatorID,
						})
					}
				}
			}
		}
	}

	if IsCompleteState(problem, state) {
		if problem.HardRestrictions.AllCandidatesMustBeAssigned {
			for _, candidate := range problem.Candidates {
				if _, ok := state.Assignments[candidate.ID]; !ok {
					violations = append(violations, types.HardConstraintViolation{
						Code:        "missing_assignment",
						Message:     fmt.Sprintf("candidato %d nao foi alocado", candidate.ID),
						CandidateID: candidate.ID,
					})
				}
			}
		}

		if problem.HardRestrictions.EnforceMinCandidatesOnCompleteState {
			for _, group := range problem.Groups {
				members := groupMembers[group.ID]
				if len(members) > 0 && len(members) < group.MinCandidates {
					violations = append(violations, types.HardConstraintViolation{
						Code:    "group_below_min_candidates",
						Message: fmt.Sprintf("grupo %d ficou abaixo do minimo de candidatos", group.ID),
						GroupID: group.ID,
					})
				}
			}
		}
	}

	return violations
}

func buildStateGroupMembers(state types.PartialAllocationState) map[int][]int {
	groupMembers := make(map[int][]int, len(state.GroupMembers)+len(state.Assignments))
	for groupID, members := range state.GroupMembers {
		groupMembers[groupID] = append([]int(nil), members...)
	}

	if len(state.Assignments) == 0 {
		return groupMembers
	}

	if len(groupMembers) == 0 {
		for candidateID, groupID := range state.Assignments {
			groupMembers[groupID] = append(groupMembers[groupID], candidateID)
		}
		for groupID := range groupMembers {
			sort.Ints(groupMembers[groupID])
		}
		return groupMembers
	}

	return groupMembers
}

func candidateByID(problem types.AllocationProblem) map[int]types.SolverCandidate {
	result := make(map[int]types.SolverCandidate, len(problem.Candidates))
	for _, candidate := range problem.Candidates {
		result[candidate.ID] = candidate
	}
	return result
}

func groupByID(problem types.AllocationProblem) map[int]types.SolverGroup {
	result := make(map[int]types.SolverGroup, len(problem.Groups))
	for _, group := range problem.Groups {
		result[group.ID] = group
	}
	return result
}

func sortedGroupKeys(groupMembers map[int][]int) []int {
	keys := make([]int, 0, len(groupMembers))
	for key := range groupMembers {
		keys = append(keys, key)
	}
	sort.Ints(keys)
	return keys
}

func containsInt(values []int, target int) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
