package main

// Usuario representa um candidato extraído do Excel.
type Usuario struct {
	Timestamp    string   `json:"timestamp"`
	Nome         string   `json:"nome"`
	CPF          string   `json:"cpf"`
	Numero       string   `json:"numero"`
	Semestre     string   `json:"semestre"`
	Curso        string   `json:"curso"`
	EmailInsper  string   `json:"email_insper"`
	EmailPessoal string   `json:"email_pessoal"`
	Opcoes       []string `json:"opcoes"`
}

// Candidato é um alias de Usuario — mantém compatibilidade com processa.go e testes.
type Candidato = Usuario

// AvaliadorInfo representa um avaliador extraído do Excel.
type AvaliadorInfo struct {
	Nome  string `json:"nome"`
	Email string `json:"email"`
	Sigla string `json:"sigla"`
}

// Restricao representa uma restrição de avaliação extraída do Excel.
type Restricao struct {
	Candidato  string `json:"candidato"`
	NaoPosso   string `json:"naoPosso"`
	PrefiroNao string `json:"prefiroNao"`
}

// MappingItem descreve o mapeamento de uma coluna do Excel para uma variável interna.
type MappingItem struct {
	NomeColuna string `json:"nomeColuna"`
	Indice     int    `json:"indice"`
	Variavel   string `json:"variavel"`
}

// UsuariosResponse é a resposta de BuildUsuariosWithMapping.
type UsuariosResponse struct {
	Usuarios   map[int]ValidationResult `json:"usuarios"`
	Duplicates [][]int                  `json:"duplicates"`
}

// MesaResult é a forma serializável de uma Mesa com nomes legíveis.
type MesaResult struct {
	ID          int      `json:"id"`
	DiaID       int      `json:"dia_id"`
	DiaNome     string   `json:"dia_nome"`
	Descricao   string   `json:"descricao"`
	Candidatos  []string `json:"candidatos"`
	Avaliadores []string `json:"avaliadores"`
}

// PessoaInfo carrega os campos exibidos para candidatos não alocados.
type PessoaInfo struct {
	ID          int    `json:"id"`
	Nome        string `json:"nome"`
	EmailInsper string `json:"email_insper"`
	Curso       string `json:"curso"`
	Semestre    int    `json:"semestre"`
}

// AlocacaoResponse é o que RunAlocacao retorna ao frontend.
type AlocacaoResponse struct {
	Mesas           []MesaResult `json:"mesas"`
	TotalAlocados   int          `json:"total_alocados"`
	NaoAlocadosInfo []PessoaInfo `json:"nao_alocados_info"`
	Pontuacao       int          `json:"pontuacao"`
}
