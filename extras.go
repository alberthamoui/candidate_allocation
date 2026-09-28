package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ==================================================
// ================= CAMPOS EXTRAS ==================
// ==================================================
//
// Colunas da planilha que não são campos fixos de candidato ou avaliador podem
// entrar como campos extras. No mapeamento, um extra é um MappingItem com
// Variavel = PREFIXO_EXTRA + nome (ex.: "extra:Turma"); o nome vira a chave em
// Extras e o título da coluna na exportação.

const PREFIXO_EXTRA = "extra:"

// nomeExtra devolve o nome do campo extra de uma variável do mapeamento.
func nomeExtra(variavel string) (string, bool) {
	if !strings.HasPrefix(variavel, PREFIXO_EXTRA) {
		return "", false
	}
	return strings.TrimSpace(strings.TrimPrefix(variavel, PREFIXO_EXTRA)), true
}

// chaveExtra normaliza um nome para comparar nomes de campos entre si
// (ignora caixa, acentos, espaços e pontuação).
func chaveExtra(nome string) string {
	return strings.Join(tokensNome(nome), "")
}

// acrescentarExtras sugere como campo extra cada coluna com título que não foi
// usada pelos campos fixos. Nomes repetidos ganham um sufixo " (2)", " (3)"...
func acrescentarExtras(itens []MappingItem, cabecalho []string) []MappingItem {
	usada := make(map[int]bool, len(itens))
	nomes := make(map[string]bool, len(itens))
	for _, m := range itens {
		if m.NomeColuna != "" {
			usada[m.Indice] = true
		}
		nomes[chaveExtra(m.Variavel)] = true
	}
	for j, col := range cabecalho {
		col = strings.TrimSpace(col)
		if usada[j] || col == "" {
			continue
		}
		nome := col
		for n := 2; nomes[chaveExtra(nome)]; n++ {
			nome = fmt.Sprintf("%s (%d)", col, n)
		}
		nomes[chaveExtra(nome)] = true
		itens = append(itens, MappingItem{NomeColuna: cabecalho[j], Indice: j, Variavel: PREFIXO_EXTRA + nome})
	}
	return itens
}

// validarExtras confere os campos extras do mapeamento e devolve seus nomes
// na ordem do mapeamento. Nome vazio, repetido ou igual a um campo fixo é erro.
func validarExtras(itens []MappingItem, campos []string) ([]string, error) {
	vistos := make(map[string]string, len(campos))
	for _, c := range campos {
		vistos[chaveExtra(c)] = c
	}
	var nomes []string
	for _, m := range itens {
		nome, ok := nomeExtra(m.Variavel)
		if !ok {
			continue
		}
		if chaveExtra(nome) == "" {
			return nil, fmt.Errorf("campo extra da coluna %q está sem nome", m.NomeColuna)
		}
		if outro, ok := vistos[chaveExtra(nome)]; ok {
			return nil, fmt.Errorf("o campo extra %q tem o mesmo nome que %q", nome, outro)
		}
		vistos[chaveExtra(nome)] = nome
		nomes = append(nomes, nome)
	}
	return nomes, nil
}

// novosExtras cria os extras de um registro com todos os campos vazios (a
// linha da planilha pode acabar antes da coluna); nil se não houver extras.
func novosExtras(nomes []string) map[string]string {
	if len(nomes) == 0 {
		return nil
	}
	extras := make(map[string]string, len(nomes))
	for _, n := range nomes {
		extras[n] = ""
	}
	return extras
}

// extrasJSON serializa os extras para gravar no banco ("{}" se não houver).
func extrasJSON(extras map[string]string) string {
	if len(extras) == 0 {
		return "{}"
	}
	b, _ := json.Marshal(extras)
	return string(b)
}

// lerExtrasJSON faz o inverso de extrasJSON; texto inválido vira mapa vazio.
func lerExtrasJSON(s string) map[string]string {
	extras := map[string]string{}
	json.Unmarshal([]byte(s), &extras)
	return extras
}
