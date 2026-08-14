package main

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"candidate_alocator/back/allocation"
	"candidate_alocator/back/logic"
	types "candidate_alocator/back/type"
)

func TestAppDetectUniquePreferenceValuesDelegates(t *testing.T) {
	app := NewApp()
	candidatos := []types.Candidato{{Opcoes: []string{"Seg 10h", "Ter 14h"}}}

	got := app.DetectUniquePreferenceValues(candidatos)

	expected := []types.UniqueValueDetection{
		{ValorOriginal: "Seg 10h", ValorNormalizado: "seg 10h", Ocorrencias: 1},
		{ValorOriginal: "Ter 14h", ValorNormalizado: "ter 14h", Ocorrencias: 1},
	}

	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("unexpected detections: %#v", got)
	}
}

func TestAppCriterionHelpersDelegate(t *testing.T) {
	app := NewApp()
	candidatos := []types.Candidato{
		{Nome: "Maria", Curso: "ADM"},
		{Nome: "Ana", Curso: "ECO"},
	}

	columns := app.ListCandidateCriterionColumns(candidatos)
	if len(columns) == 0 {
		t.Fatal("expected criterion columns to be listed")
	}

	values, err := app.DetectUniqueCandidateColumnValues(candidatos, "curso")
	if err != nil {
		t.Fatalf("DetectUniqueCandidateColumnValues returned error: %v", err)
	}
	if len(values) != 2 {
		t.Fatalf("expected 2 unique values, got %d", len(values))
	}

	normalized := app.NormalizeSoftCriteria([]types.SoftCriterion{
		{Type: " MIN_VALUE ", ColumnKey: "curso", SelectedValues: []string{" ADM ", "adm"}, Threshold: 2},
	})
	expectedNormalized := []types.SoftCriterion{
		{Type: types.SoftCriterionMinValue, ColumnKey: "curso", SelectedValues: []string{"adm"}, Threshold: 2},
	}
	if !reflect.DeepEqual(normalized, expectedNormalized) {
		t.Fatalf("unexpected normalized criteria: %#v", normalized)
	}

	if err := app.ValidatePreferenceScheduleMappings([]types.PreferenceScheduleMapping{
		{ValorPreferencia: "seg 10h", Dia: "segunda", Hora: "10:00"},
	}); err != nil {
		t.Fatalf("expected valid preference mappings, got %v", err)
	}

	if err := app.ValidateSoftCriteria([]types.SoftCriterion{
		{Type: types.SoftCriterionMinValue, ColumnKey: "curso", SelectedValues: []string{"adm"}, Threshold: 1},
	}, candidatos); err != nil {
		t.Fatalf("expected valid soft criteria, got %v", err)
	}

	if err := app.ValidateAllocationParams(types.AllocationParams{
		GruposPorHorario:    2,
		MinPessoasPorGrupo:  4,
		MaxPessoasPorGrupo:  8,
		AvaliadoresPorGrupo: 3,
		SoftCriteria: []types.SoftCriterion{
			{Type: types.SoftCriterionBalancedDistribution, ColumnKey: "curso", SelectedValues: []string{"adm", "eco"}},
		},
	}, candidatos); err != nil {
		t.Fatalf("expected valid allocation params, got %v", err)
	}

	if total := app.CountPossibleAllocationQuantities(types.AllocationParams{
		GruposPorHorario:   2,
		MinPessoasPorGrupo: 2,
		MaxPessoasPorGrupo: 3,
	}, 5); total != 10 {
		t.Fatalf("expected 10 possible allocations, got %d", total)
	}

	if total := app.CountPossibleAllocationQuantitiesAcrossSchedules(types.AllocationParams{
		GruposPorHorario:   2,
		MinPessoasPorGrupo: 2,
		MaxPessoasPorGrupo: 2,
	}, 4, 2); total != 12 {
		t.Fatalf("expected 12 possible allocations across schedules, got %d", total)
	}
}

func TestAppWorkflowDefinitionMatchesBackend(t *testing.T) {
	app := NewApp()

	got := app.GetWorkflowDefinition()
	want := logic.WorkflowDefinition()

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("app workflow definition diverged from backend contract:\ngot:  %#v\nwant: %#v", got, want)
	}
}

func TestAppSoftCriterionOptionsMatchBackend(t *testing.T) {
	app := NewApp()

	got := app.GetSoftCriterionOptions()
	want := logic.SoftCriterionOptions()

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("app soft criterion options diverged from backend contract:\ngot:  %#v\nwant: %#v", got, want)
	}
}

func TestAppLifecycleHooksClearTransientStateAndRemoveSmokeFile(t *testing.T) {
	app := NewApp()
	app.ctx = context.Background()
	app.excelData = []byte("payload")
	app.nOpcoes = 3

	if prevent := app.beforeClose(context.Background()); prevent {
		t.Fatal("expected beforeClose to allow shutdown")
	}
	if app.ctx != nil || app.excelData != nil || app.nOpcoes != 0 {
		t.Fatalf("expected transient state to be cleared, got ctx=%v excelData=%v nOpcoes=%d", app.ctx, app.excelData, app.nOpcoes)
	}

	tempDir := t.TempDir()
	smokePath := filepath.Join(tempDir, "wails-smoke.txt")
	if err := os.WriteFile(smokePath, []byte("ok"), 0o600); err != nil {
		t.Fatalf("failed to create smoke sentinel: %v", err)
	}
	t.Setenv("CANDIDATE_ALLOCATOR_WAILS_SMOKE_FILE", smokePath)

	app.excelData = []byte("payload")
	app.nOpcoes = 7
	app.shutdown(context.Background())

	if app.ctx != nil || app.excelData != nil || app.nOpcoes != 0 {
		t.Fatalf("expected shutdown to clear transient state, got ctx=%v excelData=%v nOpcoes=%d", app.ctx, app.excelData, app.nOpcoes)
	}
	if _, err := os.Stat(smokePath); !os.IsNotExist(err) {
		t.Fatalf("expected smoke sentinel to be removed, got err=%v", err)
	}
}

func TestBuildUIPeopleIncludesPreferencesRestrictionsAndExtras(t *testing.T) {
	extraValue := types.NullableString("monitoria")
	avaliadores := []types.Avaliador{{ID: 10, Nome: "Profa. Eva", Sigla: "EV", Email: "eva@insper.edu.br", Extras: map[string]*types.NullableString{"papel": &extraValue}}}
	solverCandidates := []types.SolverCandidate{{
		ID:   1,
		Name: "Ana",
		EvaluatorRestrictions: types.SolverCandidateRestrictions{
			ForbiddenEvaluatorIDs: []int{10},
			AvoidEvaluatorIDs:     []int{10},
		},
	}}

	evaluatorMap := buildUIEvaluatorMap(avaliadores, solverCandidates)
	if got := evaluatorMap[10]; got.Email != "eva@insper.edu.br" || len(got.NaoPosso) != 1 || got.PrefiroNao[0] != "Ana" || got.Extras["papel"] != "monitoria" {
		t.Fatalf("unexpected evaluator UI data: %#v", got)
	}

	candidates := []types.Candidato{{
		Nome:            "Ana",
		Semestre:        "2",
		Curso:           "ADM",
		EmailSecundario: "ana@insper.edu.br",
		Opcoes:          []string{"Segunda 10h", "Terça 14h"},
		Extras:          map[string]*types.NullableString{"papel": &extraValue, "vazio": nil},
	}}
	candidateMap := buildUICandidateMap(candidates, solverCandidates, evaluatorMap)
	got := candidateMap[1]
	if got.Semestre != 2 || len(got.Opcoes) != 2 || got.NaoPosso[0] != "Profa. Eva (EV)" || got.PrefiroNao[0] != "Profa. Eva (EV)" || got.Extras["papel"] != "monitoria" {
		t.Fatalf("unexpected candidate UI data: %#v", got)
	}
	if _, ok := got.Extras["vazio"]; ok {
		t.Fatalf("nil extra should not be exposed: %#v", got.Extras)
	}
}

func TestNullableExtrasToStringsHandlesEmptyMap(t *testing.T) {
	if got := nullableExtrasToStrings(nil); len(got) != 0 {
		t.Fatalf("expected empty extras map, got %#v", got)
	}
}

func TestStopAllocationCancelsActiveSearch(t *testing.T) {
	app := NewApp()
	ctx, cancel := context.WithCancel(context.Background())
	app.allocationRunning = true
	app.allocationCancel = cancel
	if !app.StopAllocation() {
		t.Fatal("expected active allocation to be stopped")
	}
	select {
	case <-ctx.Done():
	default:
		t.Fatal("expected allocation context cancellation")
	}
	if app.StopAllocation() {
		t.Fatal("second stop should report no cancellable search")
	}
}

func TestStartAllocationRejectsConcurrentRun(t *testing.T) {
	app := NewApp()
	app.allocationRunning = true
	if err := app.StartAllocation(types.AllocationConfiguration{}); err == nil {
		t.Fatal("expected concurrent allocation rejection")
	}
}

func TestFinishAllocationRunOnlyClearsMatchingRun(t *testing.T) {
	app := NewApp()
	app.allocationRunning = true
	app.allocationRunID = 2
	app.finishAllocationRun(1)
	if !app.allocationRunning {
		t.Fatal("stale run must not clear the active search")
	}
	app.finishAllocationRun(2)
	if app.allocationRunning || app.allocationCancel != nil {
		t.Fatal("matching run should clear execution state")
	}
}

func TestBuildUIAllocationResultMarksFeasibleSolutionAsProvisional(t *testing.T) {
	run := allocation.ConfiguredAllocationResult{
		Problem: types.AllocationProblem{
			Candidates: []types.SolverCandidate{{ID: 1, Name: "Ana"}},
			Groups:     []types.SolverGroup{{ID: 1, Label: "segunda grupo 1"}},
		},
		Result:     types.SolverResult{Status: "feasible", Assignments: map[int]int{1: 1}},
		Candidatos: []types.Candidato{{Nome: "Ana"}},
	}
	result := buildUIAllocationResult(run)
	if result.Status != "Solução provisória" || result.SolverStatus != "feasible" || len(result.Mesas) != 1 {
		t.Fatalf("unexpected provisional UI result: %#v", result)
	}
}
