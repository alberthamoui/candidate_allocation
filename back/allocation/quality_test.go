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
	if got := findQualityCharacteristic(t, report, "preference_rank_5"); got.Value != 0 {
		t.Fatalf("expected empty fifth-preference characteristic, got %#v", got)
	}
	if got := findQualityCharacteristic(t, report, "avoid_evaluator"); got.Value != 1 || got.Penalty == 0 {
		t.Fatalf("unexpected avoid-evaluator characteristic: %#v", got)
	}
	if got := findQualityCharacteristic(t, report, "configured_1_max_value"); got.Value != 2 || got.Penalty != 2 {
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
	candidateIDs, groupIDs := qualitySubjectsForCriterion(problem, state, criterion)
	if len(candidateIDs) != 2 || len(groupIDs) != 1 || groupIDs[0] != 1 {
		t.Fatalf("unexpected quality subjects: candidates=%#v groups=%#v", candidateIDs, groupIDs)
	}
	if got := configuredQualityDescription(criterion); got != "Campo curso; valores adm; limite 1." {
		t.Fatalf("unexpected criterion description: %q", got)
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
