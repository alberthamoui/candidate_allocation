package main

import (
	dbpkg "candidate_alocator/back/db"
	types "candidate_alocator/back/type"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReviewedImportKeepsPreviousDataUntilCompleteAndCanRetry(t *testing.T) {
	t.Setenv("CANDIDATE_ALLOCATOR_DATABASE", filepath.Join(t.TempDir(), "import.db"))
	if err := dbpkg.EnsureDefaultDatabase(); err != nil {
		t.Fatal(err)
	}
	db, err := dbpkg.OpenDefault()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	if _, err := dbpkg.InsertStruct(db, "pessoa", types.Candidato{Nome: "Anterior", Semestre: "1"}, types.CandidateFields()); err != nil {
		t.Fatal(err)
	}
	a := NewApp()
	defer a.clearTransientState()
	data, err := os.ReadFile("Execelteste/Base4Restricao.xlsx")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.SuggestMapping(data, 5); err != nil {
		t.Fatal(err)
	}
	if err := a.SaveUsuariosFromMaps([]map[string]interface{}{{"nome": "Nova", "semestre": "1", "opcoes": []string{"quarta 14-16"}}}); err != nil {
		t.Fatal(err)
	}
	if err := a.SaveRestricoesFromMaps([]map[string]interface{}{{"candidato": "Nova", "naoPosso": "EV"}}); err != nil {
		t.Fatal(err)
	}
	var name string
	if err := db.QueryRow("SELECT nome FROM pessoa").Scan(&name); err != nil || name != "Anterior" {
		t.Fatal(name, err)
	}
	if err := a.SaveAvaliadoresFromMaps([]map[string]interface{}{{"nome": "Eva", "sigla": "ERRADA", "email": "eva@example.test"}}); err == nil || !strings.Contains(err.Error(), "registro 1") {
		t.Fatal(err)
	}
	if err := db.QueryRow("SELECT nome FROM pessoa").Scan(&name); err != nil || name != "Anterior" {
		t.Fatal("previous import changed", name, err)
	}
	if err := a.SaveAvaliadoresFromMaps([]map[string]interface{}{{"nome": "Eva", "sigla": "EV", "email": "eva@example.test"}}); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow("SELECT nome FROM pessoa").Scan(&name); err != nil || name != "Nova" {
		t.Fatal(name, err)
	}
	if err := a.SaveAvaliadoresFromMaps([]map[string]interface{}{{"nome": "Eva", "sigla": "EV", "email": "eva@example.test"}}); err != nil {
		t.Fatal(err)
	}
	var restrictions int
	if err := db.QueryRow("SELECT COUNT(*) FROM restricoesNposso").Scan(&restrictions); err != nil || restrictions != 1 {
		t.Fatal("review lost restrictions", restrictions, err)
	}
	mappings, err := a.GetPreferenceScheduleMappings()
	if err != nil || len(mappings) != 1 || mappings[0].Hora != "14:00-16:00" {
		t.Fatal(mappings, err)
	}
	params := a.DefaultAllocationParams()
	params.MinPessoasPorGrupo = 1
	params.ScheduleMappings = mappings
	config, err := a.BuildAllocationConfigurationFromDatabase(params)
	if err != nil || config.Normalized.PreferenceMappings[0].Dia != "quarta" {
		t.Fatal(config, err)
	}
	params.ScheduleMappings[0].Hora = "16:00-14:00"
	if _, err := a.BuildAllocationConfigurationFromDatabase(params); err == nil {
		t.Fatal("invalid interval accepted")
	}
}
