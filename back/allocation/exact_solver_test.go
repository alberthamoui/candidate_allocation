package allocation

import (
	"reflect"
	"testing"

	types "candidate_alocator/back/type"
)

func TestNormalizeSolverOptions(t *testing.T) {
	got := NormalizeSolverOptions(SolverOptions{WorkerCount: 4})
	if got.WorkerCount != 4 {
		t.Fatalf("unexpected worker count: %#v", got)
	}
	if got.ParallelDepth != 1 {
		t.Fatalf("expected automatic parallel depth 1, got %#v", got)
	}
	if got.MaxDebugEvents != defaultMaxDebugEvents {
		t.Fatalf("expected default max debug events, got %#v", got)
	}

	got = NormalizeSolverOptions(SolverOptions{WorkerCount: -1, ParallelDepth: -2, MaxDebugEvents: -3})
	if got.WorkerCount != 1 || got.ParallelDepth != 0 || got.MaxDebugEvents != defaultMaxDebugEvents {
		t.Fatalf("unexpected normalized options: %#v", got)
	}
}

func TestEstimateLowerBoundIsOptimistic(t *testing.T) {
	problem := makeBoundPruneProblem()
	partial := types.PartialAllocationState{Assignments: map[int]int{1: 2}}
	complete := types.PartialAllocationState{Assignments: map[int]int{1: 2, 2: 1}}

	lowerBound := EstimateLowerBound(problem, partial)
	fullScore := ScoreAllocation(problem, complete)
	if lowerBound > fullScore.TotalPenalty {
		t.Fatalf("expected optimistic lower bound, got lower=%d full=%d", lowerBound, fullScore.TotalPenalty)
	}
	if lowerBound != 10 {
		t.Fatalf("expected lower bound 10, got %d", lowerBound)
	}
}

func TestSolveAllocationFindsKnownOptimalSolution(t *testing.T) {
	problem := mustBuildFixtureProblem(t)
	result := SolveAllocation(problem, SolverOptions{})

	if result.Status != "optimal" {
		t.Fatalf("expected optimal result, got %#v", result)
	}
	expectedAssignments := map[int]int{1: 1, 2: 2, 3: 1}
	if !reflect.DeepEqual(result.Assignments, expectedAssignments) {
		t.Fatalf("unexpected assignments: got %#v want %#v", result.Assignments, expectedAssignments)
	}
	expectedScore := ScoreAllocation(problem, types.PartialAllocationState{Assignments: expectedAssignments})
	if result.Score.TotalPenalty != expectedScore.TotalPenalty {
		t.Fatalf("unexpected score: got %#v want %#v", result.Score, expectedScore)
	}
	if result.Metrics.NodesVisited == 0 || result.Metrics.CompleteStates == 0 {
		t.Fatalf("expected populated metrics, got %#v", result.Metrics)
	}
	if len(result.DebugNotes) == 0 {
		t.Fatalf("expected debug notes, got %#v", result.DebugNotes)
	}
}

func TestSolveAllocationReturnsExplicitInfeasibleResult(t *testing.T) {
	problem := makeInfeasibleMinGroupProblem()
	result := SolveAllocation(problem, SolverOptions{})

	if result.Status != "infeasible" {
		t.Fatalf("expected infeasible result, got %#v", result)
	}
	if len(result.Assignments) != 0 {
		t.Fatalf("expected no assignments for infeasible problem, got %#v", result.Assignments)
	}
	if result.RejectionReason == "" {
		t.Fatalf("expected rejection reason, got %#v", result)
	}
	assertHasViolationCode(t, result.HardViolations, "group_below_min_candidates")
	if result.Metrics.NodesPrunedByHard == 0 {
		t.Fatalf("expected hard-pruned nodes, got %#v", result.Metrics)
	}
}

func TestSolveAllocationPrunesByBound(t *testing.T) {
	problem := makeBoundPruneProblem()
	result := SolveAllocation(problem, SolverOptions{})

	if result.Status != "optimal" {
		t.Fatalf("expected optimal result, got %#v", result)
	}
	expectedAssignments := map[int]int{1: 1, 2: 2}
	if !reflect.DeepEqual(result.Assignments, expectedAssignments) {
		t.Fatalf("unexpected assignments: got %#v want %#v", result.Assignments, expectedAssignments)
	}
	if result.Metrics.NodesPrunedByBound == 0 {
		t.Fatalf("expected bound pruning, got %#v", result.Metrics)
	}
	if result.Metrics.CompleteStates != 1 {
		t.Fatalf("expected only one complete state after bound pruning, got %#v", result.Metrics)
	}
}

func TestSolveAllocationParallelMatchesSequential(t *testing.T) {
	problem := makeBoundPruneProblem()
	sequential := SolveAllocation(problem, SolverOptions{WorkerCount: 1, ParallelDepth: 0})
	parallel := SolveAllocation(problem, SolverOptions{WorkerCount: 3, ParallelDepth: 1})

	if sequential.Status != parallel.Status {
		t.Fatalf("status mismatch: sequential=%#v parallel=%#v", sequential, parallel)
	}
	if !reflect.DeepEqual(sequential.Assignments, parallel.Assignments) {
		t.Fatalf("assignment mismatch: sequential=%#v parallel=%#v", sequential.Assignments, parallel.Assignments)
	}
	if sequential.Score.TotalPenalty != parallel.Score.TotalPenalty {
		t.Fatalf("score mismatch: sequential=%#v parallel=%#v", sequential.Score, parallel.Score)
	}
	if parallel.Metrics.ParallelTasks <= 1 {
		t.Fatalf("expected parallel task split, got %#v", parallel.Metrics)
	}
}

func TestSolveAllocationChoosesLowerSoftScore(t *testing.T) {
	problem := mustBuildFixtureProblem(t)
	bestState := types.PartialAllocationState{Assignments: map[int]int{1: 1, 2: 2, 3: 1}}
	worseState := types.PartialAllocationState{Assignments: map[int]int{1: 1, 2: 2, 3: 2}}

	bestScore := ScoreAllocation(problem, bestState)
	worseScore := ScoreAllocation(problem, worseState)
	if bestScore.TotalPenalty >= worseScore.TotalPenalty {
		t.Fatalf("fixture must keep one solution strictly better: best=%#v worse=%#v", bestScore, worseScore)
	}

	result := SolveAllocation(problem, SolverOptions{})
	if !reflect.DeepEqual(result.Assignments, bestState.Assignments) {
		t.Fatalf("expected solver to choose lower score assignment, got %#v", result.Assignments)
	}
	if result.Score.TotalPenalty != bestScore.TotalPenalty {
		t.Fatalf("expected best score %#v, got %#v", bestScore, result.Score)
	}
}

func TestSolveAllocationWithFourCandidatesAndTwoGroups(t *testing.T) {
	problem := makeFourCandidateTwoGroupProblem()
	result := SolveAllocation(problem, SolverOptions{})

	if result.Status != "optimal" {
		t.Fatalf("expected optimal result, got %#v", result)
	}
	if len(result.Assignments) != 4 {
		t.Fatalf("expected all 4 candidates assigned, got %#v", result.Assignments)
	}

	groupCounts := map[int]int{}
	for candidateID, groupID := range result.Assignments {
		if candidateID < 1 || candidateID > 4 {
			t.Fatalf("unexpected candidate id in result: %d", candidateID)
		}
		groupCounts[groupID]++
	}

	if groupCounts[1] < 1 || groupCounts[1] > 3 {
		t.Fatalf("group 1 out of bounds: %#v", groupCounts)
	}
	if groupCounts[2] < 1 || groupCounts[2] > 3 {
		t.Fatalf("group 2 out of bounds: %#v", groupCounts)
	}

	t.Logf("assignments=%#v score=%#v metrics=%#v", result.Assignments, result.Score, result.Metrics)
}

func makeBoundPruneProblem() types.AllocationProblem {
	return types.AllocationProblem{
		Candidates: []types.SolverCandidate{
			{ID: 1, Name: "Alice", PreferredGroupIDs: []int{1, 2}},
			{ID: 2, Name: "Bruno", PreferredGroupIDs: []int{2, 1}},
		},
		Groups: []types.SolverGroup{
			{ID: 1, Label: "g1", MinCandidates: 1, MaxCandidates: 1},
			{ID: 2, Label: "g2", MinCandidates: 1, MaxCandidates: 1},
		},
		HardRestrictions: types.SolverHardRestrictions{
			AllCandidatesMustBeAssigned:         true,
			RespectCandidatePreferences:         true,
			EnforceGroupCapacity:                true,
			EnforceForbiddenEvaluators:          true,
			EnforceMinCandidatesOnCompleteState: true,
		},
		SoftRules: types.SolverSoftRules{
			PreferencePenaltyByRank: []int{0, 10},
			AvoidEvaluatorPenalty:   3,
		},
	}
}

func makeFourCandidateTwoGroupProblem() types.AllocationProblem {
	return types.AllocationProblem{
		Candidates: []types.SolverCandidate{
			{ID: 1, Name: "Amanda", PreferredGroupIDs: []int{1, 2}},
			{ID: 2, Name: "Luis", PreferredGroupIDs: []int{1, 2}},
			{ID: 3, Name: "Bernardo", PreferredGroupIDs: []int{1, 2}},
			{ID: 4, Name: "A-Line", PreferredGroupIDs: []int{1, 2}},
		},
		Groups: []types.SolverGroup{
			{ID: 1, Label: "g1", MinCandidates: 1, MaxCandidates: 3},
			{ID: 2, Label: "g2", MinCandidates: 1, MaxCandidates: 3},
		},
		HardRestrictions: types.SolverHardRestrictions{
			AllCandidatesMustBeAssigned:         true,
			RespectCandidatePreferences:         true,
			EnforceGroupCapacity:                true,
			EnforceForbiddenEvaluators:          true,
			EnforceMinCandidatesOnCompleteState: true,
		},
		SoftRules: types.SolverSoftRules{
			PreferencePenaltyByRank: []int{0, 0},
			AvoidEvaluatorPenalty:   0,
		},
	}
}

func makeInfeasibleMinGroupProblem() types.AllocationProblem {
	return types.AllocationProblem{
		Candidates: []types.SolverCandidate{
			{ID: 1, Name: "Alice", PreferredGroupIDs: []int{1}},
		},
		Groups: []types.SolverGroup{
			{ID: 1, Label: "g1", MinCandidates: 2, MaxCandidates: 2},
		},
		HardRestrictions: types.SolverHardRestrictions{
			AllCandidatesMustBeAssigned:         true,
			RespectCandidatePreferences:         true,
			EnforceGroupCapacity:                true,
			EnforceForbiddenEvaluators:          true,
			EnforceMinCandidatesOnCompleteState: true,
		},
		SoftRules: types.SolverSoftRules{
			PreferencePenaltyByRank: []int{0, 1},
			AvoidEvaluatorPenalty:   3,
		},
	}
}
