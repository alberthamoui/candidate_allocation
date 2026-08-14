package allocation

import (
	"testing"

	types "candidate_alocator/back/type"
)

func TestPreferencePenaltyForRank(t *testing.T) {
	if got := preferencePenaltyForRank([]int{0, 1, 2}, 1); got != 1 {
		t.Fatalf("unexpected penalty for rank 1: %d", got)
	}
	if got := preferencePenaltyForRank([]int{0, 1, 2}, 5); got != 2 {
		t.Fatalf("unexpected fallback penalty: %d", got)
	}
}

func TestScoreAllocationPreferencePenalty(t *testing.T) {
	problem := mustBuildFixtureProblem(t)
	state := types.PartialAllocationState{
		Assignments: map[int]int{1: 1, 2: 2, 3: 2},
	}

	score := ScoreAllocation(problem, state)
	assertHasScoreComponent(t, score, "preference_rank")
	if score.TotalPenalty <= 0 {
		t.Fatalf("expected positive preference penalty, got %#v", score)
	}
}

func TestScoreAllocationAvoidEvaluatorPenalty(t *testing.T) {
	problem := mustBuildFixtureProblem(t)
	state := types.PartialAllocationState{
		Assignments: map[int]int{1: 1, 2: 2, 3: 2},
	}

	score := ScoreAllocation(problem, state)
	assertHasScoreComponent(t, score, "avoid_evaluator")
}

func TestScoreMinValueCriterion(t *testing.T) {
	problem := mustBuildFixtureProblem(t)
	problem.SoftRules.Criteria = []types.SoftCriterion{
		{Type: types.SoftCriterionMinValue, ColumnKey: "curso", SelectedValues: []string{"adm"}, Threshold: 2},
	}
	state := types.PartialAllocationState{
		Assignments: map[int]int{1: 1, 2: 2, 3: 2},
	}

	score := ScoreAllocation(problem, state)
	assertHasScoreComponent(t, score, "soft_min_value")
}

func TestScoreMaxValueCriterion(t *testing.T) {
	problem := mustBuildFixtureProblem(t)
	problem.SoftRules.Criteria = []types.SoftCriterion{
		{Type: types.SoftCriterionMaxValue, ColumnKey: "curso", SelectedValues: []string{"adm"}, Threshold: 1},
	}
	state := types.PartialAllocationState{
		Assignments: map[int]int{1: 1, 2: 2, 3: 1},
	}

	score := ScoreAllocation(problem, state)
	assertHasScoreComponent(t, score, "soft_max_value")
}

func TestScoreAtLeastOneEachCriterion(t *testing.T) {
	problem := mustBuildFixtureProblem(t)
	problem.SoftRules.Criteria = []types.SoftCriterion{
		{Type: types.SoftCriterionAtLeastOneEach, ColumnKey: "curso", SelectedValues: []string{"adm", "eco"}},
	}
	state := types.PartialAllocationState{
		Assignments: map[int]int{1: 1, 2: 2, 3: 1},
	}

	score := ScoreAllocation(problem, state)
	assertHasScoreComponent(t, score, "soft_at_least_one_each")
}

func TestScoreBalancedDistributionCriterion(t *testing.T) {
	problem := mustBuildFixtureProblem(t)
	problem.SoftRules.Criteria = []types.SoftCriterion{
		{Type: types.SoftCriterionBalancedDistribution, ColumnKey: "curso", SelectedValues: []string{"adm"}},
	}
	state := types.PartialAllocationState{
		Assignments: map[int]int{1: 1, 2: 2, 3: 1},
	}

	score := ScoreAllocation(problem, state)
	assertHasScoreComponent(t, score, "soft_balanced_distribution")
}

func TestScoreBalancedDistributionUsesMedianAbsoluteDeviation(t *testing.T) {
	candidates := make([]types.SolverCandidate, 7)
	assignments := make(map[int]int, 7)
	groupAssignments := []int{2, 3, 3, 4, 4, 4, 5}
	for index, groupID := range groupAssignments {
		candidateID := index + 1
		candidates[index] = types.SolverCandidate{ID: candidateID, Attributes: map[string]string{"semestre": "1"}}
		assignments[candidateID] = groupID
	}
	problem := types.AllocationProblem{
		Candidates: candidates,
		Groups:     []types.SolverGroup{{ID: 1}, {ID: 2}, {ID: 3}, {ID: 4}, {ID: 5}},
		SoftRules: types.SolverSoftRules{Criteria: []types.SoftCriterion{
			{Type: types.SoftCriterionBalancedDistribution, ColumnKey: "semestre", SelectedValues: []string{"1"}},
		}},
	}

	score := ScoreAllocation(problem, types.PartialAllocationState{Assignments: assignments})
	if got := scorePenaltyForCode(score, "soft_balanced_distribution"); got != 4 {
		t.Fatalf("expected median deviation |0-1|+|1-1|+|2-1|+|3-1|+|1-1|=4, got %d", got)
	}
}

func TestMedianInt(t *testing.T) {
	if got := medianInt([]int{0, 1, 2, 3, 1}); got != 1 {
		t.Fatalf("unexpected odd median: %d", got)
	}
	if got := medianInt([]int{0, 1, 2, 3}); got != 2 {
		t.Fatalf("expected deterministic upper median for even input, got %d", got)
	}
}

func TestScoreGroupTogetherCriterion(t *testing.T) {
	problem := mustBuildFixtureProblem(t)
	problem.SoftRules.Criteria = []types.SoftCriterion{
		{Type: types.SoftCriterionGroupTogether, ColumnKey: "curso", SelectedValues: []string{"adm"}},
	}
	state := types.PartialAllocationState{
		Assignments: map[int]int{1: 1, 2: 2, 3: 2},
	}

	score := ScoreAllocation(problem, state)
	assertHasScoreComponent(t, score, "soft_group_together")
	if got := scorePenaltyForCode(score, "soft_group_together"); got != 1 {
		t.Fatalf("expected one selected person in a mixed group, got penalty %d", got)
	}
}

func TestScoreGroupTogetherCountsSelectedPeopleInMixedGroups(t *testing.T) {
	problem := types.AllocationProblem{
		Candidates: []types.SolverCandidate{
			{ID: 1, Attributes: map[string]string{"curso": "adm"}},
			{ID: 2, Attributes: map[string]string{"curso": "eco"}},
			{ID: 3, Attributes: map[string]string{"curso": "direito"}},
			{ID: 4, Attributes: map[string]string{"curso": "adm"}},
			{ID: 5, Attributes: map[string]string{"curso": "adm"}},
		},
		Groups: []types.SolverGroup{{ID: 1}, {ID: 2}},
		SoftRules: types.SolverSoftRules{Criteria: []types.SoftCriterion{
			{Type: types.SoftCriterionGroupTogether, ColumnKey: "curso", SelectedValues: []string{"adm"}},
		}},
	}
	state := types.PartialAllocationState{Assignments: map[int]int{1: 1, 2: 1, 3: 1, 4: 2, 5: 2}}

	score := ScoreAllocation(problem, state)
	if got := scorePenaltyForCode(score, "soft_group_together"); got != 1 {
		t.Fatalf("expected only the selected person in the mixed group to be penalized, got %d in %#v", got, score)
	}

	pureState := types.PartialAllocationState{Assignments: map[int]int{1: 1, 2: 2, 3: 2, 4: 1, 5: 1}}
	pureScore := ScoreAllocation(problem, pureState)
	if got := scorePenaltyForCode(pureScore, "soft_group_together"); got != 0 {
		t.Fatalf("expected no penalty when selected candidates share only with selected candidates, got %d", got)
	}
}

func TestScoreAllocationInvalidState(t *testing.T) {
	problem := mustBuildFixtureProblem(t)
	state := types.PartialAllocationState{
		Assignments: map[int]int{1: 2, 2: 2, 3: 2},
	}

	score := ScoreAllocation(problem, state)
	assertHasScoreComponent(t, score, "invalid_state")
	if score.TotalPenalty != invalidStatePenalty {
		t.Fatalf("unexpected invalid penalty: %#v", score)
	}
}

func assertHasScoreComponent(t *testing.T, score types.SoftScoreBreakdown, code string) {
	t.Helper()

	for _, component := range score.Components {
		if component.Code == code {
			return
		}
	}

	t.Fatalf("expected score component %q, got %#v", code, score.Components)
}

func scorePenaltyForCode(score types.SoftScoreBreakdown, code string) int {
	penalty := 0
	for _, component := range score.Components {
		if component.Code == code {
			penalty += component.Penalty
		}
	}
	return penalty
}
