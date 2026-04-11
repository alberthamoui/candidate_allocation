package allocation

import (
	"testing"

	types "candidate_alocator/back/type"
)

func TestIsCompleteState(t *testing.T) {
	problem := mustBuildFixtureProblem(t)

	if IsCompleteState(problem, types.PartialAllocationState{Assignments: map[int]int{1: 1, 2: 2}}) {
		t.Fatal("expected incomplete state")
	}

	if !IsCompleteState(problem, types.PartialAllocationState{Assignments: map[int]int{1: 1, 2: 2, 3: 1}}) {
		t.Fatal("expected complete state")
	}
}

func TestCheckHardConstraintsPartialViable(t *testing.T) {
	problem := mustBuildFixtureProblem(t)
	state := types.PartialAllocationState{
		Assignments: map[int]int{1: 1},
	}

	violations := CheckHardConstraints(problem, state)
	if len(violations) != 0 {
		t.Fatalf("expected viable partial state, got %#v", violations)
	}
}

func TestCheckHardConstraintsCandidateDuplicate(t *testing.T) {
	problem := mustBuildFixtureProblem(t)
	state := types.PartialAllocationState{
		Assignments: map[int]int{1: 1},
		GroupMembers: map[int][]int{
			1: []int{1},
			2: []int{1},
		},
	}

	assertHasViolationCode(t, CheckHardConstraints(problem, state), "candidate_duplicate")
}

func TestCheckHardConstraintsGroupNotFound(t *testing.T) {
	problem := mustBuildFixtureProblem(t)
	state := types.PartialAllocationState{
		Assignments: map[int]int{1: 99},
	}

	assertHasViolationCode(t, CheckHardConstraints(problem, state), "group_not_found")
}

func TestCheckHardConstraintsCandidateNotFound(t *testing.T) {
	problem := mustBuildFixtureProblem(t)
	state := types.PartialAllocationState{
		Assignments: map[int]int{99: 1},
	}

	assertHasViolationCode(t, CheckHardConstraints(problem, state), "candidate_not_found")
}

func TestCheckHardConstraintsGroupOutOfPreference(t *testing.T) {
	problem := mustBuildFixtureProblem(t)
	state := types.PartialAllocationState{
		Assignments: map[int]int{2: 3},
	}
	problem.Groups = append(problem.Groups, types.SolverGroup{ID: 3, Label: "extra", MinCandidates: 1, MaxCandidates: 2})

	assertHasViolationCode(t, CheckHardConstraints(problem, state), "group_out_of_preference")
}

func TestCheckHardConstraintsCapacityExceeded(t *testing.T) {
	problem := mustBuildFixtureProblem(t)
	state := types.PartialAllocationState{
		Assignments: map[int]int{1: 1, 2: 1, 3: 1},
	}

	assertHasViolationCode(t, CheckHardConstraints(problem, state), "group_capacity_exceeded")
}

func TestCheckHardConstraintsForbiddenEvaluator(t *testing.T) {
	problem := mustBuildFixtureProblem(t)
	state := types.PartialAllocationState{
		Assignments: map[int]int{1: 2},
	}

	assertHasViolationCode(t, CheckHardConstraints(problem, state), "forbidden_evaluator")
}

func TestCheckHardConstraintsGroupBelowMinOnComplete(t *testing.T) {
	problem := mustBuildFixtureProblem(t)
	problem.Groups[1].MinCandidates = 2
	state := types.PartialAllocationState{
		Assignments: map[int]int{1: 1, 2: 1, 3: 2},
	}

	assertHasViolationCode(t, CheckHardConstraints(problem, state), "group_below_min_candidates")
}

func assertHasViolationCode(t *testing.T, violations []types.HardConstraintViolation, code string) {
	t.Helper()

	for _, violation := range violations {
		if violation.Code == code {
			return
		}
	}

	t.Fatalf("expected violation code %q, got %#v", code, violations)
}
