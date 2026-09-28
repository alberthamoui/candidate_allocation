package main

import (
	"net/url"
	"reflect"
	"strings"
	"testing"
)

func TestUnidadesCriterios(t *testing.T) {
	// 3 valores; mesa com 3 do valor 0, 1 do valor 1 e nenhum do valor 2
	cont := []int{3, 1, 0}
	casos := []struct {
		cp   criterioProb
		want int
	}{
		{criterioProb{tipo: CRITERIO_MISTURAR}, 3},                                           // pares iguais: C(3,2)
		{criterioProb{tipo: CRITERIO_AGRUPAR}, 3},                                            // pares diferentes: C(4,2) - 3
		{criterioProb{tipo: CRITERIO_MAXIMO, limite: 2, alvo: []bool{true, true, true}}, 1},  // 3 > 2
		{criterioProb{tipo: CRITERIO_MAXIMO, limite: 2, alvo: []bool{false, true, true}}, 0}, // valor 0 fora
		{criterioProb{tipo: CRITERIO_MINIMO, limite: 2, alvo: []bool{true, true, true}}, 1},  // valor 1 sozinho
		{criterioProb{tipo: CRITERIO_MINIMO, limite: 4, alvo: []bool{true, false, true}}, 1}, // 3 de 4; valor 2 ausente não conta
		{criterioProb{tipo: CRITERIO_UM_DE_CADA, alvo: []bool{true, true, true}}, 1},         // falta o valor 2
		{criterioProb{tipo: CRITERIO_UM_DE_CADA, alvo: []bool{true, true, false}}, 0},
	}
	for _, c := range casos {
		c.cp.nValores = len(cont)
		if got := c.cp.unidades(cont); got != c.want {
			t.Errorf("%s (limite %d, alvo %v): %d unidades, esperado %d", c.cp.tipo, c.cp.limite, c.cp.alvo, got, c.want)
		}
	}
}

func TestValidarCriterios(t *testing.T) {
	validos := []CriterioAlocacao{
		{Tipo: CRITERIO_MISTURAR, Coluna: COLUNA_CURSO, Peso: 3},
		{Tipo: CRITERIO_AGRUPAR, Coluna: COLUNA_SEMESTRE, Peso: 1},
		{Tipo: CRITERIO_MAXIMO, Coluna: COLUNA_CURSO, Limite: 2, Peso: 10},
		{Tipo: CRITERIO_MINIMO, Coluna: COLUNA_SEMESTRE, Valores: []string{"1"}, Limite: 2, Peso: 3},
		{Tipo: CRITERIO_UM_DE_CADA, Coluna: COLUNA_CURSO, Valores: []string{"Direito"}, Peso: 3},
	}
	for _, c := range validos {
		if err := c.validar(); err != nil {
			t.Errorf("%+v deveria ser válido: %v", c, err)
		}
	}
	invalidos := []CriterioAlocacao{
		{Tipo: "outro", Coluna: COLUNA_CURSO, Peso: 3},
		{Tipo: CRITERIO_MISTURAR, Coluna: "nome", Peso: 3},
		{Tipo: CRITERIO_MISTURAR, Coluna: COLUNA_CURSO, Peso: 0},
		{Tipo: CRITERIO_MISTURAR, Coluna: COLUNA_CURSO, Peso: LIMITE_PESO + 1},
		{Tipo: CRITERIO_MAXIMO, Coluna: COLUNA_CURSO, Limite: 0, Peso: 3},
		{Tipo: CRITERIO_MINIMO, Coluna: COLUNA_CURSO, Limite: 1, Peso: 3},
		{Tipo: CRITERIO_UM_DE_CADA, Coluna: COLUNA_CURSO, Peso: 3},
	}
	for _, c := range invalidos {
		if c.validar() == nil {
			t.Errorf("%+v deveria ser inválido", c)
		}
	}
	pa := parametrosAlocacaoPadrao()
	for range LIMITE_CRITERIOS + 1 {
		pa.Criterios = append(pa.Criterios, validos[0])
	}
	if pa.validar() == nil {
		t.Errorf("mais de %d critérios deveria ser inválido", LIMITE_CRITERIOS)
	}
}

var criteriosDeTeste = []CriterioAlocacao{
	{Tipo: CRITERIO_MISTURAR, Coluna: COLUNA_SEMESTRE, Peso: 3},
	{Tipo: CRITERIO_AGRUPAR, Coluna: COLUNA_CURSO, Peso: 1},
	{Tipo: CRITERIO_MAXIMO, Coluna: COLUNA_CURSO, Limite: 2, Peso: 3},
	{Tipo: CRITERIO_MINIMO, Coluna: COLUNA_SEMESTRE, Valores: []string{"1", "2"}, Limite: 2, Peso: 3},
	{Tipo: CRITERIO_UM_DE_CADA, Coluna: COLUNA_CURSO, Valores: []string{"direito", "Economia", "Curso que não existe"}, Peso: 3},
}

// As contagens mantidas a cada movimento dão o mesmo custo que contar do zero.
func TestCriteriosCustoIncremental(t *testing.T) {
	s := carregarSessaoXLSX(t, "Excels/teste_oficial.xlsx", 5)
	d := dadosDaSessao(t, s)
	p := montarProblema(parametrosAlocacaoPadrao(), d.horarios, d.avals, d.prefs, d.hard, d.soft)
	p.aplicarCriterios(criteriosDeTeste, d.atributos)

	e := p.novoEstado(5)
	e.definirPesos(0.5)
	for i := 0; i < 20_000; i++ {
		e.passo(20)
	}
	for m := range e.membros {
		if a, b := e.custoCriteriosMesa(m), p.custoCriteriosMembros(e.membros[m]); a != b {
			t.Fatalf("mesa %d: custo incremental %d, contado do zero %d", m, a, b)
		}
	}
	busca, real := e.busca, e.real
	e.definirPesos(0.5)
	if e.busca != busca || e.real != real {
		t.Fatalf("incremental (busca %d, real %d) != recalculado (busca %d, real %d)", busca, real, e.busca, e.real)
	}
}

// metricas conta, nas mesas formadas: pares de candidatos com o mesmo valor
// e quantos valores diferentes há, em média, por mesa.
func metricas(mesas []*Mesa, atributos map[int]map[string]string, coluna string) (paresIguais int, distintosPorMesa float64) {
	distintos := 0
	for _, m := range mesas {
		cont := map[string]int{}
		for _, pid := range m.Candidatos {
			cont[chaveValor(atributos[pid][coluna])]++
		}
		for _, x := range cont {
			paresIguais += x * (x - 1) / 2
		}
		distintos += len(cont)
	}
	return paresIguais, float64(distintos) / float64(len(mesas))
}

func TestCriteriosMudamAAlocacao(t *testing.T) {
	s := carregarSessaoXLSX(t, "Excels/teste_oficial.xlsx", 5)
	d := dadosDaSessao(t, s)
	rodar := func(criterios ...CriterioAlocacao) (ResultadoAlocacao, []*Mesa) {
		param := parametrosAlocacaoPadrao()
		param.Criterios = criterios
		return fazerMelhorAlocacaoMesas(param, d.horarios, d.avals, d.prefs, d.hard, d.soft, d.atributos, nil)
	}

	_, base := rodar()
	paresSem, _ := metricas(base, d.atributos, COLUNA_SEMESTRE)
	_, distCursoSem := metricas(base, d.atributos, COLUNA_CURSO)

	resAgrupar, agrupar := rodar(CriterioAlocacao{Tipo: CRITERIO_AGRUPAR, Coluna: COLUNA_CURSO, Peso: 10})
	_, distCursoCom := metricas(agrupar, d.atributos, COLUNA_CURSO)
	if distCursoCom >= distCursoSem {
		t.Errorf("agrupar curso: %.2f cursos por mesa, sem o critério %.2f; esperado menos", distCursoCom, distCursoSem)
	}

	resMisturar, misturar := rodar(CriterioAlocacao{Tipo: CRITERIO_MISTURAR, Coluna: COLUNA_SEMESTRE, Peso: 10})
	paresCom, _ := metricas(misturar, d.atributos, COLUNA_SEMESTRE)
	if paresCom >= paresSem {
		t.Errorf("misturar semestre: %d pares com o mesmo semestre, sem o critério %d; esperado menos", paresCom, paresSem)
	}
	t.Logf("cursos por mesa: %.2f → %.2f agrupando | pares de mesmo semestre: %d → %d misturando", distCursoSem, distCursoCom, paresSem, paresCom)

	// a pontuação desconta exatamente o custo dos critérios
	for _, caso := range []struct {
		res   ResultadoAlocacao
		mesas []*Mesa
		c     CriterioAlocacao
	}{{resAgrupar, agrupar, CriterioAlocacao{Tipo: CRITERIO_AGRUPAR, Coluna: COLUNA_CURSO, Peso: 10}},
		{resMisturar, misturar, CriterioAlocacao{Tipo: CRITERIO_MISTURAR, Coluna: COLUNA_SEMESTRE, Peso: 10}}} {
		base, _, _ := pontuarResultado(caso.res, caso.mesas, d.prefs, d.hard, d.soft)
		pares, _ := metricas(caso.mesas, d.atributos, caso.c.Coluna)
		penal := 10 * pares // misturar: pares iguais
		if caso.c.Tipo == CRITERIO_AGRUPAR {
			penal = 0
			for _, m := range caso.mesas {
				n := len(m.Candidatos)
				iguais, _ := metricas([]*Mesa{m}, d.atributos, caso.c.Coluna)
				penal += 10 * (n*(n-1)/2 - iguais)
			}
		}
		if caso.res.Pontuacao != base-penal {
			t.Errorf("%s: pontuação %d, esperado %d - %d", caso.c.Tipo, caso.res.Pontuacao, base, penal)
		}
	}
}

func TestLerParametrosComCriterios(t *testing.T) {
	q := url.Values{"criterios": {`[{"tipo":"maximo","coluna":"curso","valores":["Direito"],"limite":2,"peso":3}]`}}
	pa, err := lerParametros(q)
	if err != nil {
		t.Fatal(err)
	}
	want := []CriterioAlocacao{{Tipo: CRITERIO_MAXIMO, Coluna: COLUNA_CURSO, Valores: []string{"Direito"}, Limite: 2, Peso: 3}}
	if !reflect.DeepEqual(pa.Criterios, want) {
		t.Errorf("critérios lidos: %+v", pa.Criterios)
	}
	for _, v := range []string{`não é json`, `[{"tipo":"um_de_cada","coluna":"curso","peso":3}]`} {
		if _, err := lerParametros(url.Values{"criterios": {v}}); err == nil {
			t.Errorf("criterios=%s deveria dar erro", v)
		}
	}
}

func TestPreviaComCriterios(t *testing.T) {
	s := carregarSessaoXLSX(t, "Excels/teste_oficial.xlsx", 5)
	param := parametrosAlocacaoPadrao()
	param.Criterios = []CriterioAlocacao{
		{Tipo: CRITERIO_UM_DE_CADA, Coluna: COLUNA_CURSO, Valores: []string{"administracao", "Medicina"}, Peso: 3},
	}
	r := previa(t, s, param)

	cursos := r.ValoresColunas[COLUNA_CURSO]
	if len(cursos) != 7 || len(r.ValoresColunas[COLUNA_SEMESTRE]) != 10 {
		t.Fatalf("valores das colunas: %+v", r.ValoresColunas)
	}
	total := 0
	for _, v := range cursos {
		total += v.Quantidade
	}
	if total != 98 {
		t.Errorf("soma dos cursos = %d, esperado 98", total)
	}
	if sem := r.ValoresColunas[COLUNA_SEMESTRE]; sem[0].Valor != "1" || sem[len(sem)-1].Valor != "10" {
		t.Errorf("semestres fora de ordem numérica: %+v", sem)
	}
	// Medicina não existe; Administracao (10 candidatos) não cobre as 13+ mesas
	junto := strings.Join(r.Avisos, " | ")
	if !strings.Contains(junto, `"Medicina"`) || !strings.Contains(junto, `"Administracao"`) {
		t.Errorf("avisos: %q", r.Avisos)
	}
}

func TestQualidadeDosCriterios(t *testing.T) {
	c := func(id int, curso string, sem int) CandidatoResultado {
		return CandidatoResultado{ID: id, Curso: curso, Semestre: sem, Opcao: 1}
	}
	r := AlocacaoResponse{Mesas: []MesaResult{
		{Candidatos: []CandidatoResultado{c(1, "Direito", 1), c(2, "direito", 2), c(3, "Economia", 3)}},
		{Candidatos: []CandidatoResultado{c(4, "Economia", 1), c(5, "Economia", 1)}},
	}}
	item := func(cr CriterioAlocacao) ItemQualidade { return itemCriterio(r, 0, cr) }

	// misturar curso: mesa 1 tem 2 de direito, mesa 2 tem 2 de economia
	if it := item(CriterioAlocacao{Tipo: CRITERIO_MISTURAR, Coluna: COLUNA_CURSO}); it.Valor != 2 || !reflect.DeepEqual(it.Candidatos, []int{1, 2, 4, 5}) {
		t.Errorf("misturar: %+v", it)
	}
	// agrupar curso: só a mesa 1 mistura cursos
	if it := item(CriterioAlocacao{Tipo: CRITERIO_AGRUPAR, Coluna: COLUNA_CURSO}); it.Valor != 1 || !reflect.DeepEqual(it.Candidatos, []int{1, 2, 3}) {
		t.Errorf("agrupar: %+v", it)
	}
	// no máximo 1 de economia: só a mesa 2 passa
	if it := item(CriterioAlocacao{Tipo: CRITERIO_MAXIMO, Coluna: COLUNA_CURSO, Valores: []string{"Economia"}, Limite: 1}); it.Valor != 1 || !reflect.DeepEqual(it.Candidatos, []int{4, 5}) {
		t.Errorf("maximo: %+v", it)
	}
	// semestre: pelo menos 2 quando aparece → mesa 1 tem três sozinhos
	if it := item(CriterioAlocacao{Tipo: CRITERIO_MINIMO, Coluna: COLUNA_SEMESTRE, Limite: 2}); it.Valor != 1 || !reflect.DeepEqual(it.Candidatos, []int{1, 2, 3}) {
		t.Errorf("minimo: %+v", it)
	}
	// pelo menos um de direito e economia: mesa 2 não tem direito
	if it := item(CriterioAlocacao{Tipo: CRITERIO_UM_DE_CADA, Coluna: COLUNA_CURSO, Valores: []string{"DIREITO", "economia"}}); it.Valor != 1 || it.Tom != "atencao" || !reflect.DeepEqual(it.Candidatos, []int{4, 5}) {
		t.Errorf("um de cada: %+v", it)
	}
	if it := item(CriterioAlocacao{Tipo: CRITERIO_MAXIMO, Coluna: COLUNA_CURSO, Limite: 5}); it.Valor != 0 || it.Tom != "bom" || it.Codigo != "criterio_1" {
		t.Errorf("sem desvio: %+v", it)
	}
}
