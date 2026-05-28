package allocation

import (
	"database/sql"
	"testing"

	dbpkg "candidate_alocator/back/db"
	types "candidate_alocator/back/type"
)

func TestRunConfiguredAllocationUsesConfigurationProblemAndSolver(t *testing.T) {
	db := openTempSQLiteDB(t)
	seedConfiguredAllocationDB(t, db)

	params := types.AllocationParams{
		GruposPorHorario:    1,
		MinPessoasPorGrupo:  1,
		MaxPessoasPorGrupo:  2,
		AvaliadoresPorGrupo: 1,
		SoftCriteria: []types.SoftCriterion{
			{
				Type:           types.SoftCriterionBalancedDistribution,
				ColumnKey:      "curso",
				SelectedValues: []string{"ADM", "ECO"},
			},
		},
	}
	config, err := BuildConfigurationFromPersistedData(db, params)
	if err != nil {
		t.Fatalf("BuildConfigurationFromPersistedData returned error: %v", err)
	}

	got, err := RunConfiguredAllocation(db, config)
	if err != nil {
		t.Fatalf("RunConfiguredAllocation returned error: %v", err)
	}

	if got.Result.Status != "optimal" {
		t.Fatalf("expected optimal result, got %#v", got.Result)
	}
	if len(got.Result.Assignments) != 2 {
		t.Fatalf("expected 2 assignments, got %#v", got.Result.Assignments)
	}
	if len(got.Problem.SoftRules.Criteria) != 1 {
		t.Fatalf("expected soft criteria to reach solver problem, got %#v", got.Problem.SoftRules.Criteria)
	}
	if got.Config.Result.Status != "optimal" {
		t.Fatalf("expected config result to be updated, got %#v", got.Config.Result)
	}
}

func TestRunConfiguredAllocationRejectsInvalidConfiguration(t *testing.T) {
	db := openTempSQLiteDB(t)
	seedConfiguredAllocationDB(t, db)

	config := types.AllocationConfiguration{
		Diagnostics: types.AllocationDiagnostics{HasErrors: true},
	}

	if _, err := RunConfiguredAllocation(db, config); err == nil {
		t.Fatal("expected invalid configuration to fail")
	}
}

func seedConfiguredAllocationDB(t *testing.T, db *sql.DB) {
	t.Helper()

	if err := dbpkg.EnsureAppSchema(db); err != nil {
		t.Fatalf("EnsureAppSchema returned error: %v", err)
	}

	if _, err := db.Exec(`INSERT INTO opcoes_horario (id, opcao) VALUES (1, 'segunda 10h')`); err != nil {
		t.Fatalf("failed to insert horario: %v", err)
	}

	candidates := []struct {
		nome     string
		cpf      string
		semestre int
		curso    string
	}{
		{nome: "Alice", cpf: "00000000001", semestre: 2, curso: "ADM"},
		{nome: "Bruno", cpf: "00000000002", semestre: 4, curso: "ECO"},
	}
	for idx, candidate := range candidates {
		if _, err := db.Exec(`INSERT INTO pessoa (id, timestamp, nome, cpf, numero, semestre, curso, email_secundario, email_pessoal)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			idx+1,
			"2026-01-01",
			candidate.nome,
			candidate.cpf,
			"11999999999",
			candidate.semestre,
			candidate.curso,
			candidate.nome+"@al.insper.edu.br",
			candidate.nome+"@gmail.com",
		); err != nil {
			t.Fatalf("failed to insert candidate %s: %v", candidate.nome, err)
		}
		if _, err := db.Exec(`INSERT INTO disponibilidade (pessoa_id, horario_id, preferencia) VALUES (?, 1, 1)`, idx+1); err != nil {
			t.Fatalf("failed to insert disponibilidade for %s: %v", candidate.nome, err)
		}
	}

	for idx, sigla := range []string{"A1", "A2"} {
		if _, err := db.Exec(`INSERT INTO avaliador (id, nome, email, sigla) VALUES (?, ?, ?, ?)`,
			idx+1,
			"Avaliador "+sigla,
			sigla+"@insper.edu.br",
			sigla,
		); err != nil {
			t.Fatalf("failed to insert evaluator %s: %v", sigla, err)
		}
	}
}
