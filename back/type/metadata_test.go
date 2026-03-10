package types

import (
	"reflect"
	"slices"
	"testing"
)

func TestCandidateFieldMetadataInvariants(t *testing.T) {
	fields := CandidateFields()
	assertFieldMetadataInvariants(t, reflect.TypeOf(Candidato{}), fields)

	expectedDuplicates := duplicateJSONFieldsFromTags(t, reflect.TypeOf(Candidato{}))
	gotDuplicates := CandidateDuplicateFieldNames()
	slices.Sort(expectedDuplicates)
	slices.Sort(gotDuplicates)

	if !reflect.DeepEqual(gotDuplicates, expectedDuplicates) {
		t.Fatalf("unexpected duplicate fields: got %#v want %#v", gotDuplicates, expectedDuplicates)
	}
}

func TestAvaliadorFieldMetadataInvariants(t *testing.T) {
	fields := AvaliadorFields()
	assertFieldMetadataInvariants(t, reflect.TypeOf(Avaliador{}), fields)

	expectedDuplicates := duplicateJSONFieldsFromTags(t, reflect.TypeOf(Avaliador{}))
	gotDuplicates := AvaliadorDuplicateFieldNames()
	slices.Sort(expectedDuplicates)
	slices.Sort(gotDuplicates)

	if !reflect.DeepEqual(gotDuplicates, expectedDuplicates) {
		t.Fatalf("unexpected duplicate fields: got %#v want %#v", gotDuplicates, expectedDuplicates)
	}
}

func assertFieldMetadataInvariants(t *testing.T, modelType reflect.Type, fields []FieldSchema) {
	t.Helper()

	if len(fields) == 0 {
		t.Fatal("expected at least one field schema")
	}

	seenJSONNames := make(map[string]bool, len(fields))
	for _, field := range fields {
		if field.JSONName == "" {
			t.Fatalf("field %s is missing JSONName", field.GoName)
		}
		if seenJSONNames[field.JSONName] {
			t.Fatalf("duplicate JSONName in metadata: %s", field.JSONName)
		}
		seenJSONNames[field.JSONName] = true

		structField, ok := modelType.FieldByName(field.GoName)
		if !ok {
			t.Fatalf("field %s not found in %s", field.GoName, modelType.Name())
		}

		if field.Kind != structField.Type.Kind() {
			t.Fatalf("unexpected kind for %s: got %s want %s", field.GoName, field.Kind, structField.Type.Kind())
		}

		if field.Persist {
			if field.ColumnName == "" {
				t.Fatalf("persisted field %s must have ColumnName", field.GoName)
			}
			if field.SQLiteType == "" {
				t.Fatalf("persisted field %s must have SQLiteType", field.GoName)
			}
			continue
		}

		if structField.Tag.Get("db") == "-" {
			if field.ColumnName != "" {
				t.Fatalf("non-persisted field %s should not expose ColumnName", field.GoName)
			}
			if field.SQLiteType != "" {
				t.Fatalf("non-persisted field %s should not expose SQLiteType", field.GoName)
			}
		}
	}

	jsonFields := JSONFieldNames(reflect.New(modelType).Elem().Interface())
	slices.Sort(jsonFields)
	gotJSONNames := make([]string, 0, len(fields))
	for _, field := range fields {
		gotJSONNames = append(gotJSONNames, field.JSONName)
	}
	slices.Sort(gotJSONNames)

	if !reflect.DeepEqual(gotJSONNames, jsonFields) {
		t.Fatalf("metadata JSON names diverged from struct tags: got %#v want %#v", gotJSONNames, jsonFields)
	}
}

func duplicateJSONFieldsFromTags(t *testing.T, modelType reflect.Type) []string {
	t.Helper()

	var duplicates []string
	for i := 0; i < modelType.NumField(); i++ {
		field := modelType.Field(i)
		if jsonTagName(field) == "" {
			continue
		}
		if field.Tag.Get("app") == "duplicate" {
			duplicates = append(duplicates, jsonTagName(field))
		}
	}
	return duplicates
}
