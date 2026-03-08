package db

import (
	types "candidate_alocator/back/type"
	"testing"
)

func TestInsertStructPersistsAllCandidateFields(t *testing.T) {
	db := openTempSQLiteDB(t)
	if err := EnsureAppSchema(db); err != nil {
		t.Fatalf("EnsureAppSchema returned error: %v", err)
	}

	candidate, expected := populateStructForFields[types.Candidato](t, types.CandidateFields())
	id, err := InsertStruct(db, "pessoa", candidate, types.CandidateFields())
	if err != nil {
		t.Fatalf("InsertStruct returned error: %v", err)
	}

	for column, want := range expected {
		if got := selectColumnAsText(t, db, "pessoa", column, id); got != want {
			t.Fatalf("unexpected persisted value for pessoa.%s: got %q want %q", column, got, want)
		}
	}
}

func TestInsertStructPersistsAllAvaliadorFields(t *testing.T) {
	db := openTempSQLiteDB(t)
	if err := EnsureAppSchema(db); err != nil {
		t.Fatalf("EnsureAppSchema returned error: %v", err)
	}

	avaliador, expected := populateStructForFields[types.AvaliadorInfo](t, types.AvaliadorFields())
	id, err := InsertStruct(db, "avaliador", avaliador, types.AvaliadorFields())
	if err != nil {
		t.Fatalf("InsertStruct returned error: %v", err)
	}

	for column, want := range expected {
		if got := selectColumnAsText(t, db, "avaliador", column, id); got != want {
			t.Fatalf("unexpected persisted value for avaliador.%s: got %q want %q", column, got, want)
		}
	}
}
