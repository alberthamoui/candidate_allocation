package main

import (
	"fmt"
	"slices"
	"sort"
	"testing"
)

func itemQualidade(t *testing.T, r AlocacaoResponse, codigo string) ItemQualidade {
	t.Helper()
	for _, it := range r.Qualidade {
		if it.Codigo == codigo {
			return it
		}
	}
	t.Fatalf("item de qualidade %q não encontrado em %+v", codigo, r.Qualidade)
	return ItemQualidade{}
}

func idsOrdenados(ids []int) []int {
	c := slices.Clone(ids)
	sort.Ints(c)
	return c
}

// conferirResultado checa o que a tela usa: opção de cada candidato,
// conflitos, restrições e o relatório de qualidade.
func conferirResultado(t *testing.T, r AlocacaoResponse, nCand int) {
	t.Helper()
	porOpcao := map[int][]int{}
	var comConflito []int
	vistos := 0
	for i, m := range r.Mesas {
		if i > 0 && ordemHorario(r.Mesas[i-1].DiaNome) > ordemHorario(m.DiaNome) {
			t.Errorf("mesas fora de ordem de horário: %q antes de %q", r.Mesas[i-1].DiaNome, m.DiaNome)
		}
		naMesa := map[string]bool{}
		for _, a := range m.Avaliadores {
			naMesa[a.Nome] = true
			if a.Sigla == "" || a.Email == "" {
				t.Errorf("avaliador %q sem sigla ou email", a.Nome)
			}
		}
		for _, c := range m.Candidatos {
			vistos++
			if c.Opcao < 1 || c.Opcao > len(c.Opcoes) || c.Opcoes[c.Opcao-1] != m.DiaNome {
				t.Errorf("%s: opção %d não corresponde ao horário %q (opções %q)", c.Nome, c.Opcao, m.DiaNome, c.Opcoes)
			}
			porOpcao[c.Opcao] = append(porOpcao[c.Opcao], c.ID)
			for _, a := range c.Conflitos {
				if !naMesa[a] || !slices.Contains(c.PrefiroNao, a) {
					t.Errorf("%s: conflito com %q, que não está na mesa ou não é \"prefiro não\"", c.Nome, a)
				}
			}
			for _, a := range c.PrefiroNao {
				if naMesa[a] && !slices.Contains(c.Conflitos, a) {
					t.Errorf("%s: %q está na mesa e é \"prefiro não\", mas não aparece nos conflitos", c.Nome, a)
				}
			}
			for _, a := range c.NaoPosso {
				if naMesa[a] {
					t.Errorf("%s: avaliador \"não posso\" %q na mesa", c.Nome, a)
				}
			}
			if len(c.Conflitos) > 0 {
				comConflito = append(comConflito, c.ID)
			}
		}
	}
	for _, c := range r.NaoAlocadosInfo {
		if c.Opcao != 0 || len(c.Conflitos) != 0 {
			t.Errorf("não alocado %s com opção %d e conflitos %q", c.Nome, c.Opcao, c.Conflitos)
		}
	}
	if vistos+len(r.NaoAlocadosInfo) != nCand || vistos != r.TotalAlocados {
		t.Errorf("%d nas mesas + %d sem mesa != %d candidatos (TotalAlocados %d)", vistos, len(r.NaoAlocadosInfo), nCand, r.TotalAlocados)
	}

	// relatório de qualidade
	soma := 0
	for _, it := range r.Qualidade {
		if it.Valor != len(it.Candidatos) {
			t.Errorf("%s: valor %d, mas %d candidatos", it.Codigo, it.Valor, len(it.Candidatos))
		}
		if len(it.Codigo) > 6 && it.Codigo[:6] == "opcao_" {
			soma += it.Valor
		}
	}
	for k, ids := range porOpcao {
		it := itemQualidade(t, r, fmt.Sprintf("opcao_%d", k))
		if !slices.Equal(idsOrdenados(it.Candidatos), idsOrdenados(ids)) {
			t.Errorf("opcao_%d: candidatos do relatório diferem dos das mesas", k)
		}
	}
	semMesa := itemQualidade(t, r, "nao_alocados")
	if soma+semMesa.Valor != nCand {
		t.Errorf("soma das opções (%d) + sem mesa (%d) != %d", soma, semMesa.Valor, nCand)
	}
	if got := itemQualidade(t, r, "prefiro_nao"); !slices.Equal(idsOrdenados(got.Candidatos), idsOrdenados(comConflito)) {
		t.Errorf("prefiro_nao: %v, esperado %v", got.Candidatos, comConflito)
	}
}

func TestResultadoParaATela(t *testing.T) {
	s := carregarSessaoXLSX(t, "Excels/teste_oficial.xlsx", 5)

	// padrão: todos alocados, 96 na 1ª opção e 2 na 2ª
	r, err := s.RunAlocacao(parametrosAlocacaoPadrao(), func(any) {})
	if err != nil {
		t.Fatal(err)
	}
	conferirResultado(t, r, 98)
	if v := itemQualidade(t, r, "opcao_1").Valor + itemQualidade(t, r, "opcao_2").Valor; v != 98 {
		t.Errorf("1ª + 2ª opção = %d, esperado 98", v)
	}
	if it := itemQualidade(t, r, "nao_alocados"); it.Valor != 0 || it.Tom != "bom" {
		t.Errorf("sem mesa: %+v", it)
	}

	// capacidade curta: há candidatos sem mesa, com suas opções preenchidas
	r, err = s.RunAlocacao(ParametrosAlocacao{MesasPorHorario: 1, MinPessoasPorMesa: 5, MaxPessoasPorMesa: 5, AvaliadoresPorMesa: 5}, func(any) {})
	if err != nil {
		t.Fatal(err)
	}
	conferirResultado(t, r, 98)
	if it := itemQualidade(t, r, "nao_alocados"); it.Valor == 0 || it.Tom != "ruim" {
		t.Errorf("esperado candidatos sem mesa: %+v", it)
	}
	if len(r.NaoAlocadosInfo[0].Opcoes) == 0 {
		t.Errorf("não alocado sem as opções de horário: %+v", r.NaoAlocadosInfo[0])
	}
	if !sort.SliceIsSorted(r.NaoAlocadosInfo, func(i, j int) bool {
		return menorNome(r.NaoAlocadosInfo[i].Nome, r.NaoAlocadosInfo[j].Nome)
	}) {
		t.Error("não alocados fora de ordem alfabética")
	}
}

// Um candidato "prefiro não" que caiu com o avaliador aparece nos conflitos
// e no item de qualidade.
func TestRelatorioQualidadeConflito(t *testing.T) {
	r := AlocacaoResponse{
		Mesas: []MesaResult{{DiaNome: "segunda 8-10", Candidatos: []CandidatoResultado{
			{ID: 1, Opcao: 1, Conflitos: []string{"Beto"}},
			{ID: 2, Opcao: 3},
		}}},
		NaoAlocadosInfo: []CandidatoResultado{{ID: 3}},
	}
	q := relatorioQualidade(r, nil)
	codigos := []string{}
	for _, it := range q {
		codigos = append(codigos, it.Codigo)
	}
	// opção 2 sem ninguém não aparece; a 1ª sempre aparece
	if !slices.Equal(codigos, []string{"opcao_1", "opcao_3", "prefiro_nao", "nao_alocados"}) {
		t.Fatalf("itens: %q", codigos)
	}
	if q[2].Valor != 1 || q[2].Candidatos[0] != 1 || q[2].Tom != "atencao" {
		t.Errorf("prefiro_nao: %+v", q[2])
	}
	if q[1].Tom != "atencao" || q[3].Tom != "ruim" {
		t.Errorf("tons: 3ª opção %q, sem mesa %q", q[1].Tom, q[3].Tom)
	}
}
