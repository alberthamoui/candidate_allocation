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

func TestMappingSuggesterHandlesShuffledCandidateHeaders2(t *testing.T) {
	headers := []string{
		"Op 3",
		"Name",
		"Email",
		"Timestamp",
		"Op 1",
		"Sem",
		"Curso",
		"CPF",
		"Email Secundario",
		"Numero",
		"Op 2",
	}
	variables := candidateMappingVariables(3)

	got := suggestMappingByName(headers, variables)

	assertSuggestedColumn(t, got, "timestamp", "Timestamp", 3)
	assertSuggestedColumn(t, got, "nome", "Name", 1)
	assertSuggestedColumn(t, got, "cpf", "CPF", 7)
	assertSuggestedColumn(t, got, "numero", "Numero", 9)
	assertSuggestedColumn(t, got, "semestre", "Sem", 5)
	assertSuggestedColumn(t, got, "curso", "Curso", 6)
	assertSuggestedColumn(t, got, "email_secundario", "Email Secundario", 8)
	assertSuggestedColumn(t, got, "email_pessoal", "Email", 2)
	assertSuggestedColumn(t, got, "opcao 1", "Op 1", 4)
	assertSuggestedColumn(t, got, "opcao 2", "Op 2", 10)
	assertSuggestedColumn(t, got, "opcao 3", "Op 3", 0)
	assertUniqueMappedIndices(t, got, len(headers))
}
