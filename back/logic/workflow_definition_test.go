package logic

import (
	"testing"

	types "candidate_alocator/back/type"
)

func TestWorkflowDefinitionContainsOfficialDefaultsAndOptions(t *testing.T) {
	definition := WorkflowDefinition()

	if len(definition.Steps) == 0 {
		t.Fatal("expected workflow definition to expose steps")
	}
	if definition.DefaultAllocationParams.GruposPorHorario != DefaultAllocationParams().GruposPorHorario {
		t.Fatalf("workflow defaults diverged from DefaultAllocationParams: %#v", definition.DefaultAllocationParams)
	}
	if len(definition.SoftCriterionOptions) != len(SoftCriterionOptions()) {
		t.Fatalf("workflow soft options diverged from SoftCriterionOptions: %#v", definition.SoftCriterionOptions)
	}
}

func TestWorkflowDefinitionSoftCriteriaAreSupportedByValidation(t *testing.T) {
	candidatos := []types.Candidato{
		{Curso: "ADM"},
		{Curso: "ECO"},
	}

	for _, option := range WorkflowDefinition().SoftCriterionOptions {
		criterion := types.SoftCriterion{
			Type:           option.Type,
			ColumnKey:      "curso",
			SelectedValues: []string{"adm"},
		}
		if option.RequiresThreshold {
			criterion.Threshold = 1
		}
		if !option.RequiresThreshold {
			criterion.SelectedValues = []string{"adm", "eco"}
		}

		if err := ValidateSoftCriteria([]types.SoftCriterion{criterion}, candidatos); err != nil {
			t.Fatalf("soft criterion option %q is not accepted by validation: %v", option.Type, err)
		}
	}
}

func TestWorkflowDefinitionRequiresThresholdMatchesCriterionSemantics(t *testing.T) {
	thresholdTypes := map[types.SoftCriterionType]bool{
		types.SoftCriterionMinValue: true,
		types.SoftCriterionMaxValue: true,
	}

	for _, option := range SoftCriterionOptions() {
		if option.RequiresThreshold != thresholdTypes[option.Type] {
			t.Fatalf("unexpected threshold flag for %q: %v", option.Type, option.RequiresThreshold)
		}
	}
}
