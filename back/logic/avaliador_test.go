package logic

import (
	types "candidate_alocator/back/type"
	"testing"
)

func TestProcessAvaliadoresDeduplication(t *testing.T) {
	data := []types.Avaliador{
		{Nome: "Ana", Email: "ana@insper.edu.br", Sigla: "AN"},
		{Nome: "Ana Silva", Email: "ana@insper.edu.br", Sigla: "ANS"}, // Duplicate by Email
		{Nome: "Bruno", Email: "bruno@insper.edu.br", Sigla: "BR"},
		{Nome: "Carlos", Email: "carlos@insper.edu.br", Sigla: "BR"}, // Duplicate by Sigla
		{Nome: "Ana", Email: "outra@insper.edu.br", Sigla: "OUTRA"},  // Duplicate by Nome
	}

	resultados, duplicados := processAvaliadores(data)

	if len(resultados) != 5 {
		t.Fatalf("expected 5 results, got %d", len(resultados))
	}

	// Ana (1), Ana Silva (2), Ana (5) are all linked.
	// Bruno (3), Carlos (4) are linked by Sigla.

	// Expected groups: {1, 2, 5} and {3, 4}
	if len(duplicados) != 2 {
		t.Fatalf("expected 2 duplicate groups, got %d: %v", len(duplicados), duplicados)
	}

	foundTriple := false
	foundDouble := false
	for _, group := range duplicados {
		if len(group) == 3 {
			foundTriple = true
		} else if len(group) == 2 {
			foundDouble = true
		}
	}

	if !foundTriple || !foundDouble {
		t.Fatalf("unexpected duplicate group sizes: %v", duplicados)
	}
}

func TestProcessAvaliadoresValidation(t *testing.T) {
	data := []types.Avaliador{
		{Nome: "", Email: "ana@insper.edu.br", Sigla: "AN"},
		{Nome: "Bruno", Email: "", Sigla: "BR"},
		{Nome: "Carlos", Email: "carlos@insper.edu.br", Sigla: ""},
	}

	resultados, _ := processAvaliadores(data)

	if !containsErrorMessage(resultados[1].Erros, "nome é obrigatório") {
		t.Errorf("expected error for empty name")
	}
	if !containsErrorMessage(resultados[2].Erros, "email é obrigatório") {
		t.Errorf("expected error for empty email")
	}
	if !containsErrorMessage(resultados[3].Erros, "sigla é obrigatória") {
		t.Errorf("expected error for empty sigla")
	}
}
