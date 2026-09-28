package main

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestAcrescentarExtras(t *testing.T) {
	cabecalho := []string{"Nome", "Turma", "CPF", "", "Nome", "turma", "Observações"}
	itens := acrescentarExtras(sugerirMapeamento(cabecalho, []string{"nome", "cpf"}, false), cabecalho)

	var extras []MappingItem
	for _, m := range itens {
		if _, ok := nomeExtra(m.Variavel); ok {
			extras = append(extras, m)
		}
	}
	// coluna sem título fica de fora; nomes repetidos (inclusive com um campo fixo) ganham sufixo
	want := []MappingItem{
		{NomeColuna: "Turma", Indice: 1, Variavel: "extra:Turma"},
		{NomeColuna: "Nome", Indice: 4, Variavel: "extra:Nome (2)"},
		{NomeColuna: "turma", Indice: 5, Variavel: "extra:turma (2)"},
		{NomeColuna: "Observações", Indice: 6, Variavel: "extra:Observações"},
	}
	if !reflect.DeepEqual(extras, want) {
		t.Errorf("extras sugeridos:\n%+v\nesperado:\n%+v", extras, want)
	}
	if _, err := validarExtras(itens, []string{"nome", "cpf"}); err != nil {
		t.Errorf("sugestão deveria ser válida: %v", err)
	}
}

func TestValidarExtras(t *testing.T) {
	campos := []string{"nome", "email_insper"}
	nomes, err := validarExtras([]MappingItem{
		{Variavel: "nome"}, {Variavel: "extra: Turma "}, {Variavel: "extra:Área"},
	}, campos)
	if err != nil || !reflect.DeepEqual(nomes, []string{"Turma", "Área"}) {
		t.Errorf("obteve %q, %v", nomes, err)
	}
	invalidos := [][]MappingItem{
		{{NomeColuna: "X", Variavel: "extra:  "}},              // sem nome
		{{Variavel: "extra:Turma"}, {Variavel: "extra:TURMA"}}, // repetido
		{{Variavel: "extra:Área"}, {Variavel: "extra:area"}},   // repetido sem acento
		{{Variavel: "extra:Email Insper"}},                     // igual a campo fixo
	}
	for _, itens := range invalidos {
		if _, err := validarExtras(itens, campos); err == nil {
			t.Errorf("%+v deveria ser inválido", itens)
		}
	}
}

// planilhaComExtras monta um .xlsx com as 3 abas: candidatos e avaliadores com
// colunas extras, e restrições vazias.
func planilhaComExtras(t *testing.T) []byte {
	t.Helper()
	f := excelize.NewFile()
	f.SetSheetName("Sheet1", "Candidatos")
	f.NewSheet("Avaliadores")
	f.NewSheet("Restricoes")
	linhas := func(aba string, dados [][]any) {
		for i, linha := range dados {
			cell, _ := excelize.CoordinatesToCellName(1, i+1)
			f.SetSheetRow(aba, cell, &linha)
		}
	}
	cands := [][]any{{"Nome", "CPF", "Turma", "Semestre", "Opção 1", "Observações"}}
	for i := 1; i <= 6; i++ {
		cands = append(cands, []any{fmt.Sprintf("Cand %d", i), fmt.Sprintf("%011d", i), fmt.Sprintf("T%d", i%2+1), "3", "segunda 8-10", fmt.Sprintf("obs %d", i)})
	}
	// linha que acaba antes da coluna Observações
	cands = append(cands, []any{"Cand 7", "00000000007", "T1", "3", "segunda 8-10"})
	linhas("Candidatos", cands)

	avs := [][]any{{"Nome", "Email", "Sigla", "Área"}}
	for i := 1; i <= 5; i++ {
		avs = append(avs, []any{fmt.Sprintf("Aval %d", i), fmt.Sprintf("a%d@x.com", i), fmt.Sprintf("A%d", i), fmt.Sprintf("area %d", i)})
	}
	linhas("Avaliadores", avs)
	linhas("Restricoes", [][]any{{"Candidato", "NaoPosso", "PrefiroNao"}})

	buf, err := f.WriteToBuffer()
	if err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestExtrasFluxoCompleto(t *testing.T) {
	s := setupSession()

	// sugestão: colunas que sobram viram extras
	mapping, err := s.SuggestMapping(planilhaComExtras(t), 1, s.emailDomain)
	if err != nil {
		t.Fatal(err)
	}
	var extrasSugeridos []string
	for _, m := range mapping {
		if nome, ok := nomeExtra(m.Variavel); ok {
			extrasSugeridos = append(extrasSugeridos, nome)
		}
	}
	if !reflect.DeepEqual(extrasSugeridos, []string{"Turma", "Observações"}) {
		t.Fatalf("extras sugeridos %q", extrasSugeridos)
	}

	// build: extras preenchidos (vazio quando a linha acaba antes)
	usuarios, err := s.BuildUsuariosWithMapping(mapping)
	if err != nil {
		t.Fatal(err)
	}
	if got := usuarios.Usuarios[1].Usuario.Extras; !reflect.DeepEqual(got, map[string]string{"Turma": "T2", "Observações": "obs 1"}) {
		t.Errorf("extras do candidato 1: %v", got)
	}
	if got := usuarios.Usuarios[7].Usuario.Extras; !reflect.DeepEqual(got, map[string]string{"Turma": "T1", "Observações": ""}) {
		t.Errorf("extras do candidato 7: %v", got)
	}
	var cands []Usuario
	for i := 1; i <= len(usuarios.Usuarios); i++ {
		cands = append(cands, usuarios.Usuarios[i].Usuario)
	}
	s.SaveUsuarios(cands)

	mAv, err := s.SuggestMappingAvaliador()
	if err != nil {
		t.Fatal(err)
	}
	avals, err := s.BuildAvaliadoresWithMapping(mAv)
	if err != nil {
		t.Fatal(err)
	}
	if got := avals[0].Extras; !reflect.DeepEqual(got, map[string]string{"Área": "area 1"}) {
		t.Errorf("extras do avaliador 1: %v", got)
	}
	s.SaveAvaliadores(avals)

	// persistência
	var extrasPessoa, extrasAval string
	s.db.QueryRow(`SELECT extras FROM pessoa WHERE nome = 'Cand 2'`).Scan(&extrasPessoa)
	s.db.QueryRow(`SELECT extras FROM avaliador WHERE sigla = 'A3'`).Scan(&extrasAval)
	if !reflect.DeepEqual(lerExtrasJSON(extrasPessoa), map[string]string{"Turma": "T1", "Observações": "obs 2"}) {
		t.Errorf("extras gravados da pessoa: %s", extrasPessoa)
	}
	if !reflect.DeepEqual(lerExtrasJSON(extrasAval), map[string]string{"Área": "area 3"}) {
		t.Errorf("extras gravados do avaliador: %s", extrasAval)
	}

	// resultado: uma mesa de até 5 → 5 alocados e 2 de fora, todos com seus extras
	param := ParametrosAlocacao{MesasPorHorario: 1, MinPessoasPorMesa: 1, MaxPessoasPorMesa: 5, AvaliadoresPorMesa: 5}
	res, err := s.RunAlocacao(param, func(any) {})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(res.ExtrasCandidatos, []string{"Turma", "Observações"}) {
		t.Errorf("ExtrasCandidatos = %q", res.ExtrasCandidatos)
	}
	if len(res.Mesas) != 1 || len(res.Mesas[0].CandidatosExtras) != len(res.Mesas[0].Candidatos) || len(res.NaoAlocadosInfo) != 2 {
		t.Fatalf("resultado inesperado: %+v", res)
	}
	obs := map[string]string{} // nome → Observações, vindo do resultado
	for i, nome := range res.Mesas[0].Candidatos {
		obs[nome] = res.Mesas[0].CandidatosExtras[i]["Observações"]
	}
	for _, p := range res.NaoAlocadosInfo {
		obs[p.Nome] = p.Extras["Observações"]
	}
	for i := 1; i <= 7; i++ {
		want := fmt.Sprintf("obs %d", i)
		if i == 7 {
			want = ""
		}
		if got := obs[fmt.Sprintf("Cand %d", i)]; got != want {
			t.Errorf("Cand %d: Observações = %q, esperado %q", i, got, want)
		}
	}

	// exportação: colunas extras nas abas Lista e Não Alocados
	b, err := s.ExportResultado()
	if err != nil {
		t.Fatal(err)
	}
	f, err := excelize.OpenReader(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	lista, _ := f.GetRows("Lista")
	if !reflect.DeepEqual(lista[0], []string{"Mesa", "Candidato", "Avaliadores", "Turma", "Observações"}) {
		t.Errorf("cabeçalho da Lista: %q", lista[0])
	}
	for _, linha := range lista[1:] {
		if linha[4] != obs[linha[1]] {
			t.Errorf("Lista, %s: Observações %q, esperado %q", linha[1], linha[4], obs[linha[1]])
		}
	}
	nao, _ := f.GetRows("Não Alocados")
	if !reflect.DeepEqual(nao[0], []string{"Nome", "Email Institucional", "Curso", "Semestre", "Turma", "Observações"}) {
		t.Errorf("cabeçalho de Não Alocados: %q", nao[0])
	}
	if len(nao) != 3 {
		t.Errorf("esperado 2 não alocados na planilha, obteve %d linhas", len(nao)-1)
	}
}

func TestCamposNaoIncluemExtras(t *testing.T) {
	for _, campos := range [][]string{getUsuarioFields(2), getAvaliadorFields()} {
		for _, c := range campos {
			if strings.Contains(c, "extras") || strings.Contains(c, ",") {
				t.Errorf("campo %q não deveria estar na lista de campos mapeáveis %q", c, campos)
			}
		}
	}
	if got := getAvaliadorFields(); !reflect.DeepEqual(got, []string{"nome", "email", "sigla"}) {
		t.Errorf("campos de avaliador: %q", got)
	}
}
