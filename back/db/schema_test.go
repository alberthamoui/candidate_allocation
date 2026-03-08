package db

import (
	types "candidate_alocator/back/type"
	"testing"
)

func TestEnsureAppSchemaCreatesAllCandidateColumns(t *testing.T) {
	db := openTempSQLiteDB(t)

	if err := EnsureAppSchema(db); err != nil {
		t.Fatalf("EnsureAppSchema returned error: %v", err)
	}

	columns := fetchColumnSet(t, db, "pessoa")
	for _, field := range persistedFields(types.CandidateFields()) {
		if _, ok := columns[field.ColumnName]; !ok {
			t.Fatalf("expected column %s to exist in pessoa", field.ColumnName)
		}
	}

	indexes := fetchIndexSet(t, db, "pessoa")
	for _, field := range persistedFields(types.CandidateFields()) {
		if !field.Unique {
			continue
		}
		if _, ok := indexes[uniqueIndexName("pessoa", field)]; !ok {
			t.Fatalf("expected unique index for pessoa.%s", field.ColumnName)
		}
	}
}

func TestEnsureAppSchemaCreatesAllAvaliadorColumns(t *testing.T) {
	db := openTempSQLiteDB(t)

	if err := EnsureAppSchema(db); err != nil {
		t.Fatalf("EnsureAppSchema returned error: %v", err)
	}

	columns := fetchColumnSet(t, db, "avaliador")
	for _, field := range persistedFields(types.AvaliadorFields()) {
		if _, ok := columns[field.ColumnName]; !ok {
			t.Fatalf("expected column %s to exist in avaliador", field.ColumnName)
		}
	}

	indexes := fetchIndexSet(t, db, "avaliador")
	for _, field := range persistedFields(types.AvaliadorFields()) {
		if !field.Unique {
			continue
		}
		if _, ok := indexes[uniqueIndexName("avaliador", field)]; !ok {
			t.Fatalf("expected unique index for avaliador.%s", field.ColumnName)
		}
	}
}

func TestEnsureAppSchemaAddsMissingCandidateColumnsToExistingDatabase(t *testing.T) {
	db := openTempSQLiteDB(t)
	fields := types.CandidateFields()
	missing := lastPersistedField(t, fields)

	if _, err := db.Exec(buildLegacyCreateTableStatement("pessoa", fields, missing.ColumnName)); err != nil {
		t.Fatalf("failed to create legacy pessoa table: %v", err)
	}
	insertLegacyRow(t, db, "pessoa", fields, missing.ColumnName)

	if err := EnsureAppSchema(db); err != nil {
		t.Fatalf("EnsureAppSchema returned error: %v", err)
	}

	columns := fetchColumnSet(t, db, "pessoa")
	if _, ok := columns[missing.ColumnName]; !ok {
		t.Fatalf("expected missing column %s to be added to pessoa", missing.ColumnName)
	}
	if count := fetchCount(t, db, `SELECT COUNT(*) FROM pessoa`); count != 1 {
		t.Fatalf("expected legacy row to survive migration, got %d rows", count)
	}

	indexes := fetchIndexSet(t, db, "pessoa")
	for _, field := range persistedFields(fields) {
		if !field.Unique {
			continue
		}
		if _, ok := indexes[uniqueIndexName("pessoa", field)]; !ok {
			t.Fatalf("expected unique index for pessoa.%s after migration", field.ColumnName)
		}
	}
}

func TestEnsureAppSchemaAddsMissingAvaliadorColumnsToExistingDatabase(t *testing.T) {
	db := openTempSQLiteDB(t)
	fields := types.AvaliadorFields()
	missing := lastPersistedField(t, fields)

	if _, err := db.Exec(buildLegacyCreateTableStatement("avaliador", fields, missing.ColumnName)); err != nil {
		t.Fatalf("failed to create legacy avaliador table: %v", err)
	}
	insertLegacyRow(t, db, "avaliador", fields, missing.ColumnName)

	if err := EnsureAppSchema(db); err != nil {
		t.Fatalf("EnsureAppSchema returned error: %v", err)
	}

	columns := fetchColumnSet(t, db, "avaliador")
	if _, ok := columns[missing.ColumnName]; !ok {
		t.Fatalf("expected missing column %s to be added to avaliador", missing.ColumnName)
	}
	if count := fetchCount(t, db, `SELECT COUNT(*) FROM avaliador`); count != 1 {
		t.Fatalf("expected legacy row to survive migration, got %d rows", count)
	}

	indexes := fetchIndexSet(t, db, "avaliador")
	for _, field := range persistedFields(fields) {
		if !field.Unique {
			continue
		}
		if _, ok := indexes[uniqueIndexName("avaliador", field)]; !ok {
			t.Fatalf("expected unique index for avaliador.%s after migration", field.ColumnName)
		}
	}
}
