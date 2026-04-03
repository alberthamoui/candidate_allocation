package logic

import (
	"reflect"
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
