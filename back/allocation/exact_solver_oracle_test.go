package allocation

import (
	"math/rand"
	"testing"

	types "candidate_alocator/back/type"
)

func TestExactSolverMatchesExhaustiveOracleOnSmallProblems(t *testing.T) {
	// #nosec G404 - deterministic pseudo-random fixtures are intentional in tests.
	rng := rand.New(rand.NewSource(20260813))
	for iteration := 0; iteration < 80; iteration++ {
		problem := randomOracleProblem(rng)
		got := SolveAllocation(problem, SolverOptions{WorkerCount: 3, ParallelDepth: 1})
		want, feasible := exhaustiveOptimalScore(problem)

		if !feasible {
			if got.Status != "partial" || len(got.Assignments) >= len(problem.Candidates) || len(got.HardViolations) == 0 {
				t.Fatalf("iteration %d: expected an explicit partial fallback, got %#v", iteration, got)
			}
			continue
		}
		if got.Status != "optimal" || got.Score.TotalPenalty != want {
			t.Fatalf("iteration %d: exact solver diverged from oracle: got status=%s score=%d want=%d problem=%#v", iteration, got.Status, got.Score.TotalPenalty, want, problem)
		}
		if viable, violations := IsStateViable(problem, types.PartialAllocationState{Assignments: got.Assignments}); !viable {
			t.Fatalf("iteration %d: solver returned invalid assignment: %#v", iteration, violations)
		}
	}
}

func TestExactSolverWithAllNewBoundsMatchesExhaustiveOracle(t *testing.T) {
	// #nosec G404 - deterministic pseudo-random fixtures are intentional in tests.
	rng := rand.New(rand.NewSource(20260814))
	for iteration := 0; iteration < 60; iteration++ {
		problem := randomOracleProblem(rng)
		for candidateIndex := range problem.Candidates {
			value := "a"
			if rng.Intn(2) == 1 {
				value = "b"
			}
			problem.Candidates[candidateIndex].Attributes = map[string]string{"segment": value}
		}
		problem.SoftRules.Criteria = []types.SoftCriterion{
			{Type: types.SoftCriterionMinValue, ColumnKey: "segment", SelectedValues: []string{"a"}, Threshold: 2},
			{Type: types.SoftCriterionAtLeastOneEach, ColumnKey: "segment", SelectedValues: []string{"a", "b"}},
			{Type: types.SoftCriterionBalancedDistribution, ColumnKey: "segment", SelectedValues: []string{"a", "b"}},
		}

		got := SolveAllocation(problem, SolverOptions{WorkerCount: 3, ParallelDepth: 1})
		want, feasible := exhaustiveOptimalScore(problem)
		if !feasible {
			if got.Status != "partial" || len(got.Assignments) >= len(problem.Candidates) || len(got.HardViolations) == 0 {
				t.Fatalf("iteration %d: expected an explicit partial fallback, got %#v", iteration, got)
			}
			continue
		}
		if got.Status != "optimal" || got.Score.TotalPenalty != want {
			t.Fatalf("iteration %d: got status=%s score=%d want=%d problem=%#v", iteration, got.Status, got.Score.TotalPenalty, want, problem)
		}
	}
}

func randomOracleProblem(rng *rand.Rand) types.AllocationProblem {
	candidateCount := 3 + rng.Intn(4)
	groupCount := 2 + rng.Intn(2)
	groups := make([]types.SolverGroup, groupCount)
	for groupIndex := range groups {
		groups[groupIndex] = types.SolverGroup{
			ID: groupIndex + 1, ScheduleKey: "schedule-" + string(rune('a'+groupIndex)),
			MinCandidates: 1, MaxCandidates: 1 + rng.Intn(candidateCount), EvaluatorIDs: []int{groupIndex + 1},
		}
	}
	candidates := make([]types.SolverCandidate, candidateCount)
	for candidateIndex := range candidates {
		preferred := rng.Perm(groupCount)
		preferredGroupIDs := make([]int, groupCount)
		ranks := make(map[int]int, groupCount)
		for rank, groupIndex := range preferred {
			groupID := groupIndex + 1
			preferredGroupIDs[rank] = groupID
			ranks[groupID] = rank
		}
		candidate := types.SolverCandidate{
			ID: candidateIndex + 1, PreferredGroupIDs: preferredGroupIDs, PreferenceRankByGroupID: ranks,
		}
		if rng.Intn(4) == 0 {
			candidate.EvaluatorRestrictions.AvoidEvaluatorIDs = []int{1 + rng.Intn(groupCount)}
		}
		candidates[candidateIndex] = candidate
	}
	return types.AllocationProblem{
		Candidates: candidates, Groups: groups,
		HardRestrictions: types.SolverHardRestrictions{
			AllCandidatesMustBeAssigned: true, RespectCandidatePreferences: true,
			EnforceGroupCapacity: true, EnforceForbiddenEvaluators: true, EnforceMinCandidatesOnCompleteState: true,
		},
		SoftRules: types.SolverSoftRules{PreferencePenaltyByRank: []int{0, 1, 2}, AvoidEvaluatorPenalty: 3},
	}
}

func exhaustiveOptimalScore(problem types.AllocationProblem) (int, bool) {
	best := 0
	found := false
	assignments := make(map[int]int, len(problem.Candidates))
	var visit func(int)
	visit = func(candidateIndex int) {
		if candidateIndex == len(problem.Candidates) {
			state := types.PartialAllocationState{Assignments: cloneAssignments(assignments)}
			if viable, _ := IsStateViable(problem, state); !viable {
				return
			}
			score := ScoreAllocation(problem, state).TotalPenalty
			if !found || score < best {
				best, found = score, true
			}
			return
		}
		candidate := problem.Candidates[candidateIndex]
		for _, groupID := range candidate.PreferredGroupIDs {
			assignments[candidate.ID] = groupID
			visit(candidateIndex + 1)
		}
		delete(assignments, candidate.ID)
	}
	visit(0)
	return best, found
}

func BenchmarkSolveAllocationMediumExact(b *testing.B) {
	problem := makeOracleBenchmarkProblem(18, 3)
	b.ResetTimer()
	for iteration := 0; iteration < b.N; iteration++ {
		result := SolveAllocation(problem, SolverOptions{WorkerCount: 1})
		if result.Status != "optimal" {
			b.Fatalf("unexpected result: %#v", result)
		}
	}
}

func BenchmarkSolveAllocationThreeSoftCriteria(b *testing.B) {
	problem := makeSoftCriteriaBenchmarkProblem(18, 3)
	b.ReportAllocs()
	b.ResetTimer()
	for iteration := 0; iteration < b.N; iteration++ {
		result := SolveAllocation(problem, SolverOptions{WorkerCount: 1})
		if result.Status != "optimal" {
			b.Fatalf("unexpected result: %#v", result)
		}
	}
}

func makeSoftCriteriaBenchmarkProblem(candidateCount, groupCount int) types.AllocationProblem {
	problem := makeOracleBenchmarkProblem(candidateCount, groupCount)
	for index := range problem.Groups {
		problem.Groups[index].EvaluatorIDs = []int{index + 1}
	}
	for index := range problem.Candidates {
		value := "a"
		if index >= len(problem.Candidates)/2 {
			value = "b"
		}
		problem.Candidates[index].Attributes = map[string]string{"segment": value}
	}
	problem.SoftRules.Criteria = []types.SoftCriterion{
		{Type: types.SoftCriterionBalancedDistribution, ColumnKey: "segment", SelectedValues: []string{"a", "b"}},
		{Type: types.SoftCriterionAtLeastOneEach, ColumnKey: "segment", SelectedValues: []string{"a", "b"}},
		{Type: types.SoftCriterionMinValue, ColumnKey: "segment", SelectedValues: []string{"a"}, Threshold: 2},
	}
	return problem
}

func TestSolveAllocationDecomposesIndependentCandidateGroupGraphs(t *testing.T) {
	problem := types.AllocationProblem{
		Candidates: []types.SolverCandidate{
			{ID: 1, PreferredGroupIDs: []int{1}, PreferenceRankByGroupID: map[int]int{1: 0}},
			{ID: 2, PreferredGroupIDs: []int{2}, PreferenceRankByGroupID: map[int]int{2: 0}},
		},
		Groups: []types.SolverGroup{
			{ID: 1, ScheduleKey: "a", MinCandidates: 1, MaxCandidates: 1},
			{ID: 2, ScheduleKey: "b", MinCandidates: 1, MaxCandidates: 1},
		},
		HardRestrictions: types.SolverHardRestrictions{AllCandidatesMustBeAssigned: true, RespectCandidatePreferences: true, EnforceGroupCapacity: true, EnforceMinCandidatesOnCompleteState: true},
		SoftRules:        types.SolverSoftRules{PreferencePenaltyByRank: []int{0, 1}},
	}
	components := independentSolverComponents(problem)
	if len(components) != 2 {
		t.Fatalf("expected two independent components, got %#v", components)
	}
	result := SolveAllocation(problem, SolverOptions{})
	if result.Status != "optimal" || result.Assignments[1] != 1 || result.Assignments[2] != 2 {
		t.Fatalf("unexpected decomposed result: %#v", result)
	}
}

func TestSolveAllocationKeepsUnifiedProblemWhenPreferencesAreNotHard(t *testing.T) {
	problem := types.AllocationProblem{
		Candidates: []types.SolverCandidate{
			{ID: 1, PreferredGroupIDs: nil},
			{ID: 2, PreferredGroupIDs: nil},
		},
		Groups: []types.SolverGroup{
			{ID: 1, ScheduleKey: "a", MaxCandidates: 1},
			{ID: 2, ScheduleKey: "b", MaxCandidates: 1},
		},
		HardRestrictions: types.SolverHardRestrictions{
			AllCandidatesMustBeAssigned: true,
			EnforceGroupCapacity:        true,
		},
		SoftRules: types.SolverSoftRules{PreferencePenaltyByRank: []int{0, 1}},
	}

	if components := independentSolverComponents(problem); len(components) != 1 {
		t.Fatalf("expected one unified component, got %#v", components)
	}
	result := SolveAllocation(problem, SolverOptions{})
	if result.Status != "optimal" || len(result.Assignments) != 2 {
		t.Fatalf("unexpected unified result: %#v", result)
	}
}

func makeOracleBenchmarkProblem(candidateCount, groupCount int) types.AllocationProblem {
	groups := make([]types.SolverGroup, groupCount)
	groupIDs := make([]int, groupCount)
	for groupIndex := range groups {
		groupIDs[groupIndex] = groupIndex + 1
		groups[groupIndex] = types.SolverGroup{ID: groupIndex + 1, ScheduleKey: "shared", MinCandidates: candidateCount / groupCount, MaxCandidates: candidateCount / groupCount}
	}
	candidates := make([]types.SolverCandidate, candidateCount)
	for candidateIndex := range candidates {
		ranks := make(map[int]int, groupCount)
		for rank, groupID := range groupIDs {
			ranks[groupID] = rank
		}
		candidates[candidateIndex] = types.SolverCandidate{ID: candidateIndex + 1, PreferredGroupIDs: append([]int(nil), groupIDs...), PreferenceRankByGroupID: ranks}
	}
	return types.AllocationProblem{
		Candidates: candidates, Groups: groups,
		HardRestrictions: types.SolverHardRestrictions{AllCandidatesMustBeAssigned: true, RespectCandidatePreferences: true, EnforceGroupCapacity: true, EnforceMinCandidatesOnCompleteState: true},
		SoftRules:        types.SolverSoftRules{PreferencePenaltyByRank: []int{0, 1, 2}},
	}
}
