package types

type NullableString string

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

type Avaliador struct {
	ID     int                        `json:"id" db:"-"`
	Nome   string                     `json:"nome" db:"type=TEXT;required;unique" app:"duplicate"`
	Email  string                     `json:"email" db:"type=TEXT;required;unique" app:"duplicate"`
	Sigla  string                     `json:"sigla" db:"type=TEXT;required;unique" app:"duplicate"`
	Extras map[string]*NullableString `json:"extras" db:"type=TEXT"`
}

type Restricao struct {
	Candidato  string `json:"candidato"`
	NaoPosso   string `json:"naoPosso"`
	PrefiroNao string `json:"prefiroNao"`
}

type UniqueValueDetection struct {
	ValorOriginal    string `json:"valorOriginal"`
	ValorNormalizado string `json:"valorNormalizado"`
	Ocorrencias      int    `json:"ocorrencias"`
}

type PreferenceValueDetection = UniqueValueDetection

type PreferenceScheduleMapping struct {
	ValorPreferencia string `json:"valorPreferencia"`
	Dia              string `json:"dia"`
	Hora             string `json:"hora"`
}

type SoftCriterionType string

const (
	SoftCriterionMinValue             SoftCriterionType = "min_value"
	SoftCriterionAtLeastOneEach       SoftCriterionType = "at_least_one_each"
	SoftCriterionBalancedDistribution SoftCriterionType = "balanced_distribution"
	SoftCriterionGroupTogether        SoftCriterionType = "group_together"
	SoftCriterionMaxValue             SoftCriterionType = "max_value"
)

type CandidateCriterionColumn struct {
	Key     string `json:"key"`
	Label   string `json:"label"`
	IsExtra bool   `json:"isExtra"`
}

type SoftCriterion struct {
	Type           SoftCriterionType `json:"type"`
	ColumnKey      string            `json:"columnKey"`
	SelectedValues []string          `json:"selectedValues"`
	Threshold      int               `json:"threshold"`
}

type AllocationParams struct {
	GruposPorHorario    int             `json:"gruposPorHorario"`
	MinPessoasPorGrupo  int             `json:"minPessoasPorGrupo"`
	MaxPessoasPorGrupo  int             `json:"maxPessoasPorGrupo"`
	AvaliadoresPorGrupo int             `json:"avaliadoresPorGrupo"`
	SoftCriteria        []SoftCriterion `json:"softCriteria"`
}

type ValidationMessageLevel string

const (
	ValidationMessageLevelError   ValidationMessageLevel = "error"
	ValidationMessageLevelWarning ValidationMessageLevel = "warning"
)

type ValidationMessage struct {
	Level   ValidationMessageLevel `json:"level"`
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
}

type HumanSummary struct {
	DetectedPreferences    []string `json:"detectedPreferences"`
	MappedPreferences      []string `json:"mappedPreferences"`
	AllocationParameters   []string `json:"allocationParameters"`
	SoftCriteria           []string `json:"softCriteria"`
	NormalizedValues       []string `json:"normalizedValues"`
	ValidationObservations []string `json:"validationObservations"`
}

type NormalizedAllocationInput struct {
	PreferenceMappings []PreferenceScheduleMapping `json:"preferenceMappings"`
	Params             AllocationParams            `json:"params"`
}

type PreferenceMappingDiagnostic struct {
	DetectedValue     UniqueValueDetection      `json:"detectedValue"`
	OriginalMapping   PreferenceScheduleMapping `json:"originalMapping"`
	NormalizedMapping PreferenceScheduleMapping `json:"normalizedMapping"`
}

type SoftCriterionDiagnostic struct {
	OriginalCriterion   SoftCriterion `json:"originalCriterion"`
	NormalizedCriterion SoftCriterion `json:"normalizedCriterion"`
	Summary             string        `json:"summary"`
}

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

type AllocationExecutionResult struct {
	Status     string      `json:"status"`
	Allocation *Allocation `json:"allocation,omitempty"`
	Notes      []string    `json:"notes"`
}

type AllocationConfiguration struct {
	Summary     HumanSummary              `json:"summary"`
	Normalized  NormalizedAllocationInput `json:"normalized"`
	Diagnostics AllocationDiagnostics     `json:"diagnostics"`
	Result      AllocationExecutionResult `json:"result"`
}

type AllocationProblem struct {
	Candidates       []SolverCandidate      `json:"candidates"`
	Groups           []SolverGroup          `json:"groups"`
	HardRestrictions SolverHardRestrictions `json:"hardRestrictions"`
	SoftRules        SolverSoftRules        `json:"softRules"`
}

type SolverCandidate struct {
	ID                    int                         `json:"id"`
	Name                  string                      `json:"name"`
	PreferredGroupIDs     []int                       `json:"preferredGroupIds"`
	Attributes            map[string]string           `json:"attributes"`
	EvaluatorRestrictions SolverCandidateRestrictions `json:"evaluatorRestrictions"`
}

type SolverCandidateRestrictions struct {
	ForbiddenEvaluatorIDs []int `json:"forbiddenEvaluatorIds"`
	AvoidEvaluatorIDs     []int `json:"avoidEvaluatorIds"`
}

type SolverGroup struct {
	ID            int    `json:"id"`
	Label         string `json:"label"`
	EvaluatorIDs  []int  `json:"evaluatorIds"`
	MinCandidates int    `json:"minCandidates"`
	MaxCandidates int    `json:"maxCandidates"`
}

type SolverHardRestrictions struct {
	AllCandidatesMustBeAssigned         bool `json:"allCandidatesMustBeAssigned"`
	RespectCandidatePreferences         bool `json:"respectCandidatePreferences"`
	EnforceGroupCapacity                bool `json:"enforceGroupCapacity"`
	EnforceForbiddenEvaluators          bool `json:"enforceForbiddenEvaluators"`
	EnforceMinCandidatesOnCompleteState bool `json:"enforceMinCandidatesOnCompleteState"`
}

type SolverSoftRules struct {
	PreferencePenaltyByRank []int           `json:"preferencePenaltyByRank"`
	AvoidEvaluatorPenalty   int             `json:"avoidEvaluatorPenalty"`
	Criteria                []SoftCriterion `json:"criteria"`
}

type PartialAllocationState struct {
	Assignments  map[int]int   `json:"assignments"`
	GroupMembers map[int][]int `json:"groupMembers"`
}

type HardConstraintViolation struct {
	Code        string `json:"code"`
	Message     string `json:"message"`
	CandidateID int    `json:"candidateId"`
	GroupID     int    `json:"groupId"`
	EvaluatorID int    `json:"evaluatorId"`
}

type SoftScoreComponent struct {
	Code    string `json:"code"`
	Penalty int    `json:"penalty"`
	Message string `json:"message"`
}

type SoftScoreBreakdown struct {
	TotalPenalty int                  `json:"totalPenalty"`
	Components   []SoftScoreComponent `json:"components"`
}

type SolverResult struct {
	Status          string                    `json:"status"`
	Assignments     map[int]int               `json:"assignments"`
	Score           SoftScoreBreakdown        `json:"score"`
	HardViolations  []HardConstraintViolation `json:"hardViolations"`
	RejectionReason string                    `json:"rejectionReason"`
}

type MappingItem struct {
	NomeColuna          string `json:"nomeColuna"`
	Indice              int    `json:"indice"`
	Variavel            string `json:"variavel"`
	IncludeWhenUnmapped bool   `json:"includeWhenUnmapped"`
}

type MappingFieldInfo struct {
	Variavel  string `json:"variavel"`
	Required  bool   `json:"required"`
	Unique    bool   `json:"unique"`
	Duplicate bool   `json:"duplicate"`
}

type NaoAlocados struct {
	Candidatos  []Candidato `json:"candidatos"`
	Avaliadores []Avaliador `json:"avaliadores"`
}

type Allocation struct {
	Horarios    []Horario   `json:"horarios"`
	NaoAlocados NaoAlocados `json:"naoAlocados"`
}

type Mesa struct {
	ID          int    // único (ex.: 301 = quarta-mesa1)
	DiaID       int    // 1=segunda, 2=terça, ...
	Descricao   string // "quarta – mesa 2"
	Candidatos  []int  // já alocados
	Avaliadores []int
}

type ResultadoAlocacao struct {
	Alocacao  map[int]int
	Pontuacao int
	Alocados  int
}

type Horario struct {
	ID          int
	Descricao   string
	Candidatos  []int
	Avaliadores []int
}

type HorarioInfo struct {
	H       *Horario
	Pessoas []int
}
