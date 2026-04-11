package logic

import (
	"reflect"
	"strings"
	"testing"

	types "candidate_alocator/back/type"
)

func TestDefaultAllocationParams(t *testing.T) {
	got := DefaultAllocationParams()

	expected := types.AllocationParams{
		GruposPorHorario:    2,
		MinPessoasPorGrupo:  4,
		MaxPessoasPorGrupo:  8,
		AvaliadoresPorGrupo: 3,
		SoftCriteria:        []types.SoftCriterion{},
	}

	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("unexpected default params: %#v", got)
	}
}

func TestNormalizePreferenceScheduleMappings(t *testing.T) {
	got := NormalizePreferenceScheduleMappings([]types.PreferenceScheduleMapping{
		{ValorPreferencia: " Seg 10H ", Dia: "  segunda  feira ", Hora: " 08:00 "},
	})

	expected := []types.PreferenceScheduleMapping{
		{ValorPreferencia: "seg 10h", Dia: "segunda feira", Hora: "08:00"},
	}

	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("unexpected normalized mappings: %#v", got)
	}
}

func TestNormalizeSoftCriteria(t *testing.T) {
	got := NormalizeSoftCriteria([]types.SoftCriterion{
		{
			Type:           " MIN_VALUE ",
			ColumnKey:      "curso",
			SelectedValues: []string{" ADM ", "adm", "", "ECO"},
			Threshold:      2,
		},
	})

	expected := []types.SoftCriterion{
		{
			Type:           types.SoftCriterionMinValue,
			ColumnKey:      "curso",
			SelectedValues: []string{"adm", "eco"},
			Threshold:      2,
		},
	}
	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("unexpected normalized soft criteria: %#v", got)
	}
}

func TestNormalizeAllocationParams(t *testing.T) {
	got := NormalizeAllocationParams(types.AllocationParams{
		GruposPorHorario:    2,
		MinPessoasPorGrupo:  4,
		MaxPessoasPorGrupo:  8,
		AvaliadoresPorGrupo: 3,
		SoftCriteria: []types.SoftCriterion{
			{Type: " MIN_VALUE ", ColumnKey: "curso", SelectedValues: []string{" ADM "}, Threshold: 1},
		},
	})

	if got.SoftCriteria[0].Type != types.SoftCriterionMinValue {
		t.Fatalf("expected criterion type to be normalized, got %#v", got.SoftCriteria)
	}
}

func TestValidateSoftCriteria(t *testing.T) {
	feminino := types.NullableString("F")
	candidatos := []types.Candidato{
		{Curso: "ADM", Extras: map[string]*types.NullableString{"genero": &feminino}},
		{Curso: "ECO"},
	}

	valid := []types.SoftCriterion{
		{Type: types.SoftCriterionMinValue, ColumnKey: "curso", SelectedValues: []string{"adm"}, Threshold: 2},
		{Type: types.SoftCriterionAtLeastOneEach, ColumnKey: "curso", SelectedValues: []string{"adm", "eco"}},
		{Type: types.SoftCriterionBalancedDistribution, ColumnKey: "curso", SelectedValues: []string{"adm", "eco"}},
		{Type: types.SoftCriterionGroupTogether, ColumnKey: "genero", SelectedValues: []string{"f"}},
		{Type: types.SoftCriterionMaxValue, ColumnKey: "curso", SelectedValues: []string{"eco"}, Threshold: 1},
	}

	if err := ValidateSoftCriteria(valid, candidatos); err != nil {
		t.Fatalf("expected valid soft criteria, got error: %v", err)
	}

	tests := []struct {
		name     string
		criteria []types.SoftCriterion
	}{
		{
			name: "min with multiple values",
			criteria: []types.SoftCriterion{
				{Type: types.SoftCriterionMinValue, ColumnKey: "curso", SelectedValues: []string{"adm", "eco"}, Threshold: 1},
			},
		},
		{
			name: "max without threshold",
			criteria: []types.SoftCriterion{
				{Type: types.SoftCriterionMaxValue, ColumnKey: "curso", SelectedValues: []string{"adm"}},
			},
		},
		{
			name: "group together without values",
			criteria: []types.SoftCriterion{
				{Type: types.SoftCriterionGroupTogether, ColumnKey: "curso"},
			},
		},
		{
			name: "unknown column",
			criteria: []types.SoftCriterion{
				{Type: types.SoftCriterionBalancedDistribution, ColumnKey: "inexistente", SelectedValues: []string{"adm"}},
			},
		},
		{
			name: "unknown value",
			criteria: []types.SoftCriterion{
				{Type: types.SoftCriterionAtLeastOneEach, ColumnKey: "curso", SelectedValues: []string{"direito"}},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := ValidateSoftCriteria(tc.criteria, candidatos); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestValidateAllocationParams(t *testing.T) {
	candidatos := []types.Candidato{
		{Curso: "ADM"},
		{Curso: "ECO"},
	}

	valid := types.AllocationParams{
		GruposPorHorario:    2,
		MinPessoasPorGrupo:  4,
		MaxPessoasPorGrupo:  8,
		AvaliadoresPorGrupo: 3,
		SoftCriteria: []types.SoftCriterion{
			{Type: types.SoftCriterionBalancedDistribution, ColumnKey: "curso", SelectedValues: []string{"adm", "eco"}},
		},
	}

	if err := ValidateAllocationParams(valid, candidatos); err != nil {
		t.Fatalf("expected valid params, got error: %v", err)
	}

	tests := []struct {
		name  string
		input types.AllocationParams
	}{
		{
			name: "groups zero",
			input: types.AllocationParams{
				GruposPorHorario:    0,
				MinPessoasPorGrupo:  4,
				MaxPessoasPorGrupo:  8,
				AvaliadoresPorGrupo: 3,
			},
		},
		{
			name: "min greater than max",
			input: types.AllocationParams{
				GruposPorHorario:    2,
				MinPessoasPorGrupo:  9,
				MaxPessoasPorGrupo:  8,
				AvaliadoresPorGrupo: 3,
			},
		},
		{
			name: "evaluators zero",
			input: types.AllocationParams{
				GruposPorHorario:    2,
				MinPessoasPorGrupo:  4,
				MaxPessoasPorGrupo:  8,
				AvaliadoresPorGrupo: 0,
			},
		},
		{
			name: "invalid soft criteria",
			input: types.AllocationParams{
				GruposPorHorario:    2,
				MinPessoasPorGrupo:  4,
				MaxPessoasPorGrupo:  8,
				AvaliadoresPorGrupo: 3,
				SoftCriteria: []types.SoftCriterion{
					{Type: types.SoftCriterionMinValue, ColumnKey: "curso", SelectedValues: []string{"adm", "eco"}, Threshold: 1},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := ValidateAllocationParams(tc.input, candidatos); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestDescribeSoftCriterion(t *testing.T) {
	got := DescribeSoftCriterion(types.SoftCriterion{
		Type:           types.SoftCriterionBalancedDistribution,
		ColumnKey:      "curso",
		SelectedValues: []string{"adm", "eco"},
	})

	want := "Distribua os valores selecionados em curso de forma equilibrada entre os grupos: ADM e ECO."
	if got != want {
		t.Fatalf("unexpected soft criterion summary: %q", got)
	}
}

func TestBuildAllocationConfiguration(t *testing.T) {
	candidatos := []types.Candidato{
		{Opcoes: []string{"Seg 10h", "Ter 14h"}, Curso: "ADM"},
		{Opcoes: []string{"Seg 10h", "Qua 16h"}, Curso: "ECO"},
	}

	detections := DetectUniquePreferenceValues(candidatos)
	mappings := []types.PreferenceScheduleMapping{
		{ValorPreferencia: " Seg 10H ", Dia: " segunda ", Hora: " 10:00 "},
		{ValorPreferencia: "Ter 14h", Dia: "terca", Hora: "14:00"},
		{ValorPreferencia: "Qua 16h", Dia: "quarta", Hora: "16:00"},
	}
	params := types.AllocationParams{
		GruposPorHorario:    3,
		MinPessoasPorGrupo:  4,
		MaxPessoasPorGrupo:  8,
		AvaliadoresPorGrupo: 2,
		SoftCriteria: []types.SoftCriterion{
			{Type: " BALANCED_DISTRIBUTION ", ColumnKey: "curso", SelectedValues: []string{" ADM ", "ECO"}},
		},
	}

	got, err := BuildAllocationConfiguration(detections, mappings, params, candidatos)
	if err != nil {
		t.Fatalf("BuildAllocationConfiguration returned error: %v", err)
	}

	if got.Normalized.PreferenceMappings[0].ValorPreferencia != "seg 10h" {
		t.Fatalf("expected normalized preference mapping, got %#v", got.Normalized.PreferenceMappings)
	}
	if got.Normalized.Params.SoftCriteria[0].Type != types.SoftCriterionBalancedDistribution {
		t.Fatalf("expected normalized soft criteria in params, got %#v", got.Normalized.Params.SoftCriteria)
	}
	if got.Diagnostics.HasErrors {
		t.Fatal("expected diagnostics without errors")
	}
	if len(got.Diagnostics.PreferenceMappings) != len(mappings) {
		t.Fatalf("expected %d mapping diagnostics, got %d", len(mappings), len(got.Diagnostics.PreferenceMappings))
	}
	if got.Result.Status != "not_run" {
		t.Fatalf("expected result status not_run, got %q", got.Result.Status)
	}
	if len(got.Summary.DetectedPreferences) != 3 {
		t.Fatalf("expected 3 detected preferences in summary, got %#v", got.Summary.DetectedPreferences)
	}
	if !containsLineWithPrefix(got.Summary.ValidationObservations, "WARNING: configuracao validada com sucesso") {
		t.Fatalf("expected validation observation in summary, got %#v", got.Summary.ValidationObservations)
	}
	if !containsLineWithPrefix(got.Summary.NormalizedValues, "preferencia \"Seg 10h\" -> \"seg 10h\"") {
		t.Fatalf("expected normalized values summary, got %#v", got.Summary.NormalizedValues)
	}
}

func TestBuildAllocationConfigurationAggregatesValidationError(t *testing.T) {
	candidatos := []types.Candidato{
		{Curso: "ADM"},
	}

	_, err := BuildAllocationConfiguration(
		nil,
		[]types.PreferenceScheduleMapping{{ValorPreferencia: "ADM", Dia: "", Hora: "08:00"}},
		DefaultAllocationParams(),
		candidatos,
	)
	if err == nil {
		t.Fatal("expected validation error")
	}

	got, err := BuildAllocationConfiguration(
		nil,
		[]types.PreferenceScheduleMapping{{ValorPreferencia: "ADM", Dia: "", Hora: "08:00"}},
		DefaultAllocationParams(),
		candidatos,
	)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !got.Diagnostics.HasErrors {
		t.Fatal("expected diagnostics to flag errors")
	}
	if len(got.Diagnostics.ValidationMessages) == 0 {
		t.Fatal("expected validation messages")
	}
	if got.Diagnostics.ValidationMessages[0].Level != types.ValidationMessageLevelError {
		t.Fatalf("expected error validation message, got %#v", got.Diagnostics.ValidationMessages)
	}
}

func containsLineWithPrefix(lines []string, prefix string) bool {
	for _, line := range lines {
		if strings.HasPrefix(line, prefix) {
			return true
		}
	}
	return false
}
