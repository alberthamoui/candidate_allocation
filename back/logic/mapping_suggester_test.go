package logic

import "testing"

func TestMappingSuggesterHandlesShuffledCandidateHeaders(t *testing.T) {
	headers := []string{
		"Opcao 3",
		"Nome",
		"Email Pessoal",
		"Timestamp",
		"Opcao 1",
		"Semestre",
		"Curso",
		"CPF",
		"Email Secundario",
		"Numero",
		"Opcao 2",
	}
	variables := candidateMappingVariables(3)

	got := suggestMappingByName(headers, variables)

	assertSuggestedColumn(t, got, "timestamp", "Timestamp", 3)
	assertSuggestedColumn(t, got, "nome", "Nome", 1)
	assertSuggestedColumn(t, got, "cpf", "CPF", 7)
	assertSuggestedColumn(t, got, "numero", "Numero", 9)
	assertSuggestedColumn(t, got, "semestre", "Semestre", 5)
	assertSuggestedColumn(t, got, "curso", "Curso", 6)
	assertSuggestedColumn(t, got, "email_secundario", "Email Secundario", 8)
	assertSuggestedColumn(t, got, "email_pessoal", "Email Pessoal", 2)
	assertSuggestedColumn(t, got, "opcao 1", "Opcao 1", 4)
	assertSuggestedColumn(t, got, "opcao 2", "Opcao 2", 10)
	assertSuggestedColumn(t, got, "opcao 3", "Opcao 3", 0)
	assertUniqueMappedIndices(t, got, len(headers))
}

func TestMappingSuggesterNumericNormalization(t *testing.T) {
	testCases := []struct {
		header   string
		variable string
	}{
		{"1opcao", "opcao 1"},
		{"primeira opcao", "opcao 1"},
		{"opcao um", "opcao 1"},
		{"2opcao", "opcao 2"},
		{"segunda opcao", "opcao 2"},
		{"opcao duas", "opcao 2"},
	}

	for _, tc := range testCases {
		score := mappingSimilarityScore(tc.variable, tc.header)
		if score < 10000 { // exactNormalized score
			t.Errorf("expected exact mapping score for %q and %q, got %d", tc.variable, tc.header, score)
		}
	}
}

func TestMappingSuggesterHandlesShuffledCandidateHeaders2(t *testing.T) {
	headers := []string{
		"Opcao 3",
		"Nome",
		"Email Pessoal",
		"Timestamp",
		"Opcao 1",
		"Semestre",
		"Curso",
		"CPF",
		"Email Secundario",
		"Numero",
		"Opcao 2",
	}
	variables := candidateMappingVariables(3)

	got := suggestMappingByName(headers, variables)

	assertSuggestedColumn(t, got, "timestamp", "Timestamp", 3)
	assertSuggestedColumn(t, got, "nome", "Nome", 1)
	assertSuggestedColumn(t, got, "cpf", "CPF", 7)
	assertSuggestedColumn(t, got, "numero", "Numero", 9)
	assertSuggestedColumn(t, got, "semestre", "Semestre", 5)
	assertSuggestedColumn(t, got, "curso", "Curso", 6)
	assertSuggestedColumn(t, got, "email_secundario", "Email Secundario", 8)
	assertSuggestedColumn(t, got, "email_pessoal", "Email Pessoal", 2)
	assertSuggestedColumn(t, got, "opcao 1", "Opcao 1", 4)
	assertSuggestedColumn(t, got, "opcao 2", "Opcao 2", 10)
	assertSuggestedColumn(t, got, "opcao 3", "Opcao 3", 0)
	assertUniqueMappedIndices(t, got, len(headers))
}
