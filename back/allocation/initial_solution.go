package allocation

import (
	"sort"

	types "candidate_alocator/back/type"
)

const (
	maxInitialImprovementPasses    = 32
	localImprovementScoreThreshold = 40
)

// improveInitialAssignment applies deterministic move/swap local search to the
// min-cost-flow incumbent. Group capacities and every hard assignment rule stay
// valid while the full soft objective guides each accepted change.
func improveInitialAssignment(view solverProblemView, assignment map[int]int) map[int]int {
	state := stateFromPartial(view, types.PartialAllocationState{Assignments: cloneAssignments(assignment)})
	if state.AssignedCount != len(view.sortedCandidateIDs) {
		return cloneAssignments(assignment)
	}

	currentScore := scoreCompleteSolverState(view, &state)
	if currentScore <= 0 || currentScore >= localImprovementScoreThreshold {
		return cloneAssignments(assignment)
	}
	for pass := 0; pass < maxInitialImprovementPasses; pass++ {
		targetCandidateIndexes := penalizedCandidateIndexes(view, &state)
		if len(targetCandidateIndexes) == 0 {
			break
		}
		bestScore := currentScore
		bestMoveCandidate := -1
		bestMoveGroup := -1
		bestSwapLeft := -1
		bestSwapRight := -1

		for _, candidateIndex := range targetCandidateIndexes {
			candidateID := view.sortedCandidateIDs[candidateIndex]
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

		for _, leftIndex := range targetCandidateIndexes {
			leftID := view.sortedCandidateIDs[leftIndex]
			leftGroupID := state.Assignments[leftIndex]
			for rightIndex := 0; rightIndex < len(view.sortedCandidateIDs); rightIndex++ {
				if rightIndex == leftIndex {
					continue
				}
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

// penalizedCandidateIndexes restricts local search to people related to the
// current positive score. Swap partners may be any candidate because a full
// table often requires exchanging positions instead of making a direct move.
func penalizedCandidateIndexes(view solverProblemView, state *solverState) []int {
	partial := types.PartialAllocationState{Assignments: assignmentsMap(view, state)}
	selected := make(map[int]struct{})
	for candidateIndex, candidateID := range view.sortedCandidateIDs {
		groupID := state.Assignments[candidateIndex]
		if groupID != 0 && view.assignmentPenaltyByKey[candidateID][groupID] > 0 {
			selected[candidateID] = struct{}{}
		}
	}
	for _, criterion := range view.problem.SoftRules.Criteria {
		components := scoreSoftCriterion(view.problem, partial, criterion)
		penalty := 0
		for _, component := range components {
			penalty += component.Penalty
		}
		if penalty == 0 {
			continue
		}
		candidateIDs, _, _, _, _, _ := qualitySubjectsForCriterion(view.problem, partial, criterion)
		for _, candidateID := range candidateIDs {
			selected[candidateID] = struct{}{}
		}
	}
	indexes := make([]int, 0, len(selected))
	for candidateID := range selected {
		if candidateIndex, ok := view.candidateIndexByID[candidateID]; ok {
			indexes = append(indexes, candidateIndex)
		}
	}
	sort.Ints(indexes)
	return indexes
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
				groupCounts := make([]int, len(view.sortedGroupIDs))
				for groupIndex := range view.sortedGroupIDs {
					groupCounts[groupIndex] = state.CriterionCounts[criterionIndex][groupIndex][valueIndex]
				}
				median := medianInt(groupCounts)
				for _, count := range groupCounts {
					if count >= median {
						total += count - median
					} else {
						total += median - count
					}
				}
			}
		case types.SoftCriterionGroupTogether:
			for groupIndex := range view.sortedGroupIDs {
				selectedCount := 0
				for valueIndex := range criterion.SelectedValues {
					selectedCount += state.CriterionCounts[criterionIndex][groupIndex][valueIndex]
				}
				if selectedCount > 0 && state.GroupCounts[groupIndex] > selectedCount {
					total += selectedCount
				}
			}
		}
	}
	return total
}
