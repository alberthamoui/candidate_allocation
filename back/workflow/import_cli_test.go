package workflow

import (
	"bufio"
	"bytes"
	"candidate_alocator/back/logic"
	types "candidate_alocator/back/type"
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestRunCLIRejectsMissingFile(t *testing.T) {
	withTempWorkingDir(t, func(tmpDir string) {
		err := RunCLI(context.Background(), filepath.Join(tmpDir, "missing.xlsx"), 5)
		if err == nil {
			t.Fatal("expected RunCLI to fail for missing file")
		}
	})
}

func TestRunCLIReturnsErrorForUnreadableWorkbook(t *testing.T) {
	withTempWorkingDir(t, func(tmpDir string) {
		filePath := filepath.Join(tmpDir, "invalid.xlsx")
		if err := os.WriteFile(filePath, []byte("not-an-xlsx"), 0o644); err != nil {
			t.Fatalf("failed to create invalid workbook: %v", err)
		}

		err := RunCLI(context.Background(), filePath, 5)
		if err == nil {
			t.Fatal("expected RunCLI to fail for invalid workbook")
		}
	})
}

func TestBuildCandidatesForCLISelectsFirstValidDuplicate(t *testing.T) {
	resp := logic.UsuariosResponse{
		Usuarios: map[int]logic.ValidationResult{
			1: {
				Erros:   []logic.ErrorEntry{{Field: 3, Msg: "cpf invalido"}},
				Usuario: types.Candidato{Nome: "invalido"},
			},
			2: {
				Usuario: types.Candidato{Nome: "valido duplicado"},
			},
			3: {
				Usuario: types.Candidato{Nome: "valido unico"},
			},
		},
		Duplicates: [][]int{{1, 2}},
	}

	candidatos, summary := buildCandidatesForCLI(resp)

	if len(candidatos) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(candidatos))
	}
	if candidatos[0].Nome != "valido duplicado" {
		t.Fatalf("expected duplicate survivor to be the valid candidate, got %q", candidatos[0].Nome)
	}
	if candidatos[1].Nome != "valido unico" {
		t.Fatalf("expected unique valid candidate to be preserved, got %q", candidatos[1].Nome)
	}
	if summary.SkippedInvalid != 1 {
		t.Fatalf("expected 1 invalid candidate skipped, got %d", summary.SkippedInvalid)
	}
	if summary.SkippedDuplicate != 0 {
		t.Fatalf("expected 0 duplicate skips, got %d", summary.SkippedDuplicate)
	}
}

func TestBuildCandidatesForCLISkipsExtraDuplicates(t *testing.T) {
	resp := logic.UsuariosResponse{
		Usuarios: map[int]logic.ValidationResult{
			1: {
				Usuario: types.Candidato{Nome: "primeiro"},
			},
			2: {
				Usuario: types.Candidato{Nome: "segundo"},
			},
			3: {
				Usuario: types.Candidato{Nome: "terceiro"},
			},
			4: {
				Erros:   []logic.ErrorEntry{{Field: 8, Msg: "email invalido"}},
				Usuario: types.Candidato{Nome: "invalido"},
			},
		},
		Duplicates: [][]int{{1, 2, 3}},
	}

	candidatos, summary := buildCandidatesForCLI(resp)

	if len(candidatos) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(candidatos))
	}
	if candidatos[0].Nome != "primeiro" {
		t.Fatalf("expected first duplicate to survive, got %q", candidatos[0].Nome)
	}
	if summary.SkippedDuplicate != 2 {
		t.Fatalf("expected 2 duplicate skips, got %d", summary.SkippedDuplicate)
	}
	if summary.SkippedInvalid != 1 {
		t.Fatalf("expected 1 invalid skip, got %d", summary.SkippedInvalid)
	}
}

func TestCollectAllocationSetupBuildsExpectedSetup(t *testing.T) {
	candidatos := []types.Candidato{
		{Opcoes: []string{"Seg 10h", "Ter 14h"}, Curso: "ADM"},
		{Opcoes: []string{"Seg 10h", "Qua 16h"}, Curso: "ECO"},
	}

	input := strings.NewReader(strings.Join([]string{
		"",
		"",
		"",
		"",
		"",
		"",
		"",
		"",
		"",
		"",
		"n",
	}, "\n") + "\n")
	var output bytes.Buffer

	setup, err := collectAllocationSetup(input, &output, candidatos)
	if err != nil {
		t.Fatalf("collectAllocationSetup returned error: %v", err)
	}

	if len(setup.DetectedPreferences) != 3 {
		t.Fatalf("expected 3 detected preferences, got %d", len(setup.DetectedPreferences))
	}
	if len(setup.PreferenceMappings) != 3 {
		t.Fatalf("expected 3 preference mappings, got %d", len(setup.PreferenceMappings))
	}
	for _, mapping := range setup.PreferenceMappings {
		if mapping.Dia != "segunda" {
			t.Fatalf("expected default dia to be segunda, got %q", mapping.Dia)
		}
		if mapping.Hora != "08:00" {
			t.Fatalf("expected default hora to be 08:00, got %q", mapping.Hora)
		}
	}
	if setup.Params.AvaliadoresPorGrupo != 3 {
		t.Fatalf("expected default avaliadores por grupo = 3, got %d", setup.Params.AvaliadoresPorGrupo)
	}
	if setup.Params.MinPessoasPorGrupo != 4 || setup.Params.MaxPessoasPorGrupo != 8 {
		t.Fatalf("unexpected default min/max params: %#v", setup.Params)
	}
	if len(setup.Params.SoftCriteria) != 0 {
		t.Fatalf("expected no soft criteria by default, got %#v", setup.Params.SoftCriteria)
	}
	if !strings.Contains(output.String(), "PASSO 5") || !strings.Contains(output.String(), "PASSO 6") {
		t.Fatalf("expected setup summary output, got %q", output.String())
	}
}

func TestCollectSoftCriteriaSupportsEachCriterionType(t *testing.T) {
	sexoFeminino := types.NullableString("feminino")
	sexoMasculino := types.NullableString("masculino")
	candidatos := []types.Candidato{
		{Curso: "ADM", Extras: map[string]*types.NullableString{"genero": &sexoFeminino}},
		{Curso: "ECO", Extras: map[string]*types.NullableString{"genero": &sexoMasculino}},
	}

	tests := []struct {
		name        string
		inputLines  []string
		expected    types.SoftCriterion
		containsOut string
	}{
		{
			name: "min value",
			inputLines: []string{
				"s",
				"1",
				"6",
				"1",
				"2",
				"s",
				"n",
			},
			expected:    types.SoftCriterion{Type: types.SoftCriterionMinValue, ColumnKey: "curso", SelectedValues: []string{"adm"}, Threshold: 2},
			containsOut: "Se houver pelo menos uma pessoa de ADM em um grupo, tente manter pelo menos 2 pessoas de ADM nesse grupo.",
		},
		{
			name: "at least one each",
			inputLines: []string{
				"s",
				"2",
				"6",
				"1,2",
				"s",
				"n",
			},
			expected:    types.SoftCriterion{Type: types.SoftCriterionAtLeastOneEach, ColumnKey: "curso", SelectedValues: []string{"adm", "eco"}},
			containsOut: "Garanta representacao de todos os valores selecionados em curso: ADM e ECO.",
		},
		{
			name: "balanced distribution",
			inputLines: []string{
				"s",
				"3",
				"6",
				"1,2",
				"s",
				"n",
			},
			expected:    types.SoftCriterion{Type: types.SoftCriterionBalancedDistribution, ColumnKey: "curso", SelectedValues: []string{"adm", "eco"}},
			containsOut: "Distribua os valores selecionados em curso de forma equilibrada entre os grupos: ADM e ECO.",
		},
		{
			name: "group together",
			inputLines: []string{
				"s",
				"4",
				"9",
				"1,2",
				"s",
				"n",
			},
			expected:    types.SoftCriterion{Type: types.SoftCriterionGroupTogether, ColumnKey: "genero", SelectedValues: []string{"feminino", "masculino"}},
			containsOut: "Se possivel, mantenha no mesmo grupo as pessoas com os valores selecionados em genero: Feminino e Masculino.",
		},
		{
			name: "max value",
			inputLines: []string{
				"s",
				"5",
				"6",
				"2",
				"1",
				"s",
				"n",
			},
			expected:    types.SoftCriterion{Type: types.SoftCriterionMaxValue, ColumnKey: "curso", SelectedValues: []string{"eco"}, Threshold: 1},
			containsOut: "Se houver pessoas de ECO em um grupo, tente manter no maximo 1 pessoa de ECO nesse grupo.",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var output bytes.Buffer
			input := strings.NewReader(strings.Join(tc.inputLines, "\n") + "\n")

			got, err := collectSoftCriteria(bufio.NewReader(input), &output, candidatos)
			if err != nil {
				t.Fatalf("collectSoftCriteria returned error: %v", err)
			}
			if !reflect.DeepEqual(got, []types.SoftCriterion{tc.expected}) {
				t.Fatalf("unexpected criteria: %#v", got)
			}
			if !strings.Contains(output.String(), tc.containsOut) {
				t.Fatalf("expected output to contain %q, got %q", tc.containsOut, output.String())
			}
		})
	}
}

func TestCollectSoftCriteriaAccumulatesMultipleCriteria(t *testing.T) {
	candidatos := []types.Candidato{
		{Curso: "ADM"},
		{Curso: "ECO"},
	}
	var output bytes.Buffer
	input := strings.NewReader(strings.Join([]string{
		"s",
		"2",
		"6",
		"1,2",
		"s",
		"s",
		"5",
		"6",
		"2",
		"1",
		"s",
		"n",
	}, "\n") + "\n")

	got, err := collectSoftCriteria(bufio.NewReader(input), &output, candidatos)
	if err != nil {
		t.Fatalf("collectSoftCriteria returned error: %v", err)
	}

	expected := []types.SoftCriterion{
		{Type: types.SoftCriterionAtLeastOneEach, ColumnKey: "curso", SelectedValues: []string{"adm", "eco"}},
		{Type: types.SoftCriterionMaxValue, ColumnKey: "curso", SelectedValues: []string{"eco"}, Threshold: 1},
	}
	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("unexpected accumulated criteria: %#v", got)
	}
}

func TestPromptRequiredLineRetriesOnBlankInput(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("\nsegunda\n"))
	var out bytes.Buffer

	got, err := promptRequiredLine(reader, &out, "Dia real: ")
	if err != nil {
		t.Fatalf("promptRequiredLine returned error: %v", err)
	}
	if got != "segunda" {
		t.Fatalf("unexpected required line value: %q", got)
	}
}

func TestPromptLineWithDefaultUsesDefaultOnBlankInput(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("\n"))
	var out bytes.Buffer

	got, err := promptLineWithDefault(reader, &out, "Dia real", "segunda")
	if err != nil {
		t.Fatalf("promptLineWithDefault returned error: %v", err)
	}
	if got != "segunda" {
		t.Fatalf("unexpected default line value: %q", got)
	}
	if !strings.Contains(out.String(), "Dia real [segunda]: ") {
		t.Fatalf("expected prompt to expose default value, got %q", out.String())
	}
}

func TestReadOptionalLineReturnsTrimmedValue(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("  42  \n"))

	got, err := readOptionalLine(reader)
	if err != nil {
		t.Fatalf("readOptionalLine returned error: %v", err)
	}
	if got != "42" {
		t.Fatalf("unexpected optional line value: %q", got)
	}
}

func TestPromptMultiChoiceIndicesDeduplicatesValues(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("1,2,2\n"))
	var out bytes.Buffer

	got, err := promptMultiChoiceIndices(reader, &out, "Valores: ", 3)
	if err != nil {
		t.Fatalf("promptMultiChoiceIndices returned error: %v", err)
	}

	expected := []int{0, 1}
	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("unexpected multi-choice indices: %#v", got)
	}
}

func withTempWorkingDir(t *testing.T, fn func(tmpDir string)) {
	t.Helper()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working dir: %v", err)
	}

	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change working dir: %v", err)
	}
	defer func() {
		if err := os.Chdir(wd); err != nil {
			t.Fatalf("failed to restore working dir: %v", err)
		}
	}()

	fn(tmpDir)
}
