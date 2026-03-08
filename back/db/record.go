package db

import (
	types "candidate_alocator/back/type"
	"database/sql"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

func InsertStruct(db *sql.DB, table string, value interface{}, fields []types.FieldSchema) (int64, error) {
	rv := reflect.ValueOf(value)
	if rv.Kind() == reflect.Pointer {
		rv = rv.Elem()
	}

	var (
		columns      []string
		placeholders []string
		args         []interface{}
	)

	for _, field := range fields {
		if !field.Persist {
			continue
		}

		fieldValue := rv.Field(field.Index)
		arg, err := valueForDB(field, fieldValue)
		if err != nil {
			return 0, err
		}

		columns = append(columns, fmt.Sprintf(`"%s"`, field.ColumnName))
		placeholders = append(placeholders, "?")
		args = append(args, arg)
	}

	stmt := fmt.Sprintf(
		`INSERT INTO "%s" (%s) VALUES (%s)`,
		table,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
	)

	res, err := db.Exec(stmt, args...)
	if err != nil {
		return 0, err
	}

	return res.LastInsertId()
}

func valueForDB(field types.FieldSchema, value reflect.Value) (interface{}, error) {
	if !value.IsValid() {
		return nil, nil
	}

	switch field.SQLiteType {
	case "INTEGER":
		if value.Kind() == reflect.String {
			raw := strings.TrimSpace(value.String())
			if raw == "" {
				return nil, nil
			}

			number, err := strconv.Atoi(raw)
			if err != nil {
				return nil, fmt.Errorf("campo %s precisa ser numérico: %w", field.JSONName, err)
			}
			return number, nil
		}
	}

	switch value.Kind() {
	case reflect.String:
		return value.String(), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return value.Int(), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return int64(value.Uint()), nil
	case reflect.Float32, reflect.Float64:
		return value.Float(), nil
	case reflect.Bool:
		if value.Bool() {
			return 1, nil
		}
		return 0, nil
	default:
		return nil, nil
	}
}
