package allocation

import (
	dbpkg "candidate_alocator/back/db"
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestRunReturnsErrorWhenDatabaseIsInvalid(t *testing.T) {
	if err := Run(nil); err == nil {
		t.Fatal("expected Run to fail with nil db")
	}
}

func TestRunReturnsNilWhenAllocationSucceeds(t *testing.T) {
	db := openTempSQLiteDB(t)
	if err := dbpkg.EnsureAppSchema(db); err != nil {
		t.Fatalf("EnsureAppSchema returned error: %v", err)
	}

	if _, err := db.Exec(`INSERT INTO opcoes_horario (opcao) VALUES ('segunda 10h')`); err != nil {
		t.Fatalf("failed to insert horario: %v", err)
	}

	for i := 1; i <= 5; i++ {
		stmt := `INSERT INTO pessoa (timestamp, nome, cpf, numero, semestre, curso, email_secundario, email_pessoal)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
		if _, err := db.Exec(stmt,
			fmt.Sprintf("2026-01-%02d", i),
			fmt.Sprintf("Candidato %d", i),
			fmt.Sprintf("%011d", i),
			fmt.Sprintf("%09d", i),
			2,
			"ADM",
			fmt.Sprintf("cand%d@al.insper.edu.br", i),
			fmt.Sprintf("cand%d@gmail.com", i),
		); err != nil {
			t.Fatalf("failed to insert candidate %d: %v", i, err)
		}

		if _, err := db.Exec(`INSERT INTO disponibilidade (pessoa_id, horario_id, preferencia) VALUES (?, 1, 1)`, i); err != nil {
			t.Fatalf("failed to insert disponibilidade %d: %v", i, err)
		}

		if _, err := db.Exec(`INSERT INTO avaliador (nome, email, sigla) VALUES (?, ?, ?)`,
			fmt.Sprintf("Avaliador %d", i),
			fmt.Sprintf("avaliador%d@insper.edu.br", i),
			fmt.Sprintf("A%d", i),
		); err != nil {
			t.Fatalf("failed to insert avaliador %d: %v", i, err)
		}
	}

	if err := Run(db); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
}

func openTempSQLiteDB(t *testing.T) *sql.DB {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "allocation.db")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("failed to open sqlite db: %v", err)
	}

	t.Cleanup(func() {
		_ = db.Close()
	})

	return db
}
