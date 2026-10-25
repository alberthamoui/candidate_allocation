package logic

import (
	dbpkg "candidate_alocator/back/db"
	types "candidate_alocator/back/type"
	"path/filepath"
	"strings"
	"testing"
)

func TestReplaceImportRollsBackEveryTableOnLateFailure(t *testing.T) {
	db := createTestDB(t, t.TempDir())
	old := ImportData{Candidatos: []types.Candidato{{Nome: "Anterior", Semestre: "1", Opcoes: []string{"quarta 14-16"}}}}
	if err := ReplaceImport(db, old); err != nil {
		t.Fatal(err)
	}
	for _, invalid := range []ImportData{
		{Candidatos: []types.Candidato{{Nome: "Nova", Semestre: "1"}}, Avaliadores: []types.Avaliador{{Nome: "Eva", Email: "eva@example.test", Sigla: "EV"}}, Restricoes: []types.Restricao{{Candidato: "Nova", NaoPosso: "INEXISTENTE"}}},
		{Candidatos: []types.Candidato{{Nome: "Nova", Semestre: "1", Opcoes: []string{"sexta 10-12"}}, {Nome: "Inválida", Semestre: "texto"}}},
		{Candidatos: []types.Candidato{{Nome: "Nova", Semestre: "1"}}, Avaliadores: []types.Avaliador{{Nome: "Eva", Email: "eva@example.test", Sigla: "EV"}, {Nome: "Outra", Email: "outro@example.test", Sigla: "EV"}}},
		{},
	} {
		if err := ReplaceImport(db, invalid); err == nil {
			t.Fatal("invalid import was committed")
		}
		if got := fetchStrings(t, db, "SELECT nome FROM pessoa"); len(got) != 1 || got[0] != "Anterior" {
			t.Fatal("previous import lost", got)
		}
		if fetchCount(t, db, "SELECT COUNT(*) FROM disponibilidade") != 1 || fetchCount(t, db, "SELECT COUNT(*) FROM opcoes_horario") != 1 || fetchCount(t, db, "SELECT COUNT(*) FROM avaliador") != 0 {
			t.Fatal("partial writes escaped rollback")
		}
	}
	valid := ImportData{Candidatos: []types.Candidato{{Nome: "Nova", Semestre: "1"}}, Avaliadores: []types.Avaliador{{Nome: "Eva", Email: "eva@example.test", Sigla: "EV"}}, Restricoes: []types.Restricao{{Candidato: "Nova", NaoPosso: "EV, EV"}}}
	if err := ReplaceImport(db, valid); err != nil {
		t.Fatal(err)
	}
	if got := fetchStrings(t, db, "SELECT nome FROM pessoa"); len(got) != 1 || got[0] != "Nova" || fetchCount(t, db, "SELECT COUNT(*) FROM restricoesNposso") != 1 {
		t.Fatal("replacement not committed", got)
	}
}

func TestSaveBatchAndAvailabilityAreAtomic(t *testing.T) {
	t.Setenv("CANDIDATE_ALLOCATOR_DATABASE", filepath.Join(t.TempDir(), "import.db"))
	if err := dbpkg.EnsureDefaultDatabase(); err != nil {
		t.Fatal(err)
	}
	db, err := dbpkg.OpenDefault()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	if _, err := db.Exec(`CREATE TRIGGER fail_availability BEFORE INSERT ON disponibilidade BEGIN SELECT RAISE(ABORT, 'indisponível'); END`); err != nil {
		t.Fatal(err)
	}
	err = Save([]types.Candidato{{Nome: "Nova", Semestre: "1", Opcoes: []string{"quarta 14-16"}}})
	if err == nil || !strings.Contains(err.Error(), "registro 1, preferência 1") {
		t.Fatal(err)
	}
	if fetchCount(t, db, "SELECT COUNT(*) FROM pessoa") != 0 || fetchCount(t, db, "SELECT COUNT(*) FROM opcoes_horario") != 0 {
		t.Fatal("partial batch committed")
	}
}

func TestDecodeImportMapsAndAmbiguousRestriction(t *testing.T) {
	data, err := DecodeImportMaps([]map[string]interface{}{{"nome": "Ana", "semestre": "1"}}, []map[string]interface{}{{"nome": "Eva", "sigla": "EV", "email": "eva@example.test"}}, []map[string]interface{}{{"candidato": "Ana", "naoPosso": "EV"}})
	if err != nil || len(data.Candidatos) != 1 || len(data.Avaliadores) != 1 || len(data.Restricoes) != 1 {
		t.Fatal(data, err)
	}
	for i := 0; i < 3; i++ {
		batches := make([][]map[string]interface{}, 3)
		batches[i] = []map[string]interface{}{{"nome": func() {}, "candidato": func() {}}}
		if _, err := DecodeImportMaps(batches[0], batches[1], batches[2]); err == nil {
			t.Fatal("invalid payload accepted")
		}
	}
	db := createTestDB(t, t.TempDir())
	data.Candidatos = append(data.Candidatos, types.Candidato{Nome: "Ana", Semestre: "2", EmailPessoal: "outra@example.test", EmailSecundario: "outra@al.insper.edu.br", Numero: "2"})
	if err := ReplaceImport(db, data); err == nil {
		t.Fatal("ambiguous name accepted")
	}
}
