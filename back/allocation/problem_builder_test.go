package allocation

import (
	"reflect"
	"testing"

	types "candidate_alocator/back/type"
)

func TestBuildAllocationProblem(t *testing.T) {
	config, candidatos, avaliadores, restricoes := makeSolverFixture()

	got, err := BuildAllocationProblem(config, candidatos, avaliadores, restricoes)
	if err != nil {
		t.Fatalf("BuildAllocationProblem returned error: %v", err)
	}

	if len(got.Groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(got.Groups))
	}
	if got.Groups[0].ID != 1 || got.Groups[1].ID != 2 {
		t.Fatalf("unexpected group ids: %#v", got.Groups)
	}
	if !reflect.DeepEqual(got.Groups[0].EvaluatorIDs, []int{10}) {
		t.Fatalf("unexpected evaluator assignment for group 1: %#v", got.Groups[0].EvaluatorIDs)
	}
	if !reflect.DeepEqual(got.Groups[1].EvaluatorIDs, []int{20}) {
		t.Fatalf("unexpected evaluator assignment for group 2: %#v", got.Groups[1].EvaluatorIDs)
	}

	if !reflect.DeepEqual(got.Candidates[0].PreferredGroupIDs, []int{1, 2}) {
		t.Fatalf("unexpected preferred groups for candidate 1: %#v", got.Candidates[0].PreferredGroupIDs)
	}
	if !reflect.DeepEqual(got.Candidates[1].PreferredGroupIDs, []int{2, 1}) {
		t.Fatalf("unexpected preferred groups for candidate 2: %#v", got.Candidates[1].PreferredGroupIDs)
	}
	if !reflect.DeepEqual(got.Candidates[2].Attributes, map[string]string{"curso": "adm"}) {
		t.Fatalf("unexpected attributes for candidate 3: %#v", got.Candidates[2].Attributes)
	}
	if !reflect.DeepEqual(got.Candidates[0].EvaluatorRestrictions.ForbiddenEvaluatorIDs, []int{20}) {
		t.Fatalf("unexpected forbidden evaluators: %#v", got.Candidates[0].EvaluatorRestrictions)
	}
	if !reflect.DeepEqual(got.Candidates[2].EvaluatorRestrictions.AvoidEvaluatorIDs, []int{20}) {
		t.Fatalf("unexpected avoid evaluators: %#v", got.Candidates[2].EvaluatorRestrictions)
	}
	if !reflect.DeepEqual(got.SoftRules.PreferencePenaltyByRank, []int{0, 1, 2, 3, 4}) {
		t.Fatalf("unexpected preference penalties: %#v", got.SoftRules.PreferencePenaltyByRank)
	}
	if got.SoftRules.AvoidEvaluatorPenalty != 3 {
		t.Fatalf("unexpected avoid evaluator penalty: %d", got.SoftRules.AvoidEvaluatorPenalty)
	}
}

func TestSelectEvaluatorIDs(t *testing.T) {
	avaliadores := []types.Avaliador{
		{ID: 10, Sigla: "A1"},
		{ID: 20, Sigla: "A2"},
		{ID: 30, Sigla: "A3"},
	}

	got := selectEvaluatorIDs(avaliadores, 2, 2)
	want := []int{30, 10}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected selected evaluators: got %#v want %#v", got, want)
	}
}

func TestSelectEvaluatorIDsCapsCountToAvailableEvaluators(t *testing.T) {
	avaliadores := []types.Avaliador{
		{ID: 10, Sigla: "A1"},
		{ID: 20, Sigla: "A2"},
	}

	got := selectEvaluatorIDs(avaliadores, 0, 3)
	want := []int{10, 20}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected selected evaluators: got %#v want %#v", got, want)
	}
}

func TestParseRestrictionEvaluatorIDs(t *testing.T) {
	got, err := parseRestrictionEvaluatorIDs("A1, A2 A1", map[string]int{
		"A1": 10,
		"A2": 20,
	})
	if err != nil {
		t.Fatalf("parseRestrictionEvaluatorIDs returned error: %v", err)
	}

	want := []int{10, 20}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected parsed evaluator ids: got %#v want %#v", got, want)
	}
}

func TestBuildCandidateAttributes(t *testing.T) {
	extra := types.NullableString(" Feminino ")
	candidato := types.Candidato{
		Curso:  " ADM ",
		Extras: map[string]*types.NullableString{"genero": &extra},
	}

	got := buildCandidateAttributes(candidato, []string{"curso", "genero"})
	want := map[string]string{"curso": "adm", "genero": "feminino"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected candidate attributes: got %#v want %#v", got, want)
	}
}

func makeSolverFixture() (types.AllocationConfiguration, []types.Candidato, []types.Avaliador, []types.Restricao) {
	config := types.AllocationConfiguration{
		Normalized: types.NormalizedAllocationInput{
			PreferenceMappings: []types.PreferenceScheduleMapping{
				{ValorPreferencia: "seg", Dia: "segunda", Hora: "10:00"},
				{ValorPreferencia: "ter", Dia: "terca", Hora: "10:00"},
			},
			Params: types.AllocationParams{
				GruposPorHorario:    1,
				MinPessoasPorGrupo:  1,
				MaxPessoasPorGrupo:  2,
				AvaliadoresPorGrupo: 1,
				SoftCriteria: []types.SoftCriterion{
					{Type: types.SoftCriterionBalancedDistribution, ColumnKey: "curso", SelectedValues: []string{"adm"}},
				},
			},
		},
	}

	candidatos := []types.Candidato{
		{Nome: "Alice", Curso: "ADM", Opcoes: []string{"seg", "ter"}},
		{Nome: "Bruno", Curso: "ECO", Opcoes: []string{"ter", "seg"}},
		{Nome: "Carla", Curso: "ADM", Opcoes: []string{"seg", "ter"}},
	}

	avaliadores := []types.Avaliador{
		{ID: 10, Nome: "Avaliador 1", Sigla: "A1"},
		{ID: 20, Nome: "Avaliador 2", Sigla: "A2"},
	}

	restricoes := []types.Restricao{
		{Candidato: "Alice", NaoPosso: "A2"},
		{Candidato: "Carla", PrefiroNao: "A2"},
	}

	return config, candidatos, avaliadores, restricoes
}
