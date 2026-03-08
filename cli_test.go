package main

import (
	"candidate_alocator/back/logic"
	types "candidate_alocator/back/type"
	"testing"
)

func TestBuildCandidatesForCLISelectsFirstValidDuplicate(t *testing.T) {
	resp := UsuariosResponse{
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
	resp := UsuariosResponse{
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
