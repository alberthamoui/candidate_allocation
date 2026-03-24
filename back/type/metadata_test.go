package types

import (
	"reflect"
	"slices"
	"strconv"
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

func TestCandidateMappingFieldInfosFollowCurrentSchema(t *testing.T) {
	infos := CandidateMappingFieldInfos(3)
	got := make(map[string]MappingFieldInfo, len(infos))
	for _, info := range infos {
		got[info.Variavel] = info
	}

	for _, field := range CandidateFields() {
		switch field.JSONName {
		case "extras", "opcoes":
			continue
		}

		info, ok := got[field.JSONName]
		if !ok {
			t.Fatalf("expected mapping info for %q", field.JSONName)
		}
		if info.Required != field.Required {
			t.Fatalf("unexpected required flag for %q: got %v want %v", field.JSONName, info.Required, field.Required)
		}
		if info.Unique != field.Unique {
			t.Fatalf("unexpected unique flag for %q: got %v want %v", field.JSONName, info.Unique, field.Unique)
		}
		if info.Duplicate != field.Duplicate {
			t.Fatalf("unexpected duplicate flag for %q: got %v want %v", field.JSONName, info.Duplicate, field.Duplicate)
		}
	}

	for i := 1; i <= 3; i++ {
		key := "opcao " + strconv.Itoa(i)
		info, ok := got[key]
		if !ok {
			t.Fatalf("expected mapping info for %q", key)
		}
		if info.Required || info.Unique || info.Duplicate {
			t.Fatalf("expected option field %q to be optional and non-unique: %#v", key, info)
		}
	}
}

func TestAvaliadorMappingFieldInfosFollowCurrentSchema(t *testing.T) {
	infos := AvaliadorMappingFieldInfos()
	got := make(map[string]MappingFieldInfo, len(infos))
	for _, info := range infos {
		got[info.Variavel] = info
	}

	for _, field := range AvaliadorFields() {
		if field.JSONName == "extras" {
			continue
		}

		info, ok := got[field.JSONName]
		if !ok {
			t.Fatalf("expected mapping info for %q", field.JSONName)
		}
		if info.Required != field.Required || info.Unique != field.Unique || info.Duplicate != field.Duplicate {
			t.Fatalf("unexpected mapping info for %q: got %#v want required=%v unique=%v duplicate=%v", field.JSONName, info, field.Required, field.Unique, field.Duplicate)
		}
	}
}

func TestRestricaoMappingFieldInfosFollowCurrentSchema(t *testing.T) {
	infos := RestricaoMappingFieldInfos()
	if len(infos) != len(RestricaoFieldNames()) {
		t.Fatalf("unexpected restriction mapping info count: got %d want %d", len(infos), len(RestricaoFieldNames()))
	}

	for i, fieldName := range RestricaoFieldNames() {
		if infos[i].Variavel != fieldName {
			t.Fatalf("unexpected restriction field at index %d: got %q want %q", i, infos[i].Variavel, fieldName)
		}
		if infos[i].Required || infos[i].Unique || infos[i].Duplicate {
			t.Fatalf("expected restriction field %q to be optional metadata only, got %#v", fieldName, infos[i])
		}
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
