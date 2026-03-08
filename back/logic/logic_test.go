package logic

import (
	"bytes"
	dbpkg "candidate_alocator/back/db"
	types "candidate_alocator/back/type"
	"database/sql"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/xuri/excelize/v2"
)

type testSheet struct {
	name string
	rows [][]interface{}
}

func createWorkbook(t *testing.T, sheets ...testSheet) []byte {
	t.Helper()

	if len(sheets) == 0 {
		t.Fatal("createWorkbook requires at least one sheet")
	}

	f := excelize.NewFile()
	if err := f.SetSheetName("Sheet1", sheets[0].name); err != nil {
		t.Fatalf("failed to rename first sheet: %v", err)
	}
	writeRowsToSheet(t, f, sheets[0].name, sheets[0].rows)

	for _, sheet := range sheets[1:] {
		if _, err := f.NewSheet(sheet.name); err != nil {
			t.Fatalf("failed to create sheet %s: %v", sheet.name, err)
		}
		writeRowsToSheet(t, f, sheet.name, sheet.rows)
	}

	buf, err := f.WriteToBuffer()
	if err != nil {
		t.Fatalf("failed to serialize workbook: %v", err)
	}
	return buf.Bytes()
}

func writeRowsToSheet(t *testing.T, f *excelize.File, sheetName string, rows [][]interface{}) {
	t.Helper()

	for i, row := range rows {
		cell, err := excelize.CoordinatesToCellName(1, i+1)
		if err != nil {
			t.Fatalf("failed to calculate cell name: %v", err)
		}
		if err := f.SetSheetRow(sheetName, cell, &row); err != nil {
			t.Fatalf("failed to write row %d to %s: %v", i, sheetName, err)
		}
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

func createTestDB(t *testing.T, tmpDir string) *sql.DB {
	t.Helper()

	dbPath := filepath.Join(tmpDir, "insper.db")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("failed to open sqlite db: %v", err)
	}

	if err := dbpkg.EnsureAppSchema(db); err != nil {
		t.Fatalf("failed to sync schema: %v", err)
	}

	t.Cleanup(func() {
		_ = db.Close()
	})
	return db
}

func fetchStrings(t *testing.T, db *sql.DB, query string) []string {
	t.Helper()

	rows, err := db.Query(query)
	if err != nil {
		t.Fatalf("failed to query strings: %v", err)
	}
	defer rows.Close()

	var values []string
	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			t.Fatalf("failed to scan string row: %v", err)
		}
		values = append(values, value)
	}
	return values
}

func fetchCount(t *testing.T, db *sql.DB, query string) int {
	t.Helper()

	var count int
	if err := db.QueryRow(query).Scan(&count); err != nil {
		t.Fatalf("failed to query count: %v", err)
	}
	return count
}

func containsErrorMessage(errors []ErrorEntry, msg string) bool {
	for _, entry := range errors {
		if entry.Msg == msg {
			return true
		}
	}
	return false
}

func TestSuggestMapping(t *testing.T) {
	workbook := createWorkbook(t, testSheet{
		name: "Candidatos",
		rows: [][]interface{}{
			{"Timestamp", "Nome", "CPF"},
			{"2026-01-01", "Maria", "12345678901"},
		},
	})

	mappings, err := SuggestMapping(workbook, 2)
	if err != nil {
		t.Fatalf("SuggestMapping returned error: %v", err)
	}

	expected := []types.MappingItem{
		{NomeColuna: "Timestamp", Indice: 0, Variavel: "timestamp"},
		{NomeColuna: "Nome", Indice: 1, Variavel: "nome"},
		{NomeColuna: "CPF", Indice: 2, Variavel: "cpf"},
		{NomeColuna: "", Indice: 3, Variavel: "numero"},
		{NomeColuna: "", Indice: 4, Variavel: "semestre"},
		{NomeColuna: "", Indice: 5, Variavel: "curso"},
		{NomeColuna: "", Indice: 6, Variavel: "email_secundario"},
		{NomeColuna: "", Indice: 7, Variavel: "email_pessoal"},
		{NomeColuna: "", Indice: 8, Variavel: "opcao 1"},
		{NomeColuna: "", Indice: 9, Variavel: "opcao 2"},
	}

	if !reflect.DeepEqual(mappings, expected) {
		t.Fatalf("unexpected mappings: %#v", mappings)
	}
}

func TestSuggestMappingAvaliador(t *testing.T) {
	workbook := createWorkbook(t,
		testSheet{name: "Candidatos", rows: [][]interface{}{{"A"}}},
		testSheet{
			name: "Avaliadores",
			rows: [][]interface{}{
				{"Nome Completo", "Email", "Sigla"},
				{"Ana", "ana@insper.edu.br", "AN"},
			},
		},
	)

	mappings, err := SuggestMappingAvaliador(workbook)
	if err != nil {
		t.Fatalf("SuggestMappingAvaliador returned error: %v", err)
	}

	expected := []types.MappingItem{
		{NomeColuna: "Nome Completo", Indice: 0, Variavel: "nome"},
		{NomeColuna: "Email", Indice: 1, Variavel: "email"},
		{NomeColuna: "Sigla", Indice: 2, Variavel: "sigla"},
	}

	if !reflect.DeepEqual(mappings, expected) {
		t.Fatalf("unexpected mappings: %#v", mappings)
	}
}

func TestSuggestMappingRestricao(t *testing.T) {
	workbook := createWorkbook(t,
		testSheet{name: "Candidatos", rows: [][]interface{}{{"A"}}},
		testSheet{name: "Avaliadores", rows: [][]interface{}{{"B"}}},
		testSheet{
			name: "Restricoes",
			rows: [][]interface{}{
				{"Candidato", "Nao Posso", "Prefiro Nao"},
				{"Maria", "AB", "CD"},
			},
		},
	)

	mappings, err := SuggestMappingRestricao(workbook)
	if err != nil {
		t.Fatalf("SuggestMappingRestricao returned error: %v", err)
	}

	expected := []types.MappingItem{
		{NomeColuna: "Candidato", Indice: 0, Variavel: "candidato"},
		{NomeColuna: "Nao Posso", Indice: 1, Variavel: "naoPosso"},
		{NomeColuna: "Prefiro Nao", Indice: 2, Variavel: "prefiroNao"},
	}

	if !reflect.DeepEqual(mappings, expected) {
		t.Fatalf("unexpected mappings: %#v", mappings)
	}
}

func TestSuggestMappingForSheet(t *testing.T) {
	workbook := createWorkbook(t,
		testSheet{name: "Candidatos", rows: [][]interface{}{{"A"}}},
		testSheet{
			name: "Avaliadores",
			rows: [][]interface{}{
				{"Nome", "Email"},
				{"Ana", "ana@insper.edu.br"},
			},
		},
	)

	got, err := suggestMappingForSheet(workbook, 1, []string{"nome", "email", "sigla"})
	if err != nil {
		t.Fatalf("suggestMappingForSheet returned error: %v", err)
	}

	expected := []types.MappingItem{
		{NomeColuna: "Nome", Indice: 0, Variavel: "nome"},
		{NomeColuna: "Email", Indice: 1, Variavel: "email"},
		{NomeColuna: "", Indice: 2, Variavel: "sigla"},
	}
	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("unexpected mapping items: %#v", got)
	}
}

func TestProcessMapping(t *testing.T) {
	t.Run("valid items", func(t *testing.T) {
		items := []string{
			`[["Nome", 0], "nome"]`,
			`[["CPF", 1], "cpf"]`,
		}

		got, err := ProcessMapping(items)
		if err != nil {
			t.Fatalf("ProcessMapping returned error: %v", err)
		}

		expected := []types.MappingItem{
			{NomeColuna: "Nome", Indice: 0, Variavel: "nome"},
			{NomeColuna: "CPF", Indice: 1, Variavel: "cpf"},
		}

		if !reflect.DeepEqual(got, expected) {
			t.Fatalf("unexpected mapping items: %#v", got)
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		_, err := ProcessMapping([]string{`["Nome", 0], "nome"]`})
		if err == nil {
			t.Fatal("expected ProcessMapping to fail for invalid JSON")
		}
	})
}

func TestBuildUsuariosWithMapping(t *testing.T) {
	workbook := createWorkbook(t, testSheet{
		name: "Candidatos",
		rows: [][]interface{}{
			{"Timestamp", "Nome", "CPF", "Numero", "Semestre", "Curso", "Email Secundario", "Email Pessoal", "Opcao 1", "Opcao 2"},
			{"2026-01-01", "Maria", "12345678901", "123456789", "11", "ADM", "maria@al.insper.edu.br", "invalido", "Seg 10h", "Ter 10h"},
			{"2026-01-02", "Ana", "12345678901", "987654321", "2", "ECO", "ana@al.insper.edu.br", "ana@gmail.com", "Qua 10h", "Qui 10h"},
		},
	})

	mappingItems := []types.MappingItem{
		{NomeColuna: "Timestamp", Indice: 0, Variavel: "timestamp"},
		{NomeColuna: "Nome", Indice: 1, Variavel: "nome"},
		{NomeColuna: "CPF", Indice: 2, Variavel: "cpf"},
		{NomeColuna: "Numero", Indice: 3, Variavel: "numero"},
		{NomeColuna: "Semestre", Indice: 4, Variavel: "semestre"},
		{NomeColuna: "Curso", Indice: 5, Variavel: "curso"},
		{NomeColuna: "Email Secundario", Indice: 6, Variavel: "email_secundario"},
		{NomeColuna: "Email Pessoal", Indice: 7, Variavel: "email_pessoal"},
		{NomeColuna: "Opcao 1", Indice: 8, Variavel: "opcao 1"},
		{NomeColuna: "Opcao 2", Indice: 9, Variavel: "opcao 2"},
	}

	resp, err := BuildUsuariosWithMapping(workbook, 2, mappingItems)
	if err != nil {
		t.Fatalf("BuildUsuariosWithMapping returned error: %v", err)
	}

	if len(resp.Usuarios) != 2 {
		t.Fatalf("expected 2 users, got %d", len(resp.Usuarios))
	}

	first := resp.Usuarios[1]
	if first.Usuario.Nome != "Maria" {
		t.Fatalf("unexpected first user name: %s", first.Usuario.Nome)
	}
	if first.Usuario.Semestre != "" {
		t.Fatalf("expected invalid semestre to be cleared, got %q", first.Usuario.Semestre)
	}
	if first.Usuario.EmailPessoal != "" {
		t.Fatalf("expected invalid email_pessoal to be cleared, got %q", first.Usuario.EmailPessoal)
	}
	if !reflect.DeepEqual(first.Usuario.Opcoes, []string{"Seg 10h", "Ter 10h"}) {
		t.Fatalf("unexpected first user options: %#v", first.Usuario.Opcoes)
	}
	if !containsErrorMessage(first.Erros, "semestre inválido") {
		t.Fatalf("expected semestre inválido error, got %#v", first.Erros)
	}
	if !containsErrorMessage(first.Erros, "email_pessoal inválido") {
		t.Fatalf("expected email_pessoal inválido error, got %#v", first.Erros)
	}

	if len(resp.Duplicates) != 1 {
		t.Fatalf("expected one duplicate group, got %#v", resp.Duplicates)
	}
	if !reflect.DeepEqual(resp.DuplicateFields, []string{"cpf", "email_secundario", "email_pessoal"}) {
		t.Fatalf("unexpected duplicate fields: %#v", resp.DuplicateFields)
	}
	group := append([]int(nil), resp.Duplicates[0]...)
	slices.Sort(group)
	if !reflect.DeepEqual(group, []int{1, 2}) {
		t.Fatalf("unexpected duplicate group: %#v", resp.Duplicates)
	}
}

func TestBuildAvaliadoresWithMapping(t *testing.T) {
	withTempWorkingDir(t, func(tmpDir string) {
		db := createTestDB(t, tmpDir)

		workbook := createWorkbook(t,
			testSheet{name: "Candidatos", rows: [][]interface{}{{"A"}}},
			testSheet{
				name: "Avaliadores",
				rows: [][]interface{}{
					{"Nome", "Email", "Sigla"},
					{" Ana ", " ana@insper.edu.br ", " AN "},
					{"", "", ""},
				},
			},
		)

		mappingItems := []types.MappingItem{
			{NomeColuna: "Nome", Indice: 0, Variavel: "nome"},
			{NomeColuna: "Email", Indice: 1, Variavel: "email"},
			{NomeColuna: "Sigla", Indice: 2, Variavel: "sigla"},
		}

		got, err := BuildAvaliadoresWithMapping(workbook, mappingItems)
		if err != nil {
			t.Fatalf("BuildAvaliadoresWithMapping returned error: %v", err)
		}

		expected := []types.AvaliadorInfo{
			{Nome: "Ana", Email: "ana@insper.edu.br", Sigla: "AN"},
		}
		if !reflect.DeepEqual(got, expected) {
			t.Fatalf("unexpected avaliadores: %#v", got)
		}

		if count := fetchCount(t, db, `SELECT COUNT(*) FROM avaliador`); count != 1 {
			t.Fatalf("expected 1 persisted avaliador, got %d", count)
		}
	})
}

func TestBuildRestricoesWithMapping(t *testing.T) {
	workbook := createWorkbook(t,
		testSheet{name: "Candidatos", rows: [][]interface{}{{"A"}}},
		testSheet{name: "Avaliadores", rows: [][]interface{}{{"B"}}},
		testSheet{
			name: "Restricoes",
			rows: [][]interface{}{
				{"Candidato", "Nao Posso", "Prefiro Nao"},
				{"Maria", "AB, CD", "EF"},
			},
		},
	)

	mappingItems := []types.MappingItem{
		{NomeColuna: "Candidato", Indice: 0, Variavel: "candidato"},
		{NomeColuna: "Nao Posso", Indice: 1, Variavel: "naoPosso"},
		{NomeColuna: "Prefiro Nao", Indice: 2, Variavel: "prefiroNao"},
	}

	got, err := BuildRestricoesWithMapping(workbook, mappingItems)
	if err != nil {
		t.Fatalf("BuildRestricoesWithMapping returned error: %v", err)
	}

	expected := []types.Restricao{
		{Candidato: "Maria", NaoPosso: "AB, CD", PrefiroNao: "EF"},
	}
	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("unexpected restricoes: %#v", got)
	}
}

func TestSaveUsuariosFromMaps(t *testing.T) {
	withTempWorkingDir(t, func(tmpDir string) {
		db := createTestDB(t, tmpDir)

		input := []map[string]interface{}{
			{
				"timestamp":        "2026-01-01",
				"nome":             "Maria",
				"cpf":              "12345678901",
				"numero":           "123456789",
				"semestre":         "2",
				"curso":            "ADM",
				"email_secundario": "maria@al.insper.edu.br",
				"email_pessoal":    "maria@gmail.com",
				"opcoes":           []interface{}{" Seg 10h ", "Ter 14h"},
			},
		}

		if err := SaveUsuariosFromMaps(input); err != nil {
			t.Fatalf("SaveUsuariosFromMaps returned error: %v", err)
		}

		if count := fetchCount(t, db, `SELECT COUNT(*) FROM pessoa`); count != 1 {
			t.Fatalf("expected 1 persisted user, got %d", count)
		}
		if count := fetchCount(t, db, `SELECT COUNT(*) FROM disponibilidade`); count != 2 {
			t.Fatalf("expected 2 disponibilidades, got %d", count)
		}

		gotHorarios := fetchStrings(t, db, `SELECT opcao FROM opcoes_horario ORDER BY id`)
		expectedHorarios := []string{"seg 10h", "ter 14h"}
		if !reflect.DeepEqual(gotHorarios, expectedHorarios) {
			t.Fatalf("unexpected horarios persisted: %#v", gotHorarios)
		}
	})
}

func TestSaveRestricoesFromMaps(t *testing.T) {
	withTempWorkingDir(t, func(tmpDir string) {
		db := createTestDB(t, tmpDir)

		seedStatements := []string{
			`INSERT INTO pessoa (nome, cpf, numero, email_secundario, email_pessoal, semestre, curso)
			 VALUES ('Maria', '12345678901', '123456789', 'maria@al.insper.edu.br', 'maria@gmail.com', 2, 'ADM');`,
			`INSERT INTO avaliador (nome, email, sigla) VALUES ('Ana', 'ana@insper.edu.br', 'AN');`,
			`INSERT INTO avaliador (nome, email, sigla) VALUES ('Bruno', 'bruno@insper.edu.br', 'BR');`,
		}
		for _, stmt := range seedStatements {
			if _, err := db.Exec(stmt); err != nil {
				t.Fatalf("failed to seed test data: %v", err)
			}
		}

		input := []map[string]interface{}{
			{
				"candidato":  "Maria",
				"naoPosso":   "AN, BR",
				"prefiroNao": "BR",
			},
		}

		if err := SaveRestricoesFromMaps(input); err != nil {
			t.Fatalf("SaveRestricoesFromMaps returned error: %v", err)
		}

		if count := fetchCount(t, db, `SELECT COUNT(*) FROM restricoesNposso`); count != 2 {
			t.Fatalf("expected 2 restricoesNposso, got %d", count)
		}
		if count := fetchCount(t, db, `SELECT COUNT(*) FROM restricoesPrefiroN`); count != 1 {
			t.Fatalf("expected 1 restricaoPrefiroN, got %d", count)
		}
	})
}

func TestGetRowsFromSheet(t *testing.T) {
	workbook := createWorkbook(t,
		testSheet{name: "Candidatos", rows: [][]interface{}{{"Nome"}, {"Maria"}}},
		testSheet{name: "Avaliadores", rows: [][]interface{}{{"Nome"}, {"Ana"}}},
	)

	rows, err := getRowsFromSheet(workbook, 1)
	if err != nil {
		t.Fatalf("getRowsFromSheet returned error: %v", err)
	}

	expected := [][]string{{"Nome"}, {"Ana"}}
	if !reflect.DeepEqual(rows, expected) {
		t.Fatalf("unexpected rows: %#v", rows)
	}
}

func TestGetAvaliadorFields(t *testing.T) {
	got := getAvaliadorFields()
	expected := []string{"nome", "email", "sigla"}

	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("unexpected avaliador fields: %#v", got)
	}
}

func TestGetRestricaoFields(t *testing.T) {
	got := getRestricaoFields()
	expected := []string{"candidato", "naoPosso", "prefiroNao"}

	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("unexpected restricao fields: %#v", got)
	}
}

func TestProcessData(t *testing.T) {
	users := []types.Candidato{
		{
			Nome:            "Maria",
			CPF:             "12345678901",
			Numero:          "123456789",
			Semestre:        "2",
			EmailSecundario: "MARIA@AL.INSPER.EDU.BR",
			EmailPessoal:    "maria@gmail.com",
		},
		{
			Nome:            "Ana",
			CPF:             "12345678901",
			Numero:          "abc",
			Semestre:        "12",
			EmailSecundario: "ana@al.insper.edu.br",
			EmailPessoal:    "invalido",
		},
	}

	resultados, duplicados := processData(users)
	if len(resultados) != 2 {
		t.Fatalf("expected 2 validation results, got %d", len(resultados))
	}

	first := resultados[1].Usuario
	if first.EmailSecundario != "maria@al.insper.edu.br" {
		t.Fatalf("expected normalized insper email, got %q", first.EmailSecundario)
	}

	second := resultados[2]
	if second.Usuario.Semestre != "" {
		t.Fatalf("expected invalid semestre to be cleared, got %q", second.Usuario.Semestre)
	}
	if second.Usuario.EmailPessoal != "" {
		t.Fatalf("expected invalid personal email to be cleared, got %q", second.Usuario.EmailPessoal)
	}
	if !containsErrorMessage(second.Erros, "numero inválido") {
		t.Fatalf("expected numero inválido error, got %#v", second.Erros)
	}
	if len(duplicados) != 1 {
		t.Fatalf("expected one duplicate group, got %#v", duplicados)
	}
}

func TestGetHorarios(t *testing.T) {
	got := getHorarios([]types.Candidato{
		{Opcoes: []string{"Seg 10h", "Ter 10h"}},
		{Opcoes: []string{"Ter 10h", "Qua 10h"}},
	})

	slices.Sort(got)
	expected := []string{"Qua 10h", "Seg 10h", "Ter 10h"}
	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("unexpected horarios: %#v", got)
	}
}

func TestNormalizaOpcao(t *testing.T) {
	got := normalizaOpcao("  Seg 10H  ")
	if got != "seg 10h" {
		t.Fatalf("unexpected normalized option: %q", got)
	}
}

func TestFillDb(t *testing.T) {
	t.Run("candidatos", func(t *testing.T) {
		withTempWorkingDir(t, func(tmpDir string) {
			db := createTestDB(t, tmpDir)

			fillDb(db, []types.Candidato{
				{
					Timestamp:       "2026-01-01",
					Nome:            "Maria",
					CPF:             "12345678901",
					Numero:          "123456789",
					Semestre:        "2",
					Curso:           "ADM",
					EmailSecundario: "maria@al.insper.edu.br",
					EmailPessoal:    "maria@gmail.com",
					Opcoes:          []string{"Seg 10h", "Ter 14h"},
				},
			})

			if count := fetchCount(t, db, `SELECT COUNT(*) FROM pessoa`); count != 1 {
				t.Fatalf("expected 1 pessoa, got %d", count)
			}
			if count := fetchCount(t, db, `SELECT COUNT(*) FROM disponibilidade`); count != 2 {
				t.Fatalf("expected 2 disponibilidades, got %d", count)
			}
		})
	})

	t.Run("avaliadores", func(t *testing.T) {
		withTempWorkingDir(t, func(tmpDir string) {
			db := createTestDB(t, tmpDir)

			fillDb(db, []types.AvaliadorInfo{
				{Nome: "Ana", Email: "ana@insper.edu.br", Sigla: "AN"},
			})

			if count := fetchCount(t, db, `SELECT COUNT(*) FROM avaliador`); count != 1 {
				t.Fatalf("expected 1 avaliador, got %d", count)
			}
		})
	})

	t.Run("restricoes", func(t *testing.T) {
		withTempWorkingDir(t, func(tmpDir string) {
			db := createTestDB(t, tmpDir)

			seedStatements := []string{
				`INSERT INTO pessoa (timestamp, nome, cpf, numero, email_secundario, email_pessoal, semestre, curso)
				 VALUES ('2026-01-01', 'Maria', '12345678901', '123456789', 'maria@al.insper.edu.br', 'maria@gmail.com', 2, 'ADM');`,
				`INSERT INTO avaliador (nome, email, sigla) VALUES ('Ana', 'ana@insper.edu.br', 'AN');`,
				`INSERT INTO avaliador (nome, email, sigla) VALUES ('Bruno', 'bruno@insper.edu.br', 'BR');`,
			}
			for _, stmt := range seedStatements {
				if _, err := db.Exec(stmt); err != nil {
					t.Fatalf("failed to seed test data: %v", err)
				}
			}

			fillDb(db, []types.Restricao{
				{Candidato: "Maria", NaoPosso: "AN, BR", PrefiroNao: "BR"},
			})

			if count := fetchCount(t, db, `SELECT COUNT(*) FROM restricoesNposso`); count != 2 {
				t.Fatalf("expected 2 restricoesNposso, got %d", count)
			}
			if count := fetchCount(t, db, `SELECT COUNT(*) FROM restricoesPrefiroN`); count != 1 {
				t.Fatalf("expected 1 restricaoPrefiroN, got %d", count)
			}
		})
	})
}

func TestSave(t *testing.T) {
	t.Run("supported type", func(t *testing.T) {
		withTempWorkingDir(t, func(tmpDir string) {
			db := createTestDB(t, tmpDir)

			err := Save([]types.AvaliadorInfo{
				{Nome: "Ana", Email: "ana@insper.edu.br", Sigla: "AN"},
			})
			if err != nil {
				t.Fatalf("Save returned error: %v", err)
			}

			if count := fetchCount(t, db, `SELECT COUNT(*) FROM avaliador`); count != 1 {
				t.Fatalf("expected 1 avaliador, got %d", count)
			}
		})
	})

	t.Run("unsupported type", func(t *testing.T) {
		if err := Save("tipo-nao-suportado"); err != nil {
			t.Fatalf("expected unsupported type to return nil error, got %v", err)
		}
	})
}

func TestBuildCandidateFromRow(t *testing.T) {
	row := []string{"2026-01-01", "Maria", "12345678901", "Seg 10h", "Ter 14h"}
	mapping := []types.MappingItem{
		{Indice: 0, Variavel: "timestamp"},
		{Indice: 1, Variavel: "nome"},
		{Indice: 2, Variavel: "cpf"},
		{Indice: 3, Variavel: "opcao 1"},
		{Indice: 4, Variavel: "opcao 2"},
	}

	got, err := buildCandidateFromRow(row, 2, mapping)
	if err != nil {
		t.Fatalf("buildCandidateFromRow returned error: %v", err)
	}

	if got.Timestamp != "2026-01-01" || got.Nome != "Maria" || got.CPF != "12345678901" {
		t.Fatalf("unexpected candidate built from row: %#v", got)
	}
	if !reflect.DeepEqual(got.Opcoes, []string{"Seg 10h", "Ter 14h"}) {
		t.Fatalf("unexpected candidate options: %#v", got.Opcoes)
	}
}

func TestBuildStructFromRow(t *testing.T) {
	row := []string{"Maria", "AN, BR", "BR"}
	mapping := []types.MappingItem{
		{Indice: 0, Variavel: "candidato"},
		{Indice: 1, Variavel: "naoPosso"},
		{Indice: 2, Variavel: "prefiroNao"},
	}

	got, err := buildStructFromRow[types.Restricao](row, mapping)
	if err != nil {
		t.Fatalf("buildStructFromRow returned error: %v", err)
	}

	expected := types.Restricao{Candidato: "Maria", NaoPosso: "AN, BR", PrefiroNao: "BR"}
	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("unexpected struct built from row: %#v", got)
	}
}

func TestDecodeMapToStruct(t *testing.T) {
	got, err := decodeMapToStruct[types.Candidato](map[string]interface{}{
		"timestamp": "2026-01-01",
		"nome":      "Maria",
		"cpf":       "12345678901",
		"opcoes":    []string{"Seg 10h"},
	})
	if err != nil {
		t.Fatalf("decodeMapToStruct returned error: %v", err)
	}

	if got.Nome != "Maria" || got.CPF != "12345678901" {
		t.Fatalf("unexpected decoded struct: %#v", got)
	}
	if !reflect.DeepEqual(got.Opcoes, []string{"Seg 10h"}) {
		t.Fatalf("unexpected decoded options: %#v", got.Opcoes)
	}
}

func TestIsStructZeroValue(t *testing.T) {
	if !isStructZeroValue(types.AvaliadorInfo{}) {
		t.Fatal("expected empty struct to be zero value")
	}
	if isStructZeroValue(types.AvaliadorInfo{Nome: "Ana"}) {
		t.Fatal("expected non-empty struct to not be zero value")
	}
}

func TestGetRowsFromSheetInvalidWorkbook(t *testing.T) {
	_, err := getRowsFromSheet(bytes.Repeat([]byte("x"), 10), 0)
	if err == nil {
		t.Fatal("expected invalid workbook to return an error")
	}
}
