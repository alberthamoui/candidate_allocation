package allocation

import (
	"testing"

	types "candidate_alocator/back/type"
)

func TestBuildAllocationQualityReportIncludesBaseAndConfiguredResults(t *testing.T) {
	problem := mustBuildFixtureProblem(t)
	problem.SoftRules.PreferencePenaltyByRank = []int{0, 1, 2, 3, 4}
	problem.SoftRules.Criteria = []types.SoftCriterion{
		{Type: types.SoftCriterionMaxValue, ColumnKey: "curso", SelectedValues: []string{"adm"}, Threshold: 0},
	}
	result := types.SolverResult{
		Assignments: map[int]int{1: 1, 2: 2, 3: 2},
	}

	report := BuildAllocationQualityReport(problem, result)
	if len(report.Characteristics) != 7 {
		t.Fatalf("expected five ranks, avoid evaluator and configured criterion, got %#v", report.Characteristics)
	}
	if got := findQualityCharacteristic(t, report, "preference_rank_1"); got.Value != 2 || len(got.CandidateIDs) != 2 {
		t.Fatalf("unexpected first-preference characteristic: %#v", got)
	}
	if got := findQualityCharacteristic(t, report, "preference_rank_1"); len(got.PenalizedCandidateIDs) != 0 || len(got.NonPenalizedCandidateIDs) != 2 {
		t.Fatalf("first-preference candidates should be classified as non-penalized: %#v", got)
	}
	if got := findQualityCharacteristic(t, report, "preference_rank_5"); got.Value != 0 {
		t.Fatalf("expected empty fifth-preference characteristic, got %#v", got)
	}
	if got := findQualityCharacteristic(t, report, "avoid_evaluator"); got.Value != 1 || got.Penalty == 0 {
		t.Fatalf("unexpected avoid-evaluator characteristic: %#v", got)
	}
	if got := findQualityCharacteristic(t, report, "configured_1_max_value"); got.Value != 2 || got.Penalty != 2 || len(got.PenalizedCandidateIDs) != 2 || len(got.NonPenalizedCandidateIDs) != 0 {
		t.Fatalf("unexpected configured characteristic: %#v", got)
	}
}

func TestQualityHelpersProduceStableLabelsTonesAndSubjects(t *testing.T) {
	if got := pluralizeQualityValue(1, "item", "itens"); got != "item" {
		t.Fatalf("unexpected singular label: %q", got)
	}
	if got := pluralizeQualityValue(2, "item", "itens"); got != "itens" {
		t.Fatalf("unexpected plural label: %q", got)
	}
	if got := preferenceQualityTone(0); got != "success" {
		t.Fatalf("unexpected first-rank tone: %q", got)
	}
	if got := preferenceQualityTone(1); got != "neutral" {
		t.Fatalf("unexpected second-rank tone: %q", got)
	}
	if got := preferenceQualityTone(2); got != "warning" {
		t.Fatalf("unexpected later-rank tone: %q", got)
	}
	if got := zeroIsSuccessQualityTone(0); got != "success" {
		t.Fatalf("unexpected zero tone: %q", got)
	}
	if got := zeroIsSuccessQualityTone(2); got != "warning" {
		t.Fatalf("unexpected non-zero tone: %q", got)
	}
	gotIDs := uniqueSortedQualityIDs([]int{3, 1, 3, 2})
	if len(gotIDs) != 3 || gotIDs[0] != 1 || gotIDs[2] != 3 {
		t.Fatalf("unexpected normalized ids: %#v", gotIDs)
	}

	problem := mustBuildFixtureProblem(t)
	state := types.PartialAllocationState{Assignments: map[int]int{1: 1, 2: 2, 3: 1}}
	criterion := types.SoftCriterion{Type: types.SoftCriterionMaxValue, ColumnKey: "curso", SelectedValues: []string{"adm"}, Threshold: 1}
	candidateIDs, penalizedCandidateIDs, nonPenalizedCandidateIDs, groupIDs, penalizedGroupIDs, nonPenalizedGroupIDs := qualitySubjectsForCriterion(problem, state, criterion)
	if len(candidateIDs) != 2 || len(penalizedCandidateIDs) != 2 || len(nonPenalizedCandidateIDs) != 0 || len(groupIDs) != 1 || groupIDs[0] != 1 || len(penalizedGroupIDs) != 1 || len(nonPenalizedGroupIDs) != 0 {
		t.Fatalf("unexpected quality subjects: candidates=%#v penalized=%#v nonPenalized=%#v groups=%#v penalizedGroups=%#v nonPenalizedGroups=%#v", candidateIDs, penalizedCandidateIDs, nonPenalizedCandidateIDs, groupIDs, penalizedGroupIDs, nonPenalizedGroupIDs)
	}
	criterion.Threshold = 2
	_, penalizedCandidateIDs, nonPenalizedCandidateIDs, _, penalizedGroupIDs, nonPenalizedGroupIDs = qualitySubjectsForCriterion(problem, state, criterion)
	if len(penalizedCandidateIDs) != 0 || len(nonPenalizedCandidateIDs) != 2 || len(penalizedGroupIDs) != 0 || len(nonPenalizedGroupIDs) != 1 {
		t.Fatalf("non-violating subjects must remain visually distinguishable: penalized=%#v nonPenalized=%#v penalizedGroups=%#v nonPenalizedGroups=%#v", penalizedCandidateIDs, nonPenalizedCandidateIDs, penalizedGroupIDs, nonPenalizedGroupIDs)
	}
	if got := configuredQualityDescription(criterion); got != "Campo curso; valores adm; limite 2." {
		t.Fatalf("unexpected criterion description: %q", got)
	}
}

func TestQualitySubjectsDistinguishSoftPenaltyContributors(t *testing.T) {
	problem := types.AllocationProblem{
		Candidates: []types.SolverCandidate{
			{ID: 1, Attributes: map[string]string{"curso": "adm"}},
			{ID: 2, Attributes: map[string]string{"curso": "adm"}},
			{ID: 3, Attributes: map[string]string{"curso": "adm"}},
			{ID: 4, Attributes: map[string]string{"curso": "eco"}},
		},
		Groups: []types.SolverGroup{{ID: 1}, {ID: 2}},
	}
	state := types.PartialAllocationState{Assignments: map[int]int{1: 1, 2: 1, 3: 2, 4: 2}}
	tests := []struct {
		name                   string
		criterion              types.SoftCriterion
		penalizedCandidates    int
		nonPenalizedCandidates int
		penalizedGroups        int
		nonPenalizedGroups     int
	}{
		{name: "minimum", criterion: types.SoftCriterion{Type: types.SoftCriterionMinValue, ColumnKey: "curso", SelectedValues: []string{"adm"}, Threshold: 2}, penalizedCandidates: 1, nonPenalizedCandidates: 2, penalizedGroups: 1, nonPenalizedGroups: 1},
		{name: "maximum", criterion: types.SoftCriterion{Type: types.SoftCriterionMaxValue, ColumnKey: "curso", SelectedValues: []string{"adm"}, Threshold: 1}, penalizedCandidates: 2, nonPenalizedCandidates: 1, penalizedGroups: 1, nonPenalizedGroups: 1},
		{name: "at least one", criterion: types.SoftCriterion{Type: types.SoftCriterionAtLeastOneEach, ColumnKey: "curso", SelectedValues: []string{"adm", "eco"}}, penalizedCandidates: 0, nonPenalizedCandidates: 4, penalizedGroups: 1, nonPenalizedGroups: 1},
		{name: "balanced", criterion: types.SoftCriterion{Type: types.SoftCriterionBalancedDistribution, ColumnKey: "curso", SelectedValues: []string{"adm"}}, penalizedCandidates: 2, nonPenalizedCandidates: 1, penalizedGroups: 1, nonPenalizedGroups: 1},
		{name: "together", criterion: types.SoftCriterion{Type: types.SoftCriterionGroupTogether, ColumnKey: "curso", SelectedValues: []string{"adm"}}, penalizedCandidates: 3, nonPenalizedCandidates: 0, penalizedGroups: 2, nonPenalizedGroups: 0},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, penalizedCandidates, nonPenalizedCandidates, _, penalizedGroups, nonPenalizedGroups := qualitySubjectsForCriterion(problem, state, test.criterion)
			if len(penalizedCandidates) != test.penalizedCandidates || len(nonPenalizedCandidates) != test.nonPenalizedCandidates || len(penalizedGroups) != test.penalizedGroups || len(nonPenalizedGroups) != test.nonPenalizedGroups {
				t.Fatalf("unexpected classification: penalizedCandidates=%#v nonPenalizedCandidates=%#v penalizedGroups=%#v nonPenalizedGroups=%#v", penalizedCandidates, nonPenalizedCandidates, penalizedGroups, nonPenalizedGroups)
			}
		})
	}
}

func findQualityCharacteristic(t *testing.T, report types.AllocationQualityReport, code string) types.AllocationQualityCharacteristic {
	t.Helper()
	for _, characteristic := range report.Characteristics {
		if characteristic.Code == code {
			return characteristic
		}
	}
	t.Fatalf("quality characteristic %q not found in %#v", code, report.Characteristics)
	return types.AllocationQualityCharacteristic{}
}
