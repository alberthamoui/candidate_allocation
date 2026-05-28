package types

// NullableString represents an optional string stored in JSON fields.
type NullableString string

// Candidato holds the imported candidate data.
type Candidato struct {
	Timestamp       string                     `json:"timestamp" db:"type=TEXT"`
	Nome            string                     `json:"nome" db:"type=TEXT;required"`
	CPF             string                     `json:"cpf" db:"type=TEXT;required;unique" app:"duplicate"`
	Numero          string                     `json:"numero" db:"type=TEXT;required"`
	Semestre        string                     `json:"semestre" db:"type=INTEGER;required"`
	Curso           string                     `json:"curso" db:"type=TEXT;required"`
	EmailSecundario string                     `json:"email_secundario" db:"type=TEXT;required" app:"duplicate"`
	EmailPessoal    string                     `json:"email_pessoal" db:"type=TEXT;required" app:"duplicate"`
	Opcoes          []string                   `json:"opcoes" db:"-"`
	Extras          map[string]*NullableString `json:"extras" db:"type=TEXT"`
}

// Avaliador holds the imported evaluator data.
type Avaliador struct {
	ID     int                        `json:"id" db:"-"`
	Nome   string                     `json:"nome" db:"type=TEXT;required;unique" app:"duplicate"`
	Email  string                     `json:"email" db:"type=TEXT;required;unique" app:"duplicate"`
	Sigla  string                     `json:"sigla" db:"type=TEXT;required;unique" app:"duplicate"`
	Extras map[string]*NullableString `json:"extras" db:"type=TEXT"`
}

// Restricao holds evaluator restriction rows imported from the spreadsheet.
type Restricao struct {
	Candidato  string `json:"candidato"`
	NaoPosso   string `json:"naoPosso"`
	PrefiroNao string `json:"prefiroNao"`
}

// UniqueValueDetection describes one detected unique value in a column.
type UniqueValueDetection struct {
	ValorOriginal    string `json:"valorOriginal"`
	ValorNormalizado string `json:"valorNormalizado"`
	Ocorrencias      int    `json:"ocorrencias"`
}

// PreferenceValueDetection is kept as an alias for preference detections.
type PreferenceValueDetection = UniqueValueDetection

// PreferenceScheduleMapping maps a detected preference to a real schedule.
type PreferenceScheduleMapping struct {
	ValorPreferencia string `json:"valorPreferencia"`
	Dia              string `json:"dia"`
	Hora             string `json:"hora"`
}

// SoftCriterionType identifies the available soft allocation rules.
type SoftCriterionType string

// Supported soft criterion kinds.
const (
	SoftCriterionMinValue             SoftCriterionType = "min_value"
	SoftCriterionAtLeastOneEach       SoftCriterionType = "at_least_one_each"
	SoftCriterionBalancedDistribution SoftCriterionType = "balanced_distribution"
	SoftCriterionGroupTogether        SoftCriterionType = "group_together"
	SoftCriterionMaxValue             SoftCriterionType = "max_value"
)

// CandidateCriterionColumn describes a candidate column available for criteria.
type CandidateCriterionColumn struct {
	Key     string `json:"key"`
	Label   string `json:"label"`
	IsExtra bool   `json:"isExtra"`
}

// SoftCriterion describes one configured soft allocation rule.
type SoftCriterion struct {
	Type           SoftCriterionType `json:"type"`
	ColumnKey      string            `json:"columnKey"`
	SelectedValues []string          `json:"selectedValues"`
	Threshold      int               `json:"threshold"`
}

// WorkflowStep describes one official workflow step shared by CLI and Wails.
type WorkflowStep struct {
	Key      string `json:"key"`
	Label    string `json:"label"`
	Required bool   `json:"required"`
}

// SoftCriterionOption describes one selectable soft criterion in the workflow.
type SoftCriterionOption struct {
	Type              SoftCriterionType `json:"type"`
	Label             string            `json:"label"`
	Description       string            `json:"description"`
	RequiresThreshold bool              `json:"requiresThreshold"`
}

// WorkflowDefinition is the single backend contract consumed by CLI and Wails.
type WorkflowDefinition struct {
	Steps                   []WorkflowStep        `json:"steps"`
	DefaultAllocationParams AllocationParams      `json:"defaultAllocationParams"`
	SoftCriterionOptions    []SoftCriterionOption `json:"softCriterionOptions"`
}

// AllocationParams contains the editable allocation parameters.
type AllocationParams struct {
	GruposPorHorario    int             `json:"gruposPorHorario"`
	MinPessoasPorGrupo  int             `json:"minPessoasPorGrupo"`
	MaxPessoasPorGrupo  int             `json:"maxPessoasPorGrupo"`
	AvaliadoresPorGrupo int             `json:"avaliadoresPorGrupo"`
	SoftCriteria        []SoftCriterion `json:"softCriteria"`
}

// ValidationMessageLevel classifies validation messages.
type ValidationMessageLevel string

// Supported validation message levels.
const (
	ValidationMessageLevelError   ValidationMessageLevel = "error"
	ValidationMessageLevelWarning ValidationMessageLevel = "warning"
)

// ValidationMessage is a structured validation note.
type ValidationMessage struct {
	Level   ValidationMessageLevel `json:"level"`
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
}

// HumanSummary aggregates a human-readable allocation summary.
type HumanSummary struct {
	DetectedPreferences    []string `json:"detectedPreferences"`
	MappedPreferences      []string `json:"mappedPreferences"`
	AllocationParameters   []string `json:"allocationParameters"`
	SoftCriteria           []string `json:"softCriteria"`
	NormalizedValues       []string `json:"normalizedValues"`
	ValidationObservations []string `json:"validationObservations"`
}

// NormalizedAllocationInput is the normalized allocation input for the solver.
type NormalizedAllocationInput struct {
	PreferenceMappings []PreferenceScheduleMapping `json:"preferenceMappings"`
	Params             AllocationParams            `json:"params"`
}

// PreferenceMappingDiagnostic explains one normalized preference mapping.
type PreferenceMappingDiagnostic struct {
	DetectedValue     UniqueValueDetection      `json:"detectedValue"`
	OriginalMapping   PreferenceScheduleMapping `json:"originalMapping"`
	NormalizedMapping PreferenceScheduleMapping `json:"normalizedMapping"`
}

// SoftCriterionDiagnostic explains one normalized soft criterion.
type SoftCriterionDiagnostic struct {
	OriginalCriterion   SoftCriterion `json:"originalCriterion"`
	NormalizedCriterion SoftCriterion `json:"normalizedCriterion"`
	Summary             string        `json:"summary"`
}

// AllocationDiagnostics contains validation and normalization details.
type AllocationDiagnostics struct {
	DetectedPreferences []UniqueValueDetection        `json:"detectedPreferences"`
	OriginalMappings    []PreferenceScheduleMapping   `json:"originalMappings"`
	NormalizedMappings  []PreferenceScheduleMapping   `json:"normalizedMappings"`
	OriginalParams      AllocationParams              `json:"originalParams"`
	NormalizedParams    AllocationParams              `json:"normalizedParams"`
	PreferenceMappings  []PreferenceMappingDiagnostic `json:"preferenceMappings"`
	SoftCriteria        []SoftCriterionDiagnostic     `json:"softCriteria"`
	ValidationMessages  []ValidationMessage           `json:"validationMessages"`
	HasErrors           bool                          `json:"hasErrors"`
}

// AllocationExecutionResult is the final execution status for the workflow.
type AllocationExecutionResult struct {
	Status     string      `json:"status"`
	Allocation *Allocation `json:"allocation,omitempty"`
	Notes      []string    `json:"notes"`
}

// AllocationConfiguration bundles the allocation summary, input and result.
type AllocationConfiguration struct {
	Summary     HumanSummary              `json:"summary"`
	Normalized  NormalizedAllocationInput `json:"normalized"`
	Diagnostics AllocationDiagnostics     `json:"diagnostics"`
	Result      AllocationExecutionResult `json:"result"`
}

// AllocationProblem is the solver input model.
type AllocationProblem struct {
	Candidates       []SolverCandidate      `json:"candidates"`
	Groups           []SolverGroup          `json:"groups"`
	HardRestrictions SolverHardRestrictions `json:"hardRestrictions"`
	SoftRules        SolverSoftRules        `json:"softRules"`
}

// SolverCandidate is the solver-side candidate model.
type SolverCandidate struct {
	ID                    int                         `json:"id"`
	Name                  string                      `json:"name"`
	PreferredGroupIDs     []int                       `json:"preferredGroupIds"`
	Attributes            map[string]string           `json:"attributes"`
	EvaluatorRestrictions SolverCandidateRestrictions `json:"evaluatorRestrictions"`
}

// SolverCandidateRestrictions stores evaluator restrictions for a candidate.
type SolverCandidateRestrictions struct {
	ForbiddenEvaluatorIDs []int `json:"forbiddenEvaluatorIds"`
	AvoidEvaluatorIDs     []int `json:"avoidEvaluatorIds"`
}

// SolverGroup is the solver-side group model.
type SolverGroup struct {
	ID            int    `json:"id"`
	Label         string `json:"label"`
	EvaluatorIDs  []int  `json:"evaluatorIds"`
	MinCandidates int    `json:"minCandidates"`
	MaxCandidates int    `json:"maxCandidates"`
}

// SolverHardRestrictions toggles hard solver rules.
type SolverHardRestrictions struct {
	AllCandidatesMustBeAssigned         bool `json:"allCandidatesMustBeAssigned"`
	RespectCandidatePreferences         bool `json:"respectCandidatePreferences"`
	EnforceGroupCapacity                bool `json:"enforceGroupCapacity"`
	EnforceForbiddenEvaluators          bool `json:"enforceForbiddenEvaluators"`
	EnforceMinCandidatesOnCompleteState bool `json:"enforceMinCandidatesOnCompleteState"`
}

// SolverSoftRules contains the soft scoring configuration.
type SolverSoftRules struct {
	PreferencePenaltyByRank []int           `json:"preferencePenaltyByRank"`
	AvoidEvaluatorPenalty   int             `json:"avoidEvaluatorPenalty"`
	Criteria                []SoftCriterion `json:"criteria"`
}

// PartialAllocationState represents a partial solver state.
type PartialAllocationState struct {
	Assignments  map[int]int   `json:"assignments"`
	GroupMembers map[int][]int `json:"groupMembers"`
}

// HardConstraintViolation describes a hard-rule failure.
type HardConstraintViolation struct {
	Code        string `json:"code"`
	Message     string `json:"message"`
	CandidateID int    `json:"candidateId"`
	GroupID     int    `json:"groupId"`
	EvaluatorID int    `json:"evaluatorId"`
}

// SoftScoreComponent describes one component of the soft score.
type SoftScoreComponent struct {
	Code    string `json:"code"`
	Penalty int    `json:"penalty"`
	Message string `json:"message"`
}

// SoftScoreBreakdown aggregates the soft score and its components.
type SoftScoreBreakdown struct {
	TotalPenalty int                  `json:"totalPenalty"`
	Components   []SoftScoreComponent `json:"components"`
}

// SolverMetrics reports solver exploration statistics.
type SolverMetrics struct {
	NodesVisited       int `json:"nodesVisited"`
	CompleteStates     int `json:"completeStates"`
	NodesPrunedByHard  int `json:"nodesPrunedByHard"`
	NodesPrunedByBound int `json:"nodesPrunedByBound"`
	BestUpdates        int `json:"bestUpdates"`
	ParallelTasks      int `json:"parallelTasks"`
}

// SolverResult is the final solver output.
type SolverResult struct {
	Status          string                    `json:"status"`
	Assignments     map[int]int               `json:"assignments"`
	Score           SoftScoreBreakdown        `json:"score"`
	HardViolations  []HardConstraintViolation `json:"hardViolations"`
	RejectionReason string                    `json:"rejectionReason"`
	Metrics         SolverMetrics             `json:"metrics"`
	DebugNotes      []string                  `json:"debugNotes"`
}

// MappingItem represents one spreadsheet-to-domain mapping entry.
type MappingItem struct {
	NomeColuna          string `json:"nomeColuna"`
	Indice              int    `json:"indice"`
	Variavel            string `json:"variavel"`
	IncludeWhenUnmapped bool   `json:"includeWhenUnmapped"`
}

// MappingFieldInfo describes one field available for mapping.
type MappingFieldInfo struct {
	Variavel  string `json:"variavel"`
	Required  bool   `json:"required"`
	Unique    bool   `json:"unique"`
	Duplicate bool   `json:"duplicate"`
}

// NaoAlocados stores unallocated candidates and evaluators.
type NaoAlocados struct {
	Candidatos  []Candidato `json:"candidatos"`
	Avaliadores []Avaliador `json:"avaliadores"`
}

// Allocation is the grouped allocation result.
type Allocation struct {
	Horarios    []Horario   `json:"horarios"`
	NaoAlocados NaoAlocados `json:"naoAlocados"`
}

// Mesa is a legacy table allocation unit.
type Mesa struct {
	ID          int    // único (ex.: 301 = quarta-mesa1)
	DiaID       int    // 1=segunda, 2=terça, ...
	Descricao   string // "quarta – mesa 2"
	Candidatos  []int  // já alocados
	Avaliadores []int
}

// ResultadoAlocacao is the legacy allocation result.
type ResultadoAlocacao struct {
	Alocacao  map[int]int
	Pontuacao int
	Alocados  int
}

// Horario is a legacy schedule entity.
type Horario struct {
	ID          int
	Descricao   string
	Candidatos  []int
	Avaliadores []int
}

// HorarioInfo groups a schedule and the people assigned to it.
type HorarioInfo struct {
	H       *Horario
	Pessoas []int
}
