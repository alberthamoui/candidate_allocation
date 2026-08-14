package allocation

import types "candidate_alocator/back/type"

const maxInitialImprovementPasses = 32

// improveInitialAssignment applies deterministic move/swap local search to the
// min-cost-flow incumbent. Group capacities and every hard assignment rule stay
// valid while the full soft objective guides each accepted change.
func improveInitialAssignment(view solverProblemView, assignment map[int]int) map[int]int {
	if len(view.criteria) == 0 {
		return cloneAssignments(assignment)
	}
	state := stateFromPartial(view, types.PartialAllocationState{Assignments: cloneAssignments(assignment)})
	if state.AssignedCount != len(view.sortedCandidateIDs) {
		return cloneAssignments(assignment)
	}

	currentScore := scoreCompleteSolverState(view, &state)
	for pass := 0; pass < maxInitialImprovementPasses; pass++ {
		bestScore := currentScore
		bestMoveCandidate := -1
		bestMoveGroup := -1
		bestSwapLeft := -1
		bestSwapRight := -1

		for candidateIndex, candidateID := range view.sortedCandidateIDs {
			currentGroupID := state.Assignments[candidateIndex]
			currentGroupIndex := view.groupIndexByID[currentGroupID]
			for _, option := range view.candidateGroupOptions[candidateID] {
				if option.GroupID == currentGroupID || !movePreservesGroupBounds(view, &state, currentGroupIndex, option.GroupIndex) {
					continue
				}
				changeStateAssignment(view, &state, candidateIndex, option.GroupID)
				score := scoreCompleteSolverState(view, &state)
				changeStateAssignment(view, &state, candidateIndex, currentGroupID)
				if score < bestScore {
					bestScore = score
					bestMoveCandidate = candidateIndex
					bestMoveGroup = option.GroupID
					bestSwapLeft, bestSwapRight = -1, -1
				}
			}
		}

		for leftIndex, leftID := range view.sortedCandidateIDs {
			leftGroupID := state.Assignments[leftIndex]
			for rightIndex := leftIndex + 1; rightIndex < len(view.sortedCandidateIDs); rightIndex++ {
				rightID := view.sortedCandidateIDs[rightIndex]
				rightGroupID := state.Assignments[rightIndex]
				if leftGroupID == rightGroupID {
					continue
				}
				if _, allowed := view.assignmentPenaltyByKey[leftID][rightGroupID]; !allowed {
					continue
				}
				if _, allowed := view.assignmentPenaltyByKey[rightID][leftGroupID]; !allowed {
					continue
				}

				changeStateAssignment(view, &state, leftIndex, rightGroupID)
				changeStateAssignment(view, &state, rightIndex, leftGroupID)
				score := scoreCompleteSolverState(view, &state)
				changeStateAssignment(view, &state, rightIndex, rightGroupID)
				changeStateAssignment(view, &state, leftIndex, leftGroupID)
				if score < bestScore {
					bestScore = score
					bestSwapLeft, bestSwapRight = leftIndex, rightIndex
					bestMoveCandidate, bestMoveGroup = -1, -1
				}
			}
		}

		if bestScore >= currentScore {
			break
		}
		if bestMoveCandidate >= 0 {
			changeStateAssignment(view, &state, bestMoveCandidate, bestMoveGroup)
		} else {
			leftGroupID := state.Assignments[bestSwapLeft]
			rightGroupID := state.Assignments[bestSwapRight]
			changeStateAssignment(view, &state, bestSwapLeft, rightGroupID)
			changeStateAssignment(view, &state, bestSwapRight, leftGroupID)
		}
		currentScore = bestScore
	}
	return assignmentsMap(view, &state)
}

func movePreservesGroupBounds(view solverProblemView, state *solverState, sourceGroupIndex, targetGroupIndex int) bool {
	if view.problem.HardRestrictions.EnforceGroupCapacity {
		targetGroupID := view.sortedGroupIDs[targetGroupIndex]
		if state.GroupCounts[targetGroupIndex] >= view.groups[targetGroupID].MaxCandidates {
			return false
		}
	}
	if view.problem.HardRestrictions.EnforceMinCandidatesOnCompleteState {
		sourceGroupID := view.sortedGroupIDs[sourceGroupIndex]
		afterMove := state.GroupCounts[sourceGroupIndex] - 1
		if afterMove != 0 && afterMove < view.groups[sourceGroupID].MinCandidates {
			return false
		}
		targetGroupID := view.sortedGroupIDs[targetGroupIndex]
		targetAfterMove := state.GroupCounts[targetGroupIndex] + 1
		if targetAfterMove < view.groups[targetGroupID].MinCandidates {
			return false
		}
	}
	return true
}

func changeStateAssignment(view solverProblemView, state *solverState, candidateIndex, targetGroupID int) {
	candidateID := view.sortedCandidateIDs[candidateIndex]
	sourceGroupID := state.Assignments[candidateIndex]
	if sourceGroupID == targetGroupID {
		return
	}
	sourceGroupIndex := view.groupIndexByID[sourceGroupID]
	targetGroupIndex := view.groupIndexByID[targetGroupID]
	updateCriterionCounts(view, state, candidateIndex, sourceGroupIndex, -1)
	updateCriterionCounts(view, state, candidateIndex, targetGroupIndex, 1)
	state.GroupCounts[sourceGroupIndex]--
	state.GroupCounts[targetGroupIndex]++
	state.BasePenalty -= view.assignmentPenaltyByKey[candidateID][sourceGroupID]
	state.BasePenalty += view.assignmentPenaltyByKey[candidateID][targetGroupID]
	state.Assignments[candidateIndex] = targetGroupID
}

func scoreCompleteSolverState(view solverProblemView, state *solverState) int {
	total := state.BasePenalty
	for criterionIndex, criterionView := range view.criteria {
		criterion := criterionView.criterion
		switch criterion.Type {
		case types.SoftCriterionMinValue:
			if len(criterion.SelectedValues) == 1 {
				for groupIndex := range view.sortedGroupIDs {
					count := state.CriterionCounts[criterionIndex][groupIndex][0]
					if count > 0 && count < criterion.Threshold {
						total += criterion.Threshold - count
					}
				}
			}
		case types.SoftCriterionMaxValue:
			if len(criterion.SelectedValues) == 1 {
				for groupIndex := range view.sortedGroupIDs {
					if count := state.CriterionCounts[criterionIndex][groupIndex][0]; count > criterion.Threshold {
						total += count - criterion.Threshold
					}
				}
			}
		case types.SoftCriterionAtLeastOneEach:
			for groupIndex := range view.sortedGroupIDs {
				for valueIndex := range criterion.SelectedValues {
					if state.CriterionCounts[criterionIndex][groupIndex][valueIndex] == 0 {
						total++
					}
				}
			}
		case types.SoftCriterionBalancedDistribution:
			for valueIndex := range criterion.SelectedValues {
				minimum, maximum := 0, 0
				for groupIndex := range view.sortedGroupIDs {
					count := state.CriterionCounts[criterionIndex][groupIndex][valueIndex]
					if groupIndex == 0 || count < minimum {
						minimum = count
					}
					if groupIndex == 0 || count > maximum {
						maximum = count
					}
				}
				total += maximum - minimum
			}
		case types.SoftCriterionGroupTogether:
			groupsWithValues := 0
			for groupIndex := range view.sortedGroupIDs {
				for valueIndex := range criterion.SelectedValues {
					if state.CriterionCounts[criterionIndex][groupIndex][valueIndex] > 0 {
						groupsWithValues++
						break
					}
				}
			}
			if groupsWithValues > 1 {
				total += groupsWithValues - 1
			}
		}
	}
	return total
}
