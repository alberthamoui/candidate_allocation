package allocation

import (
	types "candidate_alocator/back/type"
	"testing"
)

func TestManualReviewKeepsAssignmentsAndReportsExceptions(t *testing.T) {
	p := types.AllocationProblem{Candidates: []types.SolverCandidate{{ID: 1, PreferredGroupIDs: []int{1}, PreferenceRankByGroupID: map[int]int{1: 0}}, {ID: 2, PreferredGroupIDs: []int{1}, PreferenceRankByGroupID: map[int]int{1: 0}}}, Groups: []types.SolverGroup{{ID: 1, ScheduleKey: "sexta", MaxCandidates: 1, EvaluatorIDs: []int{10}}, {ID: 2, ScheduleKey: "sexta", MaxCandidates: 2, EvaluatorIDs: []int{10}}}, HardRestrictions: types.SolverHardRestrictions{EnforceGroupCapacity: true}, SoftRules: types.SolverSoftRules{PreferencePenaltyByRank: []int{0, 1}}}
	result := ReviewManualAllocation(p, map[int]int{1: 1, 2: 1}, 2)
	if len(result.Assignments) != 2 || result.Status != "manual" || len(result.HardViolations) < 2 {
		t.Fatal(result)
	}
	result = ReviewManualAllocation(p, map[int]int{1: 1, 2: 2}, 1)
	found := false
	for _, v := range result.HardViolations {
		found = found || v.Code == "evaluator_overlap"
	}
	if !found {
		t.Fatal("evaluator overlap absent")
	}
}
