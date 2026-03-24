package main

import (
	types "candidate_alocator/back/type"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestWailsModelsContainAllCandidateJSONFields(t *testing.T) {
	assertWailsModelContainsJSONFields(t, "Candidato", types.JSONFieldNames(types.Candidato{}))
}

func TestWailsModelsContainAllAvaliadorJSONFields(t *testing.T) {
	assertWailsModelContainsJSONFields(t, "Avaliador", types.JSONFieldNames(types.Avaliador{}))
}

func TestWailsModelsContainMappingFieldInfoFields(t *testing.T) {
	assertWailsModelContainsJSONFields(t, "MappingFieldInfo", []string{
		"variavel",
		"required",
		"unique",
		"duplicate",
	})
}

func assertWailsModelContainsJSONFields(t *testing.T, className string, fields []string) {
	t.Helper()

	modelsPath := filepath.Join("frontend", "wailsjs", "go", "models.ts")
	content, err := os.ReadFile(modelsPath)
	if err != nil {
		t.Fatalf("failed to read %s: %v", modelsPath, err)
	}

	classBody := extractWailsModelClassBody(t, string(content), className)
	for _, field := range fields {
		propertyPattern := regexp.MustCompile(`(?m)^\s*` + regexp.QuoteMeta(field) + `:\s`)
		if !propertyPattern.MatchString(classBody) {
			t.Fatalf("field %q not found as property in Wails model %s", field, className)
		}

		assignment := `this.` + field + ` = source["` + field + `"]`
		if !strings.Contains(classBody, assignment) {
			t.Fatalf("field %q not found in constructor assignment for Wails model %s", field, className)
		}
	}
}

func extractWailsModelClassBody(t *testing.T, source, className string) string {
	t.Helper()

	re := regexp.MustCompile(`(?s)export class ` + regexp.QuoteMeta(className) + ` \{(.*?)\n\t\}`)
	matches := re.FindStringSubmatch(source)
	if len(matches) != 2 {
		t.Fatalf("failed to locate Wails model class %s", className)
	}
	return matches[1]
}
