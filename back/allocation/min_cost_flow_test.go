package allocation

import (
	"testing"

	types "candidate_alocator/back/type"
)

func TestMinCostCompletionRespectsSharedGroupCapacity(t *testing.T) {
	problem := makeBoundPruneProblem()
	view := buildSolverProblemView(problem)
	state := newSolverState(view)

	cost, assignment, feasible := minCostCompletion(view, &state)
	if !feasible {
		t.Fatal("expected a capacitated completion")
	}
	if cost != 0 {
		t.Fatalf("expected the two first preferences to fit with cost zero, got %d", cost)
	}
	if assignment[1] != 1 || assignment[2] != 2 {
		t.Fatalf("unexpected min-cost assignment: %#v", assignment)
	}
}

func TestMinCostCompletionDetectsInsufficientCapacity(t *testing.T) {
	problem := makeBoundPruneProblem()
	problem.Groups[0].MaxCandidates = 0
	problem.Groups[1].MaxCandidates = 1
	view := buildSolverProblemView(problem)
	state := newSolverState(view)

	if _, _, feasible := minCostCompletion(view, &state); feasible {
		t.Fatal("expected insufficient capacity to be infeasible")
	}
}

func TestInitialMinCostAssignmentRepairsUsedGroupsBelowMinimum(t *testing.T) {
	problem := makeOracleBenchmarkProblem(18, 3)
	problem.Groups[0].MaxCandidates = 8
	problem.Groups[1].MaxCandidates = 8
	problem.Groups[2].MaxCandidates = 8
	problem.Groups[0].MinCandidates = 4
	problem.Groups[1].MinCandidates = 4
	problem.Groups[2].MinCandidates = 4
	view := buildSolverProblemView(problem)
	state := newSolverState(view)

	assignment, feasible := initialMinCostAssignment(view, &state)
	if !feasible {
		t.Fatal("expected heuristic repair to produce a valid minimum-size distribution")
	}
	partial := types.PartialAllocationState{Assignments: assignment}
	if viable, violations := IsStateViable(problem, partial); !viable {
		t.Fatalf("expected repaired assignment to be viable, got %#v", violations)
	}
}
