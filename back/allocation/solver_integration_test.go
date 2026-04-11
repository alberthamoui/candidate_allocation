package allocation

import (
	"testing"

	types "candidate_alocator/back/type"
)

func TestSolverCoreIntegration(t *testing.T) {
	problem := mustBuildFixtureProblem(t)

	validStateA := types.PartialAllocationState{
		Assignments: map[int]int{1: 1, 2: 2, 3: 1},
	}
	validStateB := types.PartialAllocationState{
		Assignments: map[int]int{1: 1, 2: 2, 3: 2},
	}
	invalidState := types.PartialAllocationState{
		Assignments: map[int]int{1: 2, 2: 2, 3: 1},
	}

	if viable, violations := IsStateViable(problem, validStateA); !viable || len(violations) != 0 {
		t.Fatalf("expected validStateA to be hard-valid, got %#v", violations)
	}
	if viable, violations := IsStateViable(problem, validStateB); !viable || len(violations) != 0 {
		t.Fatalf("expected validStateB to be hard-valid, got %#v", violations)
	}
	if viable, violations := IsStateViable(problem, invalidState); viable {
		t.Fatalf("expected invalidState to fail, got viable with %#v", violations)
	} else {
		assertHasViolationCode(t, violations, "forbidden_evaluator")
	}

	scoreA := ScoreAllocation(problem, validStateA)
	scoreB := ScoreAllocation(problem, validStateB)
	if scoreA.TotalPenalty == scoreB.TotalPenalty {
		t.Fatalf("expected different penalties, got A=%#v B=%#v", scoreA, scoreB)
	}
	assertHasScoreComponent(t, scoreA, "soft_balanced_distribution")
	assertHasScoreComponent(t, scoreB, "avoid_evaluator")
}

func mustBuildFixtureProblem(t *testing.T) types.AllocationProblem {
	t.Helper()

	config, candidatos, avaliadores, restricoes := makeSolverFixture()
	problem, err := BuildAllocationProblem(config, candidatos, avaliadores, restricoes)
	if err != nil {
		t.Fatalf("BuildAllocationProblem returned error: %v", err)
	}
	return problem
}
