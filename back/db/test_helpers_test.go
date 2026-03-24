package db

import (
	types "candidate_alocator/back/type"
	"database/sql"
	"encoding/json"
	"fmt"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func openTempSQLiteDB(t *testing.T) *sql.DB {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("failed to open sqlite db: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})
	return db
}

func persistedFields(fields []types.FieldSchema) []types.FieldSchema {
	persisted := make([]types.FieldSchema, 0, len(fields))
	for _, field := range fields {
		if field.Persist {
			persisted = append(persisted, field)
		}
	}
	return persisted
}

func fetchColumnSet(t *testing.T, db *sql.DB, table string) map[string]struct{} {
	t.Helper()

	rows, err := db.Query(fmt.Sprintf(`PRAGMA table_info("%s")`, table))
	if err != nil {
		t.Fatalf("failed to inspect table %s: %v", table, err)
	}
	defer rows.Close()

	columns := make(map[string]struct{})
	for rows.Next() {
		var (
			cid        int
			name       string
			columnType string
			notNull    int
			defaultVal any
			pk         int
		)
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultVal, &pk); err != nil {
			t.Fatalf("failed to scan PRAGMA table_info row: %v", err)
		}
		columns[name] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("failed to iterate table info: %v", err)
	}

	return columns
}

func fetchIndexSet(t *testing.T, db *sql.DB, table string) map[string]struct{} {
	t.Helper()

	rows, err := db.Query(fmt.Sprintf(`PRAGMA index_list("%s")`, table))
	if err != nil {
		t.Fatalf("failed to inspect indexes for %s: %v", table, err)
	}
	defer rows.Close()

	indexes := make(map[string]struct{})
	for rows.Next() {
		var (
			seq     int
			name    string
			unique  int
			origin  string
			partial int
		)
		if err := rows.Scan(&seq, &name, &unique, &origin, &partial); err != nil {
			t.Fatalf("failed to scan PRAGMA index_list row: %v", err)
		}
		indexes[name] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("failed to iterate indexes: %v", err)
	}

	return indexes
}

func fetchCount(t *testing.T, db *sql.DB, query string) int {
	t.Helper()

	var count int
	if err := db.QueryRow(query).Scan(&count); err != nil {
		t.Fatalf("failed to query count: %v", err)
	}
	return count
}

func selectColumnAsText(t *testing.T, db *sql.DB, table, column string, id int64) string {
	t.Helper()

	var value sql.NullString
	query := fmt.Sprintf(`SELECT CAST("%s" AS TEXT) FROM "%s" WHERE id = ?`, column, table)
	if err := db.QueryRow(query, id).Scan(&value); err != nil {
		t.Fatalf("failed to query %s.%s: %v", table, column, err)
	}
	if !value.Valid {
		return ""
	}
	return value.String
}

func uniqueIndexName(table string, field types.FieldSchema) string {
	return fmt.Sprintf("idx_%s_%s_unique", table, field.ColumnName)
}

func firstUniqueField(t *testing.T, fields []types.FieldSchema) types.FieldSchema {
	t.Helper()

	for _, field := range fields {
		if field.Persist && field.Unique {
			return field
		}
	}

	t.Fatal("expected at least one unique field")
	return types.FieldSchema{}
}

func firstNonUniqueField(t *testing.T, fields []types.FieldSchema) types.FieldSchema {
	t.Helper()

	for _, field := range fields {
		if field.Persist && !field.Unique {
			return field
		}
	}

	t.Fatal("expected at least one non-unique field")
	return types.FieldSchema{}
}

func createUniqueIndex(t *testing.T, db *sql.DB, table, column string) {
	t.Helper()

	stmt := fmt.Sprintf(
		`CREATE UNIQUE INDEX "%s" ON "%s" ("%s") WHERE "%s" IS NOT NULL`,
		fmt.Sprintf("idx_%s_%s_unique", table, column),
		table,
		column,
		column,
	)
	if _, err := db.Exec(stmt); err != nil {
		t.Fatalf("failed to create unique index %s.%s: %v", table, column, err)
	}
}

func lastPersistedField(t *testing.T, fields []types.FieldSchema) types.FieldSchema {
	t.Helper()

	persisted := persistedFields(fields)
	if len(persisted) == 0 {
		t.Fatal("expected at least one persisted field")
	}
	return persisted[len(persisted)-1]
}

func buildLegacyCreateTableStatement(table string, fields []types.FieldSchema, omitted string) string {
	definitions := []string{`"id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT`}
	for _, field := range fields {
		if !field.Persist || field.ColumnName == omitted {
			continue
		}
		definition := fmt.Sprintf(`"%s" %s`, field.ColumnName, field.SQLiteType)
		if field.Required {
			definition += ` NOT NULL`
		}
		definitions = append(definitions, definition)
	}

	return fmt.Sprintf(`CREATE TABLE "%s" (%s);`, table, strings.Join(definitions, ", "))
}

func insertLegacyRow(t *testing.T, db *sql.DB, table string, fields []types.FieldSchema, omitted string) {
	t.Helper()

	var (
		columns []string
		values  []string
		args    []any
	)

	for idx, field := range fields {
		if !field.Persist || field.ColumnName == omitted {
			continue
		}
		columns = append(columns, fmt.Sprintf(`"%s"`, field.ColumnName))
		values = append(values, "?")
		args = append(args, syntheticDBValue(field, idx))
	}

	query := fmt.Sprintf(`INSERT INTO "%s" (%s) VALUES (%s)`, table, strings.Join(columns, ", "), strings.Join(values, ", "))
	if _, err := db.Exec(query, args...); err != nil {
		t.Fatalf("failed to seed legacy row in %s: %v", table, err)
	}
}

func populateStructForFields[T any](t *testing.T, fields []types.FieldSchema) (T, map[string]string) {
	t.Helper()

	var value T
	rv := reflect.ValueOf(&value).Elem()
	expected := make(map[string]string)

	for idx, field := range fields {
		if !field.Persist {
			continue
		}

		fieldValue := rv.Field(field.Index)
		if !fieldValue.CanSet() {
			t.Fatalf("field %s cannot be set", field.GoName)
		}

		expected[field.ColumnName] = setSyntheticFieldValue(t, field, fieldValue, idx)
	}

	return value, expected
}

func setSyntheticFieldValue(t *testing.T, field types.FieldSchema, dest reflect.Value, idx int) string {
	t.Helper()

	switch dest.Kind() {
	case reflect.String:
		if field.SQLiteType == "INTEGER" {
			value := strconv.Itoa(100 + idx)
			dest.SetString(value)
			return value
		}
		value := fmt.Sprintf("%s_value_%d", field.JSONName, idx)
		dest.SetString(value)
		return value
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		value := int64(100 + idx)
		dest.SetInt(value)
		return strconv.FormatInt(value, 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		value := uint64(100 + idx)
		dest.SetUint(value)
		return strconv.FormatUint(value, 10)
	case reflect.Float32, reflect.Float64:
		value := 10.5 + float64(idx)
		dest.SetFloat(value)
		return strconv.FormatFloat(value, 'f', -1, 64)
	case reflect.Bool:
		dest.SetBool(true)
		return "1"
	case reflect.Map:
		if dest.Type().Elem().Kind() == reflect.Ptr {
			raw := fmt.Sprintf("valor_%d", idx)
			nullable := types.NullableString(raw)
			value := map[string]*types.NullableString{
				fmt.Sprintf("%s_extra_%d", field.JSONName, idx): &nullable,
			}
			dest.Set(reflect.ValueOf(value))
			payload, err := json.Marshal(value)
			if err != nil {
				t.Fatalf("failed to marshal synthetic map for %s: %v", field.GoName, err)
			}
			return string(payload)
		}

		value := map[string]string{
			fmt.Sprintf("%s_extra_%d", field.JSONName, idx): fmt.Sprintf("valor_%d", idx),
		}
		dest.Set(reflect.ValueOf(value))
		payload, err := json.Marshal(value)
		if err != nil {
			t.Fatalf("failed to marshal synthetic map for %s: %v", field.GoName, err)
		}
		return string(payload)
	default:
		t.Fatalf("unsupported field kind %s for %s", dest.Kind(), field.GoName)
		return ""
	}
}

func syntheticDBValue(field types.FieldSchema, idx int) any {
	if field.SQLiteType == "INTEGER" {
		return 100 + idx
	}
	if field.Kind == reflect.Map {
		payload, err := json.Marshal(map[string]string{
			fmt.Sprintf("%s_legacy_%d", field.JSONName, idx): fmt.Sprintf("valor_legacy_%d", idx),
		})
		if err != nil {
			panic(err)
		}
		return string(payload)
	}
	return fmt.Sprintf("%s_legacy_%d", field.JSONName, idx)
}
