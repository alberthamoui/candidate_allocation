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

type AllocationSetup struct {
	DetectedPreferences []UniqueValueDetection      `json:"detectedPreferences"`
	PreferenceMappings  []PreferenceScheduleMapping `json:"preferenceMappings"`
	Params              AllocationParams            `json:"params"`
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
