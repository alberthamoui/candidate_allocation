package main

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"

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
