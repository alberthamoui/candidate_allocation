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

func TestWailsModelsContainMappingItemFields(t *testing.T) {
	assertWailsModelContainsJSONFields(t, "MappingItem", []string{
		"nomeColuna",
		"indice",
		"variavel",
		"includeWhenUnmapped",
	})
}

func TestWailsModelsContainUniqueValueDetectionFields(t *testing.T) {
	assertWailsModelContainsJSONFields(t, "UniqueValueDetection", []string{
		"valorOriginal",
		"valorNormalizado",
		"ocorrencias",
	})
}

func TestWailsModelsContainSoftCriterionFields(t *testing.T) {
	assertWailsModelContainsJSONFields(t, "SoftCriterion", []string{
		"type",
		"columnKey",
		"selectedValues",
		"threshold",
	})
}

func TestWailsModelsContainCandidateCriterionColumnFields(t *testing.T) {
	assertWailsModelContainsJSONFields(t, "CandidateCriterionColumn", []string{
		"key",
		"label",
		"isExtra",
	})
}

func TestWailsModelsContainAllocationParamsFields(t *testing.T) {
	modelsPath := filepath.Join("frontend", "wailsjs", "go", "models.ts")
	content, err := os.ReadFile(modelsPath)
	if err != nil {
		t.Fatalf("failed to read %s: %v", modelsPath, err)
	}

	classBody := extractWailsModelClassBody(t, string(content), "AllocationParams")
	requiredProperties := []string{
		"gruposPorHorario",
		"minPessoasPorGrupo",
		"maxPessoasPorGrupo",
		"avaliadoresPorGrupo",
		"softCriteria",
	}
	for _, field := range requiredProperties {
		propertyPattern := regexp.MustCompile(`(?m)^\s*` + regexp.QuoteMeta(field) + `:\s`)
		if !propertyPattern.MatchString(classBody) {
			t.Fatalf("field %q not found as property in Wails model AllocationParams", field)
		}
	}

	requiredAssignments := []string{
		`this.gruposPorHorario = source["gruposPorHorario"]`,
		`this.minPessoasPorGrupo = source["minPessoasPorGrupo"]`,
		`this.maxPessoasPorGrupo = source["maxPessoasPorGrupo"]`,
		`this.avaliadoresPorGrupo = source["avaliadoresPorGrupo"]`,
		`this.softCriteria = this.convertValues(source["softCriteria"], SoftCriterion)`,
	}
	for _, assignment := range requiredAssignments {
		if !strings.Contains(classBody, assignment) {
			t.Fatalf("assignment %q not found in Wails model AllocationParams", assignment)
		}
	}
}

func TestWailsModelsContainAllocationConfigurationFields(t *testing.T) {
	assertWailsModelContainsJSONFields(t, "HumanSummary", []string{
		"detectedPreferences",
		"mappedPreferences",
		"allocationParameters",
		"softCriteria",
		"normalizedValues",
		"validationObservations",
	})
	assertWailsModelContainsJSONFields(t, "ValidationMessage", []string{
		"level",
		"code",
		"message",
	})

	modelsPath := filepath.Join("frontend", "wailsjs", "go", "models.ts")
	content, err := os.ReadFile(modelsPath)
	if err != nil {
		t.Fatalf("failed to read %s: %v", modelsPath, err)
	}

	assertWailsModelContainsAssignments(t, string(content), "NormalizedAllocationInput", []string{
		`this.preferenceMappings = this.convertValues(source["preferenceMappings"], PreferenceScheduleMapping)`,
		`this.params = this.convertValues(source["params"], AllocationParams)`,
	})
	assertWailsModelContainsAssignments(t, string(content), "PreferenceMappingDiagnostic", []string{
		`this.detectedValue = this.convertValues(source["detectedValue"], UniqueValueDetection)`,
		`this.originalMapping = this.convertValues(source["originalMapping"], PreferenceScheduleMapping)`,
		`this.normalizedMapping = this.convertValues(source["normalizedMapping"], PreferenceScheduleMapping)`,
	})
	assertWailsModelContainsAssignments(t, string(content), "SoftCriterionDiagnostic", []string{
		`this.originalCriterion = this.convertValues(source["originalCriterion"], SoftCriterion)`,
		`this.normalizedCriterion = this.convertValues(source["normalizedCriterion"], SoftCriterion)`,
		`this.summary = source["summary"]`,
	})
	assertWailsModelContainsAssignments(t, string(content), "AllocationDiagnostics", []string{
		`this.detectedPreferences = this.convertValues(source["detectedPreferences"], UniqueValueDetection)`,
		`this.originalMappings = this.convertValues(source["originalMappings"], PreferenceScheduleMapping)`,
		`this.normalizedMappings = this.convertValues(source["normalizedMappings"], PreferenceScheduleMapping)`,
		`this.originalParams = this.convertValues(source["originalParams"], AllocationParams)`,
		`this.normalizedParams = this.convertValues(source["normalizedParams"], AllocationParams)`,
		`this.preferenceMappings = this.convertValues(source["preferenceMappings"], PreferenceMappingDiagnostic)`,
		`this.softCriteria = this.convertValues(source["softCriteria"], SoftCriterionDiagnostic)`,
		`this.validationMessages = this.convertValues(source["validationMessages"], ValidationMessage)`,
		`this.hasErrors = source["hasErrors"]`,
	})
	assertWailsModelContainsAssignments(t, string(content), "AllocationExecutionResult", []string{
		`this.status = source["status"]`,
		`this.allocation = this.convertValues(source["allocation"], Allocation)`,
		`this.notes = source["notes"]`,
	})
	assertWailsModelContainsAssignments(t, string(content), "AllocationConfiguration", []string{
		`this.summary = this.convertValues(source["summary"], HumanSummary)`,
		`this.normalized = this.convertValues(source["normalized"], NormalizedAllocationInput)`,
		`this.diagnostics = this.convertValues(source["diagnostics"], AllocationDiagnostics)`,
		`this.result = this.convertValues(source["result"], AllocationExecutionResult)`,
	})
}

func TestWailsAppBindingsContainNewAllocationHelpers(t *testing.T) {
	bindingsPath := filepath.Join("frontend", "wailsjs", "go", "main", "App.d.ts")
	content, err := os.ReadFile(bindingsPath)
	if err != nil {
		t.Fatalf("failed to read %s: %v", bindingsPath, err)
	}

	source := string(content)
	requiredSnippets := []string{
		"export function BuildAllocationConfiguration",
		"export function CountPossibleAllocationQuantities",
		"export function DetectUniquePreferenceValues",
		"export function ListCandidateCriterionColumns",
		"export function DetectUniqueCandidateColumnValues",
		"export function NormalizePreferenceScheduleMappings",
		"export function NormalizeSoftCriteria",
		"export function ValidateSoftCriteria",
		"export function ValidateAllocationParams",
	}
	for _, snippet := range requiredSnippets {
		if !strings.Contains(source, snippet) {
			t.Fatalf("expected App.d.ts to contain %q", snippet)
		}
	}
}

func TestWailsModelsDoNotGenerateInvalidStringClass(t *testing.T) {
	modelsPath := filepath.Join("frontend", "wailsjs", "go", "models.ts")
	content, err := os.ReadFile(modelsPath)
	if err != nil {
		t.Fatalf("failed to read %s: %v", modelsPath, err)
	}

	source := string(content)
	if strings.Contains(source, "export class string") {
		t.Fatalf("unexpected invalid string class generated in %s", modelsPath)
	}
	if !strings.Contains(source, "export class NullableString") {
		t.Fatalf("expected NullableString helper class to exist in %s", modelsPath)
	}
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

func assertWailsModelContainsAssignments(t *testing.T, source, className string, assignments []string) {
	t.Helper()

	classBody := extractWailsModelClassBody(t, source, className)
	for _, assignment := range assignments {
		if !strings.Contains(classBody, assignment) {
			t.Fatalf("assignment %q not found in Wails model %s", assignment, className)
		}
	}
}
