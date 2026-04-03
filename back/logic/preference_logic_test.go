package logic

import (
	"reflect"
	"testing"

	types "candidate_alocator/back/type"
)

func TestDetectUniquePreferenceValues(t *testing.T) {
	candidatos := []types.Candidato{
		{Opcoes: []string{" Seg 10h ", "Ter 14h", ""}},
		{Opcoes: []string{"seg 10h", "Qua 16h"}},
	}

	got := DetectUniquePreferenceValues(candidatos)

	expected := []types.UniqueValueDetection{
		{ValorOriginal: "Seg 10h", ValorNormalizado: "seg 10h", Ocorrencias: 2},
		{ValorOriginal: "Ter 14h", ValorNormalizado: "ter 14h", Ocorrencias: 1},
		{ValorOriginal: "Qua 16h", ValorNormalizado: "qua 16h", Ocorrencias: 1},
	}

	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("unexpected detections: %#v", got)
	}
}

func TestListCandidateCriterionColumnsIncludesCoreAndExtras(t *testing.T) {
	nullable := types.NullableString("F")
	candidatos := []types.Candidato{
		{
			Extras: map[string]*types.NullableString{
				"genero": &nullable,
			},
		},
	}

	got := ListCandidateCriterionColumns(candidatos)

	keys := make([]string, 0, len(got))
	for _, column := range got {
		keys = append(keys, column.Key)
	}

	expectedKeys := []string{
		"timestamp",
		"nome",
		"cpf",
		"numero",
		"semestre",
		"curso",
		"email_secundario",
		"email_pessoal",
		"genero",
	}

	if !reflect.DeepEqual(keys, expectedKeys) {
		t.Fatalf("unexpected criterion columns: %#v", keys)
	}
}

func TestDetectUniqueCandidateColumnValuesSupportsCoreAndExtras(t *testing.T) {
	feminino := types.NullableString(" Feminino ")
	masculino := types.NullableString("masculino")
	candidatos := []types.Candidato{
		{Curso: "ADM", CPF: "123", Extras: map[string]*types.NullableString{"genero": &feminino}},
		{Curso: "adm", CPF: "456", Extras: map[string]*types.NullableString{"genero": &masculino}},
		{Curso: "ECO", CPF: "789"},
	}

	cursoValues, err := DetectUniqueCandidateColumnValues(candidatos, "curso")
	if err != nil {
		t.Fatalf("DetectUniqueCandidateColumnValues(curso) returned error: %v", err)
	}
	expectedCurso := []types.UniqueValueDetection{
		{ValorOriginal: "ADM", ValorNormalizado: "adm", Ocorrencias: 2},
		{ValorOriginal: "ECO", ValorNormalizado: "eco", Ocorrencias: 1},
	}
	if !reflect.DeepEqual(cursoValues, expectedCurso) {
		t.Fatalf("unexpected curso detections: %#v", cursoValues)
	}

	generoValues, err := DetectUniqueCandidateColumnValues(candidatos, "genero")
	if err != nil {
		t.Fatalf("DetectUniqueCandidateColumnValues(genero) returned error: %v", err)
	}
	expectedGenero := []types.UniqueValueDetection{
		{ValorOriginal: "Feminino", ValorNormalizado: "feminino", Ocorrencias: 1},
		{ValorOriginal: "masculino", ValorNormalizado: "masculino", Ocorrencias: 1},
	}
	if !reflect.DeepEqual(generoValues, expectedGenero) {
		t.Fatalf("unexpected genero detections: %#v", generoValues)
	}

	if _, err := DetectUniqueCandidateColumnValues(candidatos, "opcoes"); err == nil {
		t.Fatal("expected opcoes to be rejected as soft criterion column")
	}
	if _, err := DetectUniqueCandidateColumnValues(candidatos, "inexistente"); err == nil {
		t.Fatal("expected unknown column to fail")
	}
}

func TestValidatePreferenceScheduleMappings(t *testing.T) {
	valid := []types.PreferenceScheduleMapping{
		{ValorPreferencia: "seg 10h", Dia: "segunda", Hora: "10:00"},
		{ValorPreferencia: "ter 14h", Dia: "terca", Hora: "14:00"},
	}

	if err := ValidatePreferenceScheduleMappings(valid); err != nil {
		t.Fatalf("expected valid mappings, got error: %v", err)
	}

	tests := []struct {
		name    string
		mapping []types.PreferenceScheduleMapping
	}{
		{
			name: "missing day",
			mapping: []types.PreferenceScheduleMapping{
				{ValorPreferencia: "seg 10h", Hora: "10:00"},
			},
		},
		{
			name: "duplicate preference",
			mapping: []types.PreferenceScheduleMapping{
				{ValorPreferencia: "seg 10h", Dia: "segunda", Hora: "10:00"},
				{ValorPreferencia: " SEG 10H ", Dia: "terca", Hora: "14:00"},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := ValidatePreferenceScheduleMappings(tc.mapping); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}
