package types

type Candidato struct {
	Timestamp       string            `json:"timestamp" db:"type=TEXT"`
	Nome            string            `json:"nome" db:"type=TEXT;required"`
	CPF             string            `json:"cpf" db:"type=TEXT;required;unique" app:"duplicate"`
	Numero          string            `json:"numero" db:"type=TEXT;required"`
	Semestre        string            `json:"semestre" db:"type=INTEGER;required"`
	Curso           string            `json:"curso" db:"type=TEXT;required"`
	EmailSecundario string            `json:"email_secundario" db:"type=TEXT;required" app:"duplicate"`
	EmailPessoal    string            `json:"email_pessoal" db:"type=TEXT;required" app:"duplicate"`
	Opcoes          []string          `json:"opcoes" db:"-"`
	Extras          map[string]string `json:"extras" db:"type=TEXT"`
}

type AvaliadorInfo struct {
	Nome   string            `json:"nome" db:"type=TEXT;required;unique"`
	Email  string            `json:"email" db:"type=TEXT;required;unique"`
	Sigla  string            `json:"sigla" db:"type=TEXT;required;unique"`
	Extras map[string]string `json:"extras" db:"type=TEXT"`
}

type Restricao struct {
	Candidato  string `json:"candidato"`
	NaoPosso   string `json:"naoPosso"`
	PrefiroNao string `json:"prefiroNao"`
}

type MappingItem struct {
	NomeColuna string `json:"nomeColuna"`
	Indice     int    `json:"indice"`
	Variavel   string `json:"variavel"`
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

type Avaliador struct {
	ID    int    `json:"id"`
	Nome  string `json:"nome"`
	Email string `json:"email"`
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
