package main

import (
	"reflect"
	"testing"
)

// colunasSugeridas devolve, para cada campo, o nome da coluna sugerida.
func colunasSugeridas(cabecalho, campos []string) map[string]string {
	res := map[string]string{}
	for _, m := range sugerirMapeamento(cabecalho, campos) {
		res[m.Variavel] = m.NomeColuna
	}
	return res
}

func TestTokensNome(t *testing.T) {
	casos := map[string][]string{
		"Primeira Opção": {"1", "opcao"},
		"opcao 1":        {"opcao", "1"},
		"Opcao1":         {"opcao", "1"},
		"1ª opção":       {"1", "opcao"},
		"naoPosso":       {"nao", "posso"},
		"NÃO POSSO":      {"nao", "posso"},
		"email_insper":   {"email", "insper"},
		"E-mail Pessoal": {"e", "mail", "pessoal"},
		"TimeStamp":      {"time", "stamp"},
		"CPF":            {"cpf"},
	}
	for entrada, want := range casos {
		if got := tokensNome(entrada); !reflect.DeepEqual(got, want) {
			t.Errorf("tokensNome(%q) = %q, esperado %q", entrada, got, want)
		}
	}
}

func TestSugestaoCabecalhosReais(t *testing.T) {
	casos := []struct {
		nome      string
		cabecalho []string
		campos    []string
		want      map[string]string
	}{
		{
			nome:      "teste_oficial candidatos",
			cabecalho: []string{"Timestamp", "Nome", "CPF", "Numero", "Semestre", "Curso", "Email Institucional", "Email Pessoal", "Opcao 1", "Opcao 2", "Opcao 3", "Opcao 4", "Opcao 5"},
			campos:    getUsuarioFields(5),
			want: map[string]string{
				"timestamp": "Timestamp", "nome": "Nome", "cpf": "CPF", "numero": "Numero", "semestre": "Semestre", "curso": "Curso",
				"email_insper": "Email Institucional", "email_pessoal": "Email Pessoal",
				"opcao 1": "Opcao 1", "opcao 2": "Opcao 2", "opcao 3": "Opcao 3", "opcao 4": "Opcao 4", "opcao 5": "Opcao 5",
			},
		},
		{
			// 4 opções na planilha e 5 pedidas: "opcao 5" fica sem coluna
			nome:      "base_exemplo candidatos",
			cabecalho: []string{"TimeStamp", "Nome", "CPF", "Numero", "Semestre", "Curso", "Email Insper", "Email Pessoal", "Primeira Opção", "Segunda Opção", "Terceira Opção", "Quarta Opção"},
			campos:    getUsuarioFields(5),
			want: map[string]string{
				"timestamp": "TimeStamp", "email_insper": "Email Insper", "email_pessoal": "Email Pessoal",
				"opcao 1": "Primeira Opção", "opcao 2": "Segunda Opção", "opcao 3": "Terceira Opção", "opcao 4": "Quarta Opção", "opcao 5": "",
			},
		},
		{
			nome:      "colunas embaralhadas",
			cabecalho: []string{"Opção 3", "E-mail pessoal", "Curso", "Opção 1", "Nome completo", "Semestre atual", "CPF", "Opção 2", "Telefone", "E-mail institucional"},
			campos:    getUsuarioFields(3),
			want: map[string]string{
				"nome": "Nome completo", "cpf": "CPF", "numero": "Telefone", "semestre": "Semestre atual", "curso": "Curso",
				"email_insper": "E-mail institucional", "email_pessoal": "E-mail pessoal",
				"opcao 1": "Opção 1", "opcao 2": "Opção 2", "opcao 3": "Opção 3", "timestamp": "",
			},
		},
		{
			// "Avaliador" não parece com nenhum campo: fica com a coluna que sobrou
			nome:      "base_exemplo avaliadores",
			cabecalho: []string{"Avaliador", "Email", "Sigla"},
			campos:    getAvaliadorFields(),
			want:      map[string]string{"nome": "Avaliador", "email": "Email", "sigla": "Sigla"},
		},
		{
			nome:      "base_exemplo restrições",
			cabecalho: []string{"CANDIDATOS", "NÃO POSSO", "PREFIRO NÃO"},
			campos:    getRestricaoFields(),
			want:      map[string]string{"candidato": "CANDIDATOS", "naoPosso": "NÃO POSSO", "prefiroNao": "PREFIRO NÃO"},
		},
		{
			nome:      "restrições fora de ordem",
			cabecalho: []string{"PrefiroNao", "Candidato", "NaoPosso"},
			campos:    getRestricaoFields(),
			want:      map[string]string{"candidato": "Candidato", "naoPosso": "NaoPosso", "prefiroNao": "PrefiroNao"},
		},
	}
	for _, c := range casos {
		got := colunasSugeridas(c.cabecalho, c.campos)
		for campo, col := range c.want {
			if got[campo] != col {
				t.Errorf("%s: campo %q → %q, esperado %q", c.nome, campo, got[campo], col)
			}
		}
	}
}

func TestSugestaoSemColunaRepetidaEIndices(t *testing.T) {
	cabecalho := []string{"Nome", "Opção 2", "Opção 1"}
	itens := sugerirMapeamento(cabecalho, getUsuarioFields(3))
	if len(itens) != len(getUsuarioFields(3)) {
		t.Fatalf("esperado um item por campo, obteve %d", len(itens))
	}
	usadas := map[int]bool{}
	for _, m := range itens {
		if m.NomeColuna == "" {
			if m.Indice < len(cabecalho) {
				t.Errorf("campo %q sem coluna deveria ter índice fora do cabeçalho, obteve %d", m.Variavel, m.Indice)
			}
			continue
		}
		if cabecalho[m.Indice] != m.NomeColuna {
			t.Errorf("campo %q: índice %d não corresponde à coluna %q", m.Variavel, m.Indice, m.NomeColuna)
		}
		if usadas[m.Indice] {
			t.Errorf("coluna %q sugerida para mais de um campo", m.NomeColuna)
		}
		usadas[m.Indice] = true
	}
}

func TestSugestaoCabecalhoVazio(t *testing.T) {
	for _, m := range sugerirMapeamento(nil, getAvaliadorFields()) {
		if m.NomeColuna != "" || m.Indice < 0 {
			t.Errorf("sem cabeçalho, esperado campo sem coluna; obteve %+v", m)
		}
	}
}

func TestNumeroDaOpcao(t *testing.T) {
	casos := map[string]int{ // 0 = não é coluna de opção
		"Opcao 1": 1, "Primeira Opção": 1, "1ª opção": 1, "Opção1": 1,
		"Quinta Opção": 5, "Horário 2": 2, "Disponibilidade 3": 3, "Preferência 4": 4,
		"Opção de horário 6": 6, "Décima Opção": 10,
		"Opção": 0, "Opção 11": 0, "Segunda-feira": 0, "Nome": 0, "Observações": 0,
		"Email 2": 0, "Opção 1 ou 2": 0,
	}
	for col, want := range casos {
		got, ok := numeroDaOpcao(col)
		if (want == 0 && ok) || (want != 0 && (!ok || got != want)) {
			t.Errorf("numeroDaOpcao(%q) = %d, %v; esperado %d", col, got, ok, want)
		}
	}
}

func TestContarOpcoes(t *testing.T) {
	base := []string{"TimeStamp", "Nome", "CPF", "Numero"}
	casos := []struct {
		extras []string
		want   int
	}{
		{[]string{"Primeira Opção", "Segunda Opção", "Terceira Opção"}, 3},
		{[]string{"Primeira Opção", "Segunda Opção", "Terceira Opção", "Quarta Opção", "Quinta Opção"}, 5},
		{[]string{"Opção 1", "Opção 2", "Opção 4"}, 4}, // a 3 falta: fica sem coluna
		{[]string{"Observações"}, 0},
	}
	for _, c := range casos {
		if got := contarOpcoes(append(append([]string{}, base...), c.extras...)); got != c.want {
			t.Errorf("contarOpcoes(%q) = %d, esperado %d", c.extras, got, c.want)
		}
	}
}

// Uma coluna que sobrou nunca vira opção de horário.
func TestOpcaoNaoPegaColunaQueSobrou(t *testing.T) {
	cabecalho := []string{"Nome", "CPF", "Primeira Opção", "Segunda Opção", "Observações"}
	got := colunasSugeridas(cabecalho, getUsuarioFields(4))
	if got["opcao 1"] != "Primeira Opção" || got["opcao 2"] != "Segunda Opção" {
		t.Errorf("opções 1 e 2 = %q, %q", got["opcao 1"], got["opcao 2"])
	}
	if got["opcao 3"] != "" || got["opcao 4"] != "" {
		t.Errorf("opções sem coluna de opção ficaram com %q e %q", got["opcao 3"], got["opcao 4"])
	}
	// os outros campos continuam pegando o que sobrou, como antes
	if got["timestamp"] != "Observações" {
		t.Errorf("timestamp = %q, esperado a coluna que sobrou", got["timestamp"])
	}
}
