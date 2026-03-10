package logic

import (
	"bytes"
	dbpkg "candidate_alocator/back/db"
	types "candidate_alocator/back/type"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
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

func TestNormalizeMappingName(t *testing.T) {
	got := normalizeMappingName("EmailPessoal-1")
	if got != "email pessoal 1" {
		t.Fatalf("unexpected normalized name: %q", got)
	}
}

func TestMappingSimilarityScore(t *testing.T) {
	exact := mappingSimilarityScore("email_pessoal", "Email Pessoal")
	partial := mappingSimilarityScore("email_pessoal", "Email")
	mismatch := mappingSimilarityScore("opcao 1", "Opcao 2")

	if exact <= partial {
		t.Fatalf("expected exact score to beat partial match: exact=%d partial=%d", exact, partial)
	}
	if partial <= mismatch {
		t.Fatalf("expected partial score to beat mismatched option number: partial=%d mismatch=%d", partial, mismatch)
	}
}

func TestResolveMappingConflictsUsesBestAvailableUniqueColumn(t *testing.T) {
	headers := []string{"Email Secundario", "Email Pessoal"}
	variables := []string{"email_pessoal", "email_secundario"}

	got, _ := resolveMappingConflicts(headers, variables, buildMappingCandidates(headers, variables))

	assertSuggestedColumn(t, got, "email_pessoal", "Email Pessoal", 1)
	assertSuggestedColumn(t, got, "email_secundario", "Email Secundario", 0)
	assertUniqueMappedIndices(t, got, len(headers))
}

func TestResolveMappingConflictsFallsBackToNextBestColumn(t *testing.T) {
	headers := []string{"Email Pessoal", "Email Secundario"}
	variables := []string{"email_pessoal", "email_secundario"}

	got, _ := resolveMappingConflicts(headers, variables, buildMappingCandidates(headers, variables))

	assertSuggestedColumn(t, got, "email_pessoal", "Email Pessoal", 0)
	assertSuggestedColumn(t, got, "email_secundario", "Email Secundario", 1)
	assertUniqueMappedIndices(t, got, len(headers))
}

func TestSuggestMappingByNameLeavesRemainingVariablesUnmappedWhenHeadersAreFewer(t *testing.T) {
	headers := []string{"Nome", "CPF"}
	variables := []string{"nome", "cpf", "email"}

	got := suggestMappingByName(headers, variables)

	assertSuggestedColumn(t, got, "nome", "Nome", 0)
	assertSuggestedColumn(t, got, "cpf", "CPF", 1)
	assertUnmappedVariable(t, got, "email", len(headers))
	assertUniqueMappedIndices(t, got, len(headers))
}

func TestSuggestMappingByNameMapsAllVariablesWhenHeadersAreEnough(t *testing.T) {
	headers := []string{"Email", "Nome", "CPF"}
	variables := []string{"nome", "cpf", "email"}

	got := suggestMappingByName(headers, variables)

	assertSuggestedColumn(t, got, "nome", "Nome", 1)
	assertSuggestedColumn(t, got, "cpf", "CPF", 2)
	assertSuggestedColumn(t, got, "email", "Email", 0)
	assertUniqueMappedIndices(t, got, len(headers))
}

func TestSuggestMappingTimestampCamelCase(t *testing.T) {
	headers := []string{"TimeStamp", "Nome"}
	variables := []string{"timestamp", "nome"}

	got := suggestMappingByName(headers, variables)

	assertSuggestedColumn(t, got, "timestamp", "TimeStamp", 0)
	assertSuggestedColumn(t, got, "nome", "Nome", 1)
	assertUniqueMappedIndices(t, got, len(headers))
}

func TestSuggestMappingUsesCurrentCandidateSchema(t *testing.T) {
	workbook := createWorkbook(t, testSheet{
		name: "Candidatos",
		rows: [][]interface{}{
			{"Email Pessoal", "CPF", "Opcao 2", "Nome", "Opcao 1", "Email Secundario", "Numero", "Curso", "Semestre", "Timestamp"},
			{"2026-01-01", "Maria", "12345678901"},
		},
	})

	mappings, err := SuggestMapping(workbook, 2)
	if err != nil {
		t.Fatalf("SuggestMapping returned error: %v", err)
	}

	assertSuggestedColumn(t, mappings, "email_pessoal", "Email Pessoal", 0)
	assertSuggestedColumn(t, mappings, "cpf", "CPF", 1)
	assertSuggestedColumn(t, mappings, "opcao 2", "Opcao 2", 2)
	assertSuggestedColumn(t, mappings, "nome", "Nome", 3)
	assertSuggestedColumn(t, mappings, "opcao 1", "Opcao 1", 4)
	assertSuggestedColumn(t, mappings, "email_secundario", "Email Secundario", 5)
	assertSuggestedColumn(t, mappings, "numero", "Numero", 6)
	assertSuggestedColumn(t, mappings, "curso", "Curso", 7)
	assertSuggestedColumn(t, mappings, "semestre", "Semestre", 8)
	assertSuggestedColumn(t, mappings, "timestamp", "Timestamp", 9)
	assertUniqueMappedIndices(t, mappings, 10)
}

func TestSuggestMappingAvaliadorUsesCurrentSchema(t *testing.T) {
	workbook := createWorkbook(t,
		testSheet{name: "Candidatos", rows: [][]interface{}{{"A"}}},
		testSheet{
			name: "Avaliadores",
			rows: [][]interface{}{
				{"Sigla", "Nome", "Email"},
				{"Ana", "ana@insper.edu.br", "AN"},
			},
		},
	)

	mappings, err := SuggestMappingAvaliador(workbook)
	if err != nil {
		t.Fatalf("SuggestMappingAvaliador returned error: %v", err)
	}

	assertSuggestedColumn(t, mappings, "sigla", "Sigla", 0)
	assertSuggestedColumn(t, mappings, "nome", "Nome", 1)
	assertSuggestedColumn(t, mappings, "email", "Email", 2)
	assertUniqueMappedIndices(t, mappings, 3)
}

func TestSuggestMappingRestricao(t *testing.T) {
	workbook := createWorkbook(t,
		testSheet{name: "Candidatos", rows: [][]interface{}{{"A"}}},
		testSheet{name: "Avaliadores", rows: [][]interface{}{{"B"}}},
		testSheet{
			name: "Restricoes",
			rows: [][]interface{}{
				{"Prefiro Nao", "Candidato", "Nao Posso"},
				{"Maria", "AB", "CD"},
			},
		},
	)

	mappings, err := SuggestMappingRestricao(workbook)
	if err != nil {
		t.Fatalf("SuggestMappingRestricao returned error: %v", err)
	}

	assertSuggestedColumn(t, mappings, "prefiroNao", "Prefiro Nao", 0)
	assertSuggestedColumn(t, mappings, "candidato", "Candidato", 1)
	assertSuggestedColumn(t, mappings, "naoPosso", "Nao Posso", 2)
	assertUniqueMappedIndices(t, mappings, 3)
}

func TestSuggestMappingForSheet(t *testing.T) {
	workbook := createWorkbook(t,
		testSheet{name: "Candidatos", rows: [][]interface{}{{"A"}}},
		testSheet{
			name: "Avaliadores",
			rows: [][]interface{}{
				{"Email", "Nome"},
				{"Ana", "ana@insper.edu.br"},
			},
		},
	)

	got, err := suggestMappingForSheet(workbook, 1, []string{"nome", "email", "sigla"})
	if err != nil {
		t.Fatalf("suggestMappingForSheet returned error: %v", err)
	}

	assertSuggestedColumn(t, got, "email", "Email", 0)
	assertSuggestedColumn(t, got, "nome", "Nome", 1)
	assertUnmappedVariable(t, got, "sigla", 2)
	assertUniqueMappedIndices(t, got, 2)
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
			{"Timestamp", "Nome", "CPF", "Numero", "Semestre", "Curso", "Email Secundario", "Email Pessoal", "Opcao 1", "Opcao 2", "LinkedIn"},
			{"2026-01-01", "Maria", "12345678901", "123456789", "11", "ADM", "maria@al.insper.edu.br", "invalido", "Seg 10h", "Ter 10h", "linkedin.com/in/maria"},
			{"2026-01-02", "Ana", "12345678901", "987654321", "2", "ECO", "ana@al.insper.edu.br", "ana@gmail.com", "Qua 10h", "Qui 10h", ""},
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
		{NomeColuna: "LinkedIn", Indice: 10, Variavel: "linkedin"},
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
	if !reflect.DeepEqual(first.Usuario.Extras, map[string]string{"linkedin": "linkedin.com/in/maria"}) {
		t.Fatalf("unexpected extras: %#v", first.Usuario.Extras)
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

func TestBuildAvaliadoresWithMappingSupportsCurrentSchema(t *testing.T) {
	withTempWorkingDir(t, func(tmpDir string) {
		db := createTestDB(t, tmpDir)
		header := avaliadorHeaders()
		values := withPadding(avaliadorValues())

		workbook := createWorkbook(t,
			testSheet{name: "Candidatos", rows: [][]interface{}{{"A"}}},
			testSheet{
				name: "Avaliadores",
				rows: [][]interface{}{
					interfaceSlice(header),
					interfaceSlice(values),
					{"", "", ""},
				},
			},
		)

		mappingItems := expectedMappingItems(header, mappableAvaliadorFields())

		resp, err := BuildAvaliadoresWithMapping(workbook, mappingItems)
		if err != nil {
			t.Fatalf("BuildAvaliadoresWithMapping returned error: %v", err)
		}

		if len(resp.Avaliadores) != 1 {
			t.Fatalf("expected 1 avaliador, got %d", len(resp.Avaliadores))
		}
		got := resp.Avaliadores[1].Avaliador
		expected := expectedAvaliador()
		if !reflect.DeepEqual(got, expected) {
			t.Fatalf("unexpected avaliadores: %#v", got)
		}

		if err := Save([]types.Avaliador{got}); err != nil {
			t.Fatalf("Save returned error: %v", err)
		}

		if count := fetchCount(t, db, `SELECT COUNT(*) FROM avaliador`); count != 1 {
			t.Fatalf("expected 1 persisted avaliador, got %d", count)
		}
	})
}

func TestBuildUsuariosWithMappingStoresUnusedColumnsInExtras(t *testing.T) {
	workbook := createWorkbook(t, testSheet{
		name: "Candidatos",
		rows: [][]interface{}{
			{"Nome", "CPF", "Observacao", "Github"},
			{"Maria", "12345678901", "Aluna destaque", "github.com/maria"},
		},
	})

	mappingItems := []types.MappingItem{
		{NomeColuna: "Nome", Indice: 0, Variavel: "nome"},
		{NomeColuna: "CPF", Indice: 1, Variavel: "cpf"},
		{NomeColuna: "Observacao", Indice: 2, Variavel: "obs"},
		{NomeColuna: "Github", Indice: 3, Variavel: "git"},
	}

	resp, err := BuildUsuariosWithMapping(workbook, 0, mappingItems)
	if err != nil {
		t.Fatalf("BuildUsuariosWithMapping returned error: %v", err)
	}

	got := resp.Usuarios[1].Usuario.Extras
	expected := map[string]string{
		"obs": "Aluna destaque",
		"git": "github.com/maria",
	}
	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("unexpected extras: %#v", got)
	}
}

func TestBuildAvaliadoresWithMappingStoresUnusedColumnsInExtras(t *testing.T) {
	withTempWorkingDir(t, func(tmpDir string) {
		_ = createTestDB(t, tmpDir)

		workbook := createWorkbook(t,
			testSheet{name: "Candidatos", rows: [][]interface{}{{"A"}}},
			testSheet{
				name: "Avaliadores",
				rows: [][]interface{}{
					{"Nome", "Email", "Sigla", "LinkedIn"},
					{"Ana", "ana@insper.edu.br", "AN", "linkedin.com/in/ana"},
				},
			},
		)

		mappingItems := []types.MappingItem{
			{NomeColuna: "Nome", Indice: 0, Variavel: "nome"},
			{NomeColuna: "Email", Indice: 1, Variavel: "email"},
			{NomeColuna: "Sigla", Indice: 2, Variavel: "sigla"},
			{NomeColuna: "LinkedIn", Indice: 3, Variavel: "link"},
		}

		resp, err := BuildAvaliadoresWithMapping(workbook, mappingItems)
		if err != nil {
			t.Fatalf("BuildAvaliadoresWithMapping returned error: %v", err)
		}

		if len(resp.Avaliadores) != 1 {
			t.Fatalf("expected 1 avaliador, got %d", len(resp.Avaliadores))
		}
		got := resp.Avaliadores[1].Avaliador
		expected := types.Avaliador{
			Nome:  "Ana",
			Email: "ana@insper.edu.br",
			Sigla: "AN",
			Extras: map[string]string{
				"link": "linkedin.com/in/ana",
			},
		}
		if !reflect.DeepEqual(got, expected) {
			t.Fatalf("unexpected avaliadores with extras: %#v", got)
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
				{"Candidato", "Nao Posso", "Prefiro Nao", "Observacao"},
				{"Maria", "AB, CD", "EF", "ignorar"},
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

func TestBuildRestricoesWithMappingIgnoresUnusedColumns(t *testing.T) {
	workbook := createWorkbook(t,
		testSheet{name: "Candidatos", rows: [][]interface{}{{"A"}}},
		testSheet{name: "Avaliadores", rows: [][]interface{}{{"B"}}},
		testSheet{
			name: "Restricoes",
			rows: [][]interface{}{
				{"Candidato", "Nao Posso", "Prefiro Nao", "Observacao"},
				{"Maria", "AB", "CD", "ignorar"},
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

	expected := []types.Restricao{{Candidato: "Maria", NaoPosso: "AB", PrefiroNao: "CD"}}
	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("unexpected restricoes ignoring extras: %#v", got)
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

func TestSaveUsuariosFromMapsSupportsCurrentCandidateSchema(t *testing.T) {
	withTempWorkingDir(t, func(tmpDir string) {
		db := createTestDB(t, tmpDir)

		if err := SaveUsuariosFromMaps([]map[string]interface{}{candidateMapPayload()}); err != nil {
			t.Fatalf("SaveUsuariosFromMaps returned error: %v", err)
		}

		if count := fetchCount(t, db, `SELECT COUNT(*) FROM pessoa`); count != 1 {
			t.Fatalf("expected 1 persisted user, got %d", count)
		}

		expected := candidateExpectedDBValues()
		for _, field := range types.CandidateFields() {
			if !field.Persist {
				continue
			}

			var got sql.NullString
			query := fmt.Sprintf(`SELECT CAST("%s" AS TEXT) FROM pessoa LIMIT 1`, field.ColumnName)
			if err := db.QueryRow(query).Scan(&got); err != nil {
				t.Fatalf("failed to query persisted candidate field %s: %v", field.ColumnName, err)
			}

			if got.String != expected[field.ColumnName] {
				t.Fatalf("unexpected value for pessoa.%s: got %q want %q", field.ColumnName, got.String, expected[field.ColumnName])
			}
		}

		if count := fetchCount(t, db, `SELECT COUNT(*) FROM disponibilidade`); count != 2 {
			t.Fatalf("expected 2 disponibilidades, got %d", count)
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
	expected := mappableAvaliadorFields()

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

			fillDb(db, []types.Avaliador{
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

			avaliador := expectedAvaliador()
			avaliador.Extras = avaliadorExtrasPayload()

			err := Save([]types.Avaliador{avaliador})
			if err != nil {
				t.Fatalf("Save returned error: %v", err)
			}

			if count := fetchCount(t, db, `SELECT COUNT(*) FROM avaliador`); count != 1 {
				t.Fatalf("expected 1 avaliador, got %d", count)
			}

			expected := avaliadorExpectedDBValues()
			for _, field := range types.AvaliadorFields() {
				if !field.Persist {
					continue
				}

				var got sql.NullString
				query := fmt.Sprintf(`SELECT CAST("%s" AS TEXT) FROM avaliador LIMIT 1`, field.ColumnName)
				if err := db.QueryRow(query).Scan(&got); err != nil {
					t.Fatalf("failed to query persisted avaliador field %s: %v", field.ColumnName, err)
				}

				if got.String != expected[field.ColumnName] {
					t.Fatalf("unexpected value for avaliador.%s: got %q want %q", field.ColumnName, got.String, expected[field.ColumnName])
				}
			}
		})
	})

	t.Run("unsupported type", func(t *testing.T) {
		if err := Save("tipo-nao-suportado"); err != nil {
			t.Fatalf("expected unsupported type to return nil error, got %v", err)
		}
	})
}

func TestBuildCandidateFromRowSupportsCurrentCandidateSchema(t *testing.T) {
	header := append(candidateHeaders(2), "LinkedIn")
	row := append(candidateRowValues(2), "linkedin.com/in/maria")
	mapping := expectedMappingItems(header, append(candidateMappingVariables(2), "link"))

	got, err := buildCandidateFromRow(row, header, 2, mapping)
	if err != nil {
		t.Fatalf("buildCandidateFromRow returned error: %v", err)
	}

	expected := expectedCandidateForBuild()
	expected.Extras = map[string]string{"link": "linkedin.com/in/maria"}
	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("unexpected candidate built from row: %#v", got)
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

func TestBuildStructFromRowWithExtras(t *testing.T) {
	row := []string{"Ana", "ana@insper.edu.br", "AN", "linkedin.com/in/ana"}
	header := []string{"Nome", "Email", "Sigla", "LinkedIn"}
	mapping := []types.MappingItem{
		{Indice: 0, Variavel: "nome"},
		{Indice: 1, Variavel: "email"},
		{Indice: 2, Variavel: "sigla"},
		{Indice: 3, Variavel: "link"},
	}

	got, err := buildStructFromRowWithExtras[types.Avaliador](row, header, mapping)
	if err != nil {
		t.Fatalf("buildStructFromRowWithExtras returned error: %v", err)
	}

	expected := types.Avaliador{
		Nome:  "Ana",
		Email: "ana@insper.edu.br",
		Sigla: "AN",
		Extras: map[string]string{
			"link": "linkedin.com/in/ana",
		},
	}
	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("unexpected struct built with extras: %#v", got)
	}
}

func TestDecodeMapToStruct(t *testing.T) {
	got, err := decodeMapToStruct[types.Candidato](map[string]interface{}{
		"timestamp": "2026-01-01",
		"nome":      "Maria",
		"cpf":       "12345678901",
		"opcoes":    []string{"Seg 10h"},
		"extras": map[string]string{
			"linkedin": "linkedin.com/in/maria",
		},
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
	if !reflect.DeepEqual(got.Extras, map[string]string{"linkedin": "linkedin.com/in/maria"}) {
		t.Fatalf("unexpected decoded extras: %#v", got.Extras)
	}
}

func TestIsStructZeroValue(t *testing.T) {
	if !isStructZeroValue(types.Avaliador{}) {
		t.Fatal("expected empty struct to be zero value")
	}
	if isStructZeroValue(types.Avaliador{Nome: "Ana"}) {
		t.Fatal("expected non-empty struct to not be zero value")
	}
}

func TestGetRowsFromSheetInvalidWorkbook(t *testing.T) {
	_, err := getRowsFromSheet(bytes.Repeat([]byte("x"), 10), 0)
	if err == nil {
		t.Fatal("expected invalid workbook to return an error")
	}
}

func expectedMappingItems(headers []string, variables []string) []types.MappingItem {
	items := make([]types.MappingItem, 0, len(variables))
	for i, variable := range variables {
		column := ""
		if i < len(headers) {
			column = headers[i]
		}

		items = append(items, types.MappingItem{
			NomeColuna: column,
			Indice:     i,
			Variavel:   variable,
		})
	}
	return items
}

func mappingItemByVariable(t *testing.T, mappings []types.MappingItem, variable string) types.MappingItem {
	t.Helper()

	for _, item := range mappings {
		if item.Variavel == variable {
			return item
		}
	}

	t.Fatalf("mapping for variable %q not found", variable)
	return types.MappingItem{}
}

func assertSuggestedColumn(t *testing.T, mappings []types.MappingItem, variable string, column string, index int) {
	t.Helper()

	item := mappingItemByVariable(t, mappings, variable)
	if item.NomeColuna != column || item.Indice != index {
		t.Fatalf("unexpected mapping for %s: %#v", variable, item)
	}
}

func assertUnmappedVariable(t *testing.T, mappings []types.MappingItem, variable string, minIndex int) {
	t.Helper()

	item := mappingItemByVariable(t, mappings, variable)
	if item.NomeColuna != "" {
		t.Fatalf("expected %s to be unmapped, got %#v", variable, item)
	}
	if item.Indice < minIndex {
		t.Fatalf("expected unmapped index for %s to be outside header range, got %#v", variable, item)
	}
}

func assertUniqueMappedIndices(t *testing.T, mappings []types.MappingItem, headerCount int) {
	t.Helper()

	seen := make(map[int]string)
	for _, item := range mappings {
		if item.Indice < 0 || item.Indice >= headerCount {
			continue
		}
		if previous, exists := seen[item.Indice]; exists {
			t.Fatalf("duplicate mapped index %d for %s and %s", item.Indice, previous, item.Variavel)
		}
		seen[item.Indice] = item.Variavel
	}
}

func candidateMappingVariables(optionCount int) []string {
	var variables []string
	for _, field := range mappableCandidateFields() {
		if field == "opcoes" {
			for i := 1; i <= optionCount; i++ {
				variables = append(variables, fmt.Sprintf("opcao %d", i))
			}
			continue
		}
		variables = append(variables, field)
	}
	return variables
}

func candidateHeaders(optionCount int) []string {
	var headers []string
	for _, variable := range candidateMappingVariables(optionCount) {
		headers = append(headers, candidateHeaderForVariable(variable))
	}
	return headers
}

func candidateHeaderForVariable(variable string) string {
	switch variable {
	case "timestamp":
		return "Timestamp"
	case "nome":
		return "Nome"
	case "cpf":
		return "CPF"
	case "numero":
		return "Numero"
	case "semestre":
		return "Semestre"
	case "curso":
		return "Curso"
	case "email_secundario":
		return "Email Secundario"
	case "email_pessoal":
		return "Email Pessoal"
	default:
		if strings.HasPrefix(variable, "opcao ") {
			return "Opcao " + strings.TrimPrefix(variable, "opcao ")
		}
		return strings.Title(strings.ReplaceAll(variable, "_", " "))
	}
}

func candidateRowValues(optionCount int) []string {
	var values []string
	for _, variable := range candidateMappingVariables(optionCount) {
		values = append(values, candidateValueForVariable(variable))
	}
	return values
}

func candidateValueForVariable(variable string) string {
	switch variable {
	case "timestamp":
		return "2026-01-01"
	case "nome":
		return "Maria"
	case "cpf":
		return "12345678901"
	case "numero":
		return "123456789"
	case "semestre":
		return "2"
	case "curso":
		return "ADM"
	case "email_secundario":
		return "maria@al.insper.edu.br"
	case "email_pessoal":
		return "maria@gmail.com"
	default:
		if strings.HasPrefix(variable, "opcao ") {
			return "Horario " + strings.TrimPrefix(variable, "opcao ")
		}
		return fmt.Sprintf("%s_value", variable)
	}
}

func expectedCandidateForBuild() types.Candidato {
	candidate := types.Candidato{}
	rv := reflect.ValueOf(&candidate).Elem()
	for _, field := range types.CandidateFields() {
		if field.JSONName == "opcoes" || field.JSONName == "extras" {
			continue
		}
		rv.Field(field.Index).SetString(candidateValueForVariable(field.JSONName))
	}
	candidate.Opcoes = []string{"Horario 1", "Horario 2"}
	return candidate
}

func candidateMapPayload() map[string]interface{} {
	payload := make(map[string]interface{})
	for _, field := range types.CandidateFields() {
		if field.JSONName == "opcoes" {
			continue
		}
		if field.JSONName == "extras" {
			payload[field.JSONName] = candidateExtrasPayload()
			continue
		}
		payload[field.JSONName] = candidateValueForVariable(field.JSONName)
	}
	payload["opcoes"] = []interface{}{" Seg 10h ", "Ter 14h"}
	return payload
}

func candidateExpectedDBValues() map[string]string {
	expected := make(map[string]string)
	for _, field := range types.CandidateFields() {
		if !field.Persist {
			continue
		}

		if field.JSONName == "extras" {
			expected[field.ColumnName] = candidateExtrasJSON()
			continue
		}
		value := candidateValueForVariable(field.JSONName)
		if field.SQLiteType == "INTEGER" {
			value = strconv.Itoa(mustAtoi(value))
		}
		expected[field.ColumnName] = value
	}
	return expected
}

func avaliadorHeaders() []string {
	fields := mappableAvaliadorFields()
	headers := make([]string, 0, len(fields))
	for _, field := range fields {
		switch field {
		case "nome":
			headers = append(headers, "Nome")
		case "email":
			headers = append(headers, "Email")
		case "sigla":
			headers = append(headers, "Sigla")
		default:
			headers = append(headers, strings.Title(strings.ReplaceAll(field, "_", " ")))
		}
	}
	return headers
}

func avaliadorValues() []string {
	fields := mappableAvaliadorFields()
	values := make([]string, 0, len(fields))
	for _, field := range fields {
		values = append(values, avaliadorValueForField(field))
	}
	return values
}

func avaliadorValueForField(field string) string {
	switch field {
	case "nome":
		return "Ana"
	case "email":
		return "ana@insper.edu.br"
	case "sigla":
		return "AN"
	default:
		return fmt.Sprintf("%s_value", field)
	}
}

func expectedAvaliador() types.Avaliador {
	avaliador := types.Avaliador{}
	rv := reflect.ValueOf(&avaliador).Elem()
	for _, field := range types.AvaliadorFields() {
		if field.JSONName == "extras" || field.JSONName == "id" {
			continue
		}
		if field.Kind == reflect.String {
			rv.Field(field.Index).SetString(avaliadorValueForField(field.JSONName))
		}
	}
	return avaliador
}

func avaliadorExpectedDBValues() map[string]string {
	expected := make(map[string]string)
	for _, field := range types.AvaliadorFields() {
		if !field.Persist || field.JSONName == "id" {
			continue
		}
		if field.JSONName == "extras" {
			expected[field.ColumnName] = avaliadorExtrasJSON()
			continue
		}
		expected[field.ColumnName] = avaliadorValueForField(field.JSONName)
	}
	return expected
}

func mappableCandidateFields() []string {
	fields := types.JSONFieldNames(types.Candidato{})
	filtered := make([]string, 0, len(fields))
	for _, field := range fields {
		if field == "extras" {
			continue
		}
		filtered = append(filtered, field)
	}
	return filtered
}

func mappableAvaliadorFields() []string {
	fields := types.JSONFieldNames(types.Avaliador{})
	filtered := make([]string, 0, len(fields))
	for _, field := range fields {
		if field == "extras" || field == "id" {
			continue
		}
		filtered = append(filtered, field)
	}
	return filtered
}

func candidateExtrasPayload() map[string]string {
	return map[string]string{
		"linkedin":      "linkedin.com/in/maria",
		"empresa_atual": "XP",
	}
}

func candidateExtrasJSON() string {
	return `{"empresa_atual":"XP","linkedin":"linkedin.com/in/maria"}`
}

func avaliadorExtrasPayload() map[string]string {
	return map[string]string{
		"area": "Financas",
	}
}

func avaliadorExtrasJSON() string {
	return `{"area":"Financas"}`
}

func interfaceSlice(values []string) []interface{} {
	result := make([]interface{}, 0, len(values))
	for _, value := range values {
		result = append(result, value)
	}
	return result
}

func withPadding(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, " "+value+" ")
	}
	return result
}

func mustAtoi(value string) int {
	number, err := strconv.Atoi(value)
	if err != nil {
		panic(err)
	}
	return number
}
