package logic

import types "candidate_alocator/back/type"

// WorkflowDefinition is the single workflow contract shared by CLI and Wails.
func WorkflowDefinition() types.WorkflowDefinition {
	return types.WorkflowDefinition{
		Steps: []types.WorkflowStep{
			{Key: "import", Label: "Importacao", Required: true},
			{Key: "candidate_mapping", Label: "Mapeamento de candidatos", Required: true},
			{Key: "evaluator_mapping", Label: "Mapeamento de avaliadores", Required: true},
			{Key: "restriction_mapping", Label: "Mapeamento de restricoes", Required: true},
			{Key: "verification", Label: "Verificacao", Required: true},
			{Key: "allocation_config", Label: "Configuracao de alocacao", Required: true},
			{Key: "allocation_run", Label: "Execucao", Required: true},
		},
		DefaultAllocationParams: DefaultAllocationParams(),
		SoftCriterionOptions:    SoftCriterionOptions(),
		BaseOptimization:        BaseOptimizationPolicy(),
	}
}

// BaseOptimizationPolicy returns the official always-on objective defaults.
func BaseOptimizationPolicy() types.BaseOptimizationPolicy {
	return types.BaseOptimizationPolicy{
		PreferencePenaltyByRank: []int{0, 1, 2, 3, 4},
		AvoidEvaluatorPenalty:   3,
	}
}

// SoftCriterionOptions returns the official selectable soft criteria.
func SoftCriterionOptions() []types.SoftCriterionOption {
	return []types.SoftCriterionOption{
		{
			Type:              types.SoftCriterionMinValue,
			Label:             "Mínimo por valor único",
			Description:       "Se existe o valor selecionado no grupo, garanta pelo menos N",
			RequiresThreshold: true,
		},
		{
			Type:              types.SoftCriterionAtLeastOneEach,
			Label:             "Pelo menos 1 de cada valor",
			Description:       "Garanta representacao para os valores selecionados",
			RequiresThreshold: false,
		},
		{
			Type:              types.SoftCriterionBalancedDistribution,
			Label:             "Distribuição equilibrada",
			Description:       "Espalhe os valores selecionados entre os grupos",
			RequiresThreshold: false,
		},
		{
			Type:              types.SoftCriterionGroupTogether,
			Label:             "Agrupamento",
			Description:       "Prefira manter os valores selecionados juntos",
			RequiresThreshold: false,
		},
		{
			Type:              types.SoftCriterionMaxValue,
			Label:             "Máximo por valor único",
			Description:       "Limite o numero de pessoas do valor selecionado por grupo",
			RequiresThreshold: true,
		},
	}
}
