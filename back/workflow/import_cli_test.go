package workflow

import (
	"candidate_alocator/back/logic"
	types "candidate_alocator/back/type"
	"context"
	"os"
	"path/filepath"
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
