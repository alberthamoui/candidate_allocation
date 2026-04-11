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
