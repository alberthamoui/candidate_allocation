package types

import (
	"fmt"
	"reflect"
	"strings"
)

type FieldSchema struct {
	Index      int
	GoName     string
	JSONName   string
	ColumnName string
	SQLiteType string
	Required   bool
	Unique     bool
	Duplicate  bool
	Persist    bool
	Kind       reflect.Kind
}

func CandidateFields() []FieldSchema {
	return DescribeStruct(Candidato{})
}

func CandidateDuplicateFieldNames() []string {
	fields := CandidateFields()
	duplicates := make([]string, 0, len(fields))
	for _, field := range fields {
		if field.Duplicate {
			duplicates = append(duplicates, field.JSONName)
		}
	}
	return duplicates
}

func AvaliadorFields() []FieldSchema {
	return DescribeStruct(Avaliador{})
}

func AvaliadorDuplicateFieldNames() []string {
	fields := AvaliadorFields()
	duplicates := make([]string, 0, len(fields))
	for _, field := range fields {
		if field.Duplicate {
			duplicates = append(duplicates, field.JSONName)
		}
	}
	return duplicates
}

func RestricaoFieldNames() []string {
	return JSONFieldNames(Restricao{})
}

func JSONFieldNames(model interface{}) []string {
	typ := reflect.TypeOf(model)
	if typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}

	var fields []string
	for i := 0; i < typ.NumField(); i++ {
		jsonName := jsonTagName(typ.Field(i))
		if jsonName == "" {
			continue
		}
		fields = append(fields, jsonName)
	}
	return fields
}

func DescribeStruct(model interface{}) []FieldSchema {
	typ := reflect.TypeOf(model)
	if typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}

	var fields []FieldSchema
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		jsonName := jsonTagName(field)
		if jsonName == "" {
			continue
		}

		dbMeta := parseDBTag(field.Tag.Get("db"))
		appMeta := parseAppTag(field.Tag.Get("app"))
		if dbMeta.skip {
			fields = append(fields, FieldSchema{
				Index:     i,
				GoName:    field.Name,
				JSONName:  jsonName,
				Duplicate: appMeta.duplicate,
				Persist:   false,
				Kind:      field.Type.Kind(),
			})
			continue
		}

		sqliteType := dbMeta.sqliteType
		if sqliteType == "" {
			sqliteType = inferSQLiteType(field.Type)
		}

		fields = append(fields, FieldSchema{
			Index:      i,
			GoName:     field.Name,
			JSONName:   jsonName,
			ColumnName: jsonName,
			SQLiteType: sqliteType,
			Required:   dbMeta.required,
			Unique:     dbMeta.unique,
			Duplicate:  appMeta.duplicate,
			Persist:    sqliteType != "",
			Kind:       field.Type.Kind(),
		})
	}

	return fields
}

func jsonTagName(field reflect.StructField) string {
	tag := field.Tag.Get("json")
	if tag == "" || tag == "-" {
		return ""
	}

	name := strings.Split(tag, ",")[0]
	if name == "" || name == "-" {
		return ""
	}
	return name
}

type dbTagMeta struct {
	skip       bool
	sqliteType string
	required   bool
	unique     bool
}

type appTagMeta struct {
	duplicate bool
}

func parseDBTag(tag string) dbTagMeta {
	if tag == "-" {
		return dbTagMeta{skip: true}
	}

	var meta dbTagMeta
	for _, part := range strings.Split(tag, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		switch {
		case part == "required":
			meta.required = true
		case part == "unique":
			meta.unique = true
		case strings.HasPrefix(part, "type="):
			meta.sqliteType = strings.TrimSpace(strings.TrimPrefix(part, "type="))
		default:
			panic(fmt.Sprintf("unsupported db tag option %q", part))
		}
	}

	return meta
}

func parseAppTag(tag string) appTagMeta {
	var meta appTagMeta
	for _, part := range strings.Split(tag, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		switch part {
		case "duplicate":
			meta.duplicate = true
		default:
			panic(fmt.Sprintf("unsupported app tag option %q", part))
		}
	}

	return meta
}

func inferSQLiteType(typ reflect.Type) string {
	switch typ.Kind() {
	case reflect.String:
		return "TEXT"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return "INTEGER"
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "INTEGER"
	case reflect.Float32, reflect.Float64:
		return "REAL"
	case reflect.Bool:
		return "INTEGER"
	default:
		return ""
	}
}
