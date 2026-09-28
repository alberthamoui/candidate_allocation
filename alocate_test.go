package main

import (
	"context"
	"math"
	"math/rand"
	"net/url"
	"os"
	"reflect"
	"sort"
	"testing"
)

// carregarSessaoXLSX passa um .xlsx pelo mesmo fluxo do app (mapeamento
// sugerido → build → save) e devolve a sessão com o banco preenchido.
func carregarSessaoXLSX(t testing.TB, path string, nOpcoes int) *Session {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("lendo %s: %v", path, err)
	}
	s := setupSession()

	mapping, err := s.SuggestMapping(data, nOpcoes, s.emailDomain)
	if err != nil {
		t.Fatal(err)
	}
	usuarios, err := s.BuildUsuariosWithMapping(mapping)
	if err != nil {
		t.Fatal(err)
	}
	var cands []Usuario
	for i := 1; i <= len(usuarios.Usuarios); i++ {
		cands = append(cands, usuarios.Usuarios[i].Usuario)
	}
	if err := s.SaveUsuarios(cands); err != nil {
		t.Fatal(err)
	}

	mAv, err := s.SuggestMappingAvaliador()
	if err != nil {
		t.Fatal(err)
	}
	avals, err := s.BuildAvaliadoresWithMapping(mAv)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SaveAvaliadores(avals); err != nil {
		t.Fatal(err)
	}

	mRe, err := s.SuggestMappingRestricao()
	if err != nil {
		t.Fatal(err)
	}
	restricoes, err := s.BuildRestricoesWithMapping(mRe)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SaveRestricoes(restricoes); err != nil {
		t.Fatal(err)
	}
	return s
}

// dadosDaSessao lê os dados da alocação do banco da sessão.
func dadosDaSessao(t testing.TB, s *Session) dadosAlocacao {
	t.Helper()
	d, err := s.carregarDados()
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func previa(t testing.TB, s *Session, param ParametrosAlocacao) CapacidadeResponse {
	t.Helper()
	r, err := s.PreviaCapacidade(param)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

// validarAlocacao confere todas as regras que uma alocação precisa respeitar.
func validarAlocacao(t *testing.T, param ParametrosAlocacao, res ResultadoAlocacao, mesas []*Mesa, prefs map[int][]int, hard map[int]map[int]bool) {
	t.Helper()
	vistos := map[int]bool{}
	avPorHorario := map[int]map[int]bool{}
	mesasPorHorario := map[int]int{}
	for _, m := range mesas {
		if n := len(m.Candidatos); n < param.MinPessoasPorMesa || n > param.MaxPessoasPorMesa {
			t.Errorf("%s: %d candidatos (esperado %d..%d)", m.Descricao, n, param.MinPessoasPorMesa, param.MaxPessoasPorMesa)
		}
		if len(m.Avaliadores) != param.AvaliadoresPorMesa {
			t.Errorf("%s: %d avaliadores (esperado %d)", m.Descricao, len(m.Avaliadores), param.AvaliadoresPorMesa)
		}
		if mesasPorHorario[m.DiaID]++; mesasPorHorario[m.DiaID] > param.MesasPorHorario {
			t.Errorf("%s: mais de %d mesas no horário", m.Descricao, param.MesasPorHorario)
		}
		if avPorHorario[m.DiaID] == nil {
			avPorHorario[m.DiaID] = map[int]bool{}
		}
		for _, a := range m.Avaliadores {
			if avPorHorario[m.DiaID][a] {
				t.Errorf("%s: avaliador %d em duas mesas do mesmo horário", m.Descricao, a)
			}
			avPorHorario[m.DiaID][a] = true
		}
		for _, c := range m.Candidatos {
			if vistos[c] {
				t.Errorf("candidato %d em mais de uma mesa", c)
			}
			vistos[c] = true
			if res.Alocacao[c] != m.ID {
				t.Errorf("candidato %d: Alocacao aponta mesa %d, mas está na %d", c, res.Alocacao[c], m.ID)
			}
			escolheu := false
			for _, h := range prefs[c] {
				escolheu = escolheu || h == m.DiaID
			}
			if !escolheu {
				t.Errorf("candidato %d alocado em horário que não escolheu (%s)", c, m.Descricao)
			}
			for _, a := range m.Avaliadores {
				if hard[a][c] {
					t.Errorf("%s: candidato %d com avaliador %d viola 'não posso'", m.Descricao, c, a)
				}
			}
		}
	}
	if len(vistos) != len(res.Alocacao) || res.Alocados != len(res.Alocacao) {
		t.Errorf("contagem inconsistente: %d nas mesas, %d em Alocacao, Alocados=%d", len(vistos), len(res.Alocacao), res.Alocados)
	}
}

func TestAlocacaoTesteOficial(t *testing.T) {
	s := carregarSessaoXLSX(t, "Excels/teste_oficial.xlsx", 5)
	d := dadosDaSessao(t, s)
	avals, hard, soft, horarios, prefs := d.avals, d.hard, d.soft, d.horarios, d.prefs

	param := parametrosAlocacaoPadrao()
	res, mesas := fazerMelhorAlocacaoMesas(context.Background(), param, horarios, avals, prefs, hard, soft, nil, nil)
	validarAlocacao(t, param, res, mesas, prefs, hard)

	score, pen, _ := pontuarResultado(res, mesas, prefs, hard, soft)
	if score != res.Pontuacao {
		t.Errorf("Pontuacao %d difere de pontuarResultado %d", res.Pontuacao, score)
	}
	limite := SCORE_BASE - montarProblema(param, horarios, avals, prefs, hard, soft).limiteInferior()
	t.Logf("score=%d (máximo teórico %d) | alocados=%d/%d | penalidades=%v", score, limite, res.Alocados, len(prefs), pen)
	if pen["nao_alocado"] != 0 {
		t.Errorf("%d candidatos não alocados; há capacidade para todos", pen["nao_alocado"])
	}
}

func TestAlocacaoSemAvaliadoresSuficientes(t *testing.T) {
	// base_exemplo tem 3 avaliadores: não dá para formar mesa de 5
	s := carregarSessaoXLSX(t, "Excels/base_exemplo.xlsx", 5)
	d := dadosDaSessao(t, s)
	res, mesas := fazerMelhorAlocacaoMesas(context.Background(), parametrosAlocacaoPadrao(), d.horarios, d.avals, d.prefs, d.hard, d.soft, nil, nil)
	if len(mesas) != 0 || res.Alocados != 0 {
		t.Errorf("esperado nenhuma mesa, obteve %d mesas e %d alocados", len(mesas), res.Alocados)
	}
}

func TestAlocacaoParametrosEditaveis(t *testing.T) {
	s := carregarSessaoXLSX(t, "Excels/teste_oficial.xlsx", 5)
	d := dadosDaSessao(t, s)
	avals, hard, soft, horarios, prefs := d.avals, d.hard, d.soft, d.horarios, d.prefs

	casos := []ParametrosAlocacao{
		{MesasPorHorario: 2, MinPessoasPorMesa: 3, MaxPessoasPorMesa: 6, AvaliadoresPorMesa: 3},
		{MesasPorHorario: 4, MinPessoasPorMesa: 4, MaxPessoasPorMesa: 4, AvaliadoresPorMesa: 4},
		{MesasPorHorario: 1, MinPessoasPorMesa: 6, MaxPessoasPorMesa: 10, AvaliadoresPorMesa: 7},
		{MesasPorHorario: 10, MinPessoasPorMesa: 1, MaxPessoasPorMesa: 2, AvaliadoresPorMesa: 1},
	}
	for _, param := range casos {
		res, mesas := fazerMelhorAlocacaoMesas(context.Background(), param, horarios, avals, prefs, hard, soft, nil, nil)
		validarAlocacao(t, param, res, mesas, prefs, hard)
		score, _, _ := pontuarResultado(res, mesas, prefs, hard, soft)
		if score != res.Pontuacao {
			t.Errorf("%+v: Pontuacao %d difere de pontuarResultado %d", param, res.Pontuacao, score)
		}
		p := montarProblema(param, horarios, avals, prefs, hard, soft)
		if res.Alocados > p.maxAlocaveis() {
			t.Errorf("%+v: %d alocados, mais que o máximo possível %d", param, res.Alocados, p.maxAlocaveis())
		}
		t.Logf("%+v: %d mesas, %d/%d alocados (máximo %d), score %d", param, len(mesas), res.Alocados, len(prefs), p.maxAlocaveis(), res.Pontuacao)
	}
}

func TestParametrosValidar(t *testing.T) {
	if err := parametrosAlocacaoPadrao().validar(); err != nil {
		t.Fatalf("padrão inválido: %v", err)
	}
	invalidos := []ParametrosAlocacao{
		{MesasPorHorario: 0, MinPessoasPorMesa: 5, MaxPessoasPorMesa: 8, AvaliadoresPorMesa: 5},
		{MesasPorHorario: LIMITE_MESAS_POR_HORARIO + 1, MinPessoasPorMesa: 5, MaxPessoasPorMesa: 8, AvaliadoresPorMesa: 5},
		{MesasPorHorario: 5, MinPessoasPorMesa: 0, MaxPessoasPorMesa: 8, AvaliadoresPorMesa: 5},
		{MesasPorHorario: 5, MinPessoasPorMesa: 9, MaxPessoasPorMesa: 8, AvaliadoresPorMesa: 5},
		{MesasPorHorario: 5, MinPessoasPorMesa: 5, MaxPessoasPorMesa: LIMITE_PESSOAS_POR_MESA + 1, AvaliadoresPorMesa: 5},
		{MesasPorHorario: 5, MinPessoasPorMesa: 5, MaxPessoasPorMesa: 8, AvaliadoresPorMesa: 0},
		{MesasPorHorario: 5, MinPessoasPorMesa: 5, MaxPessoasPorMesa: 8, AvaliadoresPorMesa: LIMITE_AVALIADORES_POR_MESA + 1},
	}
	for _, pa := range invalidos {
		if pa.validar() == nil {
			t.Errorf("%+v deveria ser inválido", pa)
		}
	}
}

func TestLerParametros(t *testing.T) {
	pa, err := lerParametros(url.Values{"mesas_por_horario": {"2"}, "avaliadores_por_mesa": {"3"}})
	if err != nil {
		t.Fatal(err)
	}
	want := parametrosAlocacaoPadrao()
	want.MesasPorHorario, want.AvaliadoresPorMesa = 2, 3
	if !reflect.DeepEqual(pa, want) {
		t.Errorf("obteve %+v, esperado %+v", pa, want)
	}
	if _, err := lerParametros(url.Values{"min_pessoas_por_mesa": {"abc"}}); err == nil {
		t.Error("esperado erro para valor não numérico")
	}
	if _, err := lerParametros(url.Values{"min_pessoas_por_mesa": {"9"}, "max_pessoas_por_mesa": {"8"}}); err == nil {
		t.Error("esperado erro para mínimo maior que o máximo")
	}
}

func TestPreviaCapacidade(t *testing.T) {
	s := carregarSessaoXLSX(t, "Excels/teste_oficial.xlsx", 5)

	// padrão: 19 avaliadores / 5 = 3 mesas por horário, 9 horários
	r := previa(t, s, parametrosAlocacaoPadrao())
	if r.Candidatos != 98 || r.Avaliadores != 19 || r.MesasPorHorario != 3 || r.CapacidadePorHorario != 24 || r.CapacidadeTotal != 9*24 || r.MaxAlocaveis != 98 {
		t.Errorf("padrão: %+v", r)
	}
	if len(r.Horarios) != 9 {
		t.Errorf("esperado 9 horários, obteve %d", len(r.Horarios))
	}
	primeiras := 0
	for _, h := range r.Horarios {
		primeiras += h.PrimeiraOpcao
		if h.PrimeiraOpcao > h.Interessados {
			t.Errorf("%s: 1ª opção %d > interessados %d", h.Descricao, h.PrimeiraOpcao, h.Interessados)
		}
	}
	if primeiras != 98 {
		t.Errorf("soma das 1ªs opções = %d, esperado 98", primeiras)
	}
	if r.Horarios[0].Descricao != "segunda 8-10" || r.Horarios[len(r.Horarios)-1].Descricao != "sexta 8-10" {
		t.Errorf("horários fora de ordem: %v", r.Horarios)
	}
	// pediu 5 mesas mas só cabem 3: avisa
	if len(r.Avisos) != 1 {
		t.Errorf("esperado 1 aviso (mesas limitadas pelos avaliadores), obteve %q", r.Avisos)
	}

	// 1 mesa de até 5 por horário: 45 vagas para 98 candidatos
	r = previa(t, s, ParametrosAlocacao{MesasPorHorario: 1, MinPessoasPorMesa: 5, MaxPessoasPorMesa: 5, AvaliadoresPorMesa: 5})
	if r.CapacidadeTotal != 45 || r.MaxAlocaveis > 45 || len(r.Avisos) == 0 {
		t.Errorf("capacidade curta: %+v", r)
	}

	// base_exemplo tem 3 avaliadores: nenhuma mesa de 5
	r = previa(t, carregarSessaoXLSX(t, "Excels/base_exemplo.xlsx", 5), parametrosAlocacaoPadrao())
	if r.Avaliadores != 3 || r.MesasPorHorario != 0 || r.CapacidadeTotal != 0 || r.MaxAlocaveis != 0 || len(r.Avisos) == 0 {
		t.Errorf("sem mesas: %+v", r)
	}
}

func TestLimiteInferior(t *testing.T) {
	// 10 avaliadores → 2 mesas por horário → 16 vagas por horário.
	// 20 candidatos querem A e depois B: 16 ficam em A (custo 0), 4 em B (custo 1 cada).
	horarios := map[int]*Horario{1: {ID: 1, Descricao: "A"}, 2: {ID: 2, Descricao: "B"}}
	var avals []*Avaliador
	for i := 1; i <= 10; i++ {
		avals = append(avals, &Avaliador{ID: i})
	}
	prefs := map[int][]int{}
	for c := 1; c <= 20; c++ {
		prefs[c] = []int{1, 2}
	}
	p := montarProblema(parametrosAlocacaoPadrao(), horarios, avals, prefs, nil, nil)
	if got, want := p.limiteInferior(), 4*-PONTOS_OPCAO_2; got != want {
		t.Errorf("limiteInferior = %d, esperado %d", got, want)
	}
	if got := p.maxAlocaveis(); got != 20 {
		t.Errorf("maxAlocaveis = %d, esperado 20", got)
	}

	// só 1 horário e 17 candidatos: 1 fica de fora
	for c := 1; c <= 17; c++ {
		prefs[c] = []int{1}
	}
	for c := 18; c <= 20; c++ {
		delete(prefs, c)
	}
	p = montarProblema(parametrosAlocacaoPadrao(), horarios, avals, prefs, nil, nil)
	if got := p.limiteInferior(); got != custoNaoAlocado {
		t.Errorf("limiteInferior = %d, esperado %d", got, custoNaoAlocado)
	}
	if got := p.maxAlocaveis(); got != 16 {
		t.Errorf("maxAlocaveis = %d, esperado 16", got)
	}
}

func TestPontosOpcao(t *testing.T) {
	want := []int{0, -1, -3, -5, -7, -9, -11}
	for nivel, w := range want {
		if got := pontosOpcao(nivel); got != w {
			t.Errorf("pontosOpcao(%d) = %d, esperado %d", nivel, got, w)
		}
	}
}

func TestHungaroContraForcaBruta(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	for caso := 0; caso < 300; caso++ {
		n := 1 + rng.Intn(5)
		m := n + rng.Intn(4)
		h := novoHungaro(n, m)
		for i := range h.custo[:n*m] {
			h.custo[i] = rng.Intn(20)
		}
		// força bruta: todas as escolhas de colunas distintas para as linhas
		melhor := math.MaxInt
		usada := make([]bool, m)
		var rec func(i, soma int)
		rec = func(i, soma int) {
			if i == n {
				melhor = min(melhor, soma)
				return
			}
			for j := 0; j < m; j++ {
				if !usada[j] {
					usada[j] = true
					rec(i+1, soma+h.custo[i*m+j])
					usada[j] = false
				}
			}
		}
		rec(0, 0)

		// com corte: o custo parcial é limite inferior e só para se passar do corte
		corte := rng.Intn(40) - 5
		parcial, completo := h.resolver(n, m, corte)
		if parcial > melhor || (!completo && parcial <= corte) || (completo && parcial != melhor) {
			t.Fatalf("caso %d: corte %d → custo %d completo=%v, ótimo %d", caso, corte, parcial, completo, melhor)
		}
		if !completo && melhor <= corte {
			t.Fatalf("caso %d: desistiu com ótimo %d <= corte %d", caso, melhor, corte)
		}

		got, _ := h.resolver(n, m, math.MaxInt)
		soma, linhas := 0, map[int]bool{}
		for j := 1; j <= m; j++ {
			if i := h.p[j]; i > 0 {
				soma += h.custo[(i-1)*m+j-1]
				linhas[i] = true
			}
		}
		if got != melhor || soma != melhor || len(linhas) != n {
			t.Fatalf("caso %d (%dx%d): húngaro=%d, atribuição soma %d com %d linhas, força bruta=%d", caso, n, m, got, soma, len(linhas), melhor)
		}
	}
}

func TestCustoIncrementalConsistente(t *testing.T) {
	s := carregarSessaoXLSX(t, "Excels/teste_oficial.xlsx", 5)
	d := dadosDaSessao(t, s)
	p := montarProblema(parametrosAlocacaoPadrao(), d.horarios, d.avals, d.prefs, d.hard, d.soft)

	e := p.novoEstado(1)
	e.definirPesos(0.5)
	for i := 0; i < 20_000; i++ {
		e.passo(20)
	}
	busca, real := e.busca, e.real
	e.definirPesos(0.5) // recalcula tudo do zero
	if e.busca != busca || e.real != real {
		t.Fatalf("incremental (busca %d, real %d) != recalculado (busca %d, real %d)", busca, real, e.busca, e.real)
	}
}

// Os atalhos de aceitar (limite inferior e corte do húngaro) não podem mudar
// a decisão: ela tem de ser a mesma do critério de Metropolis com o delta
// calculado do zero.
func TestAceitarEquivaleAoCalculoCompleto(t *testing.T) {
	s := carregarSessaoXLSX(t, "Excels/teste_oficial.xlsx", 5)
	d := dadosDaSessao(t, s)
	p := montarProblema(parametrosAlocacaoPadrao(), d.horarios, d.avals, d.prefs, d.hard, d.soft)

	e := p.novoEstado(3)
	scratch := make([][]int, p.nMesas)
	custoCompleto := func() int {
		total := 0
		for s := 0; s < p.nSlot; s++ {
			conf, _, _ := e.calcSlot(s, scratch, math.MaxInt)
			total += e.baseDoSlot(s) + conf
		}
		for _, m := range e.mesaDe {
			if m < 0 {
				total += custoNaoAlocado
			}
		}
		return total
	}

	rng := rand.New(rand.NewSource(11))
	aceitos := 0
	for tentativa := 0; tentativa < 20_000; tentativa++ {
		if tentativa%2_000 == 0 {
			e.definirPesos(float64(tentativa) / 20_000)
		}
		temp := []float64{0.05, 1, 10, 50}[rng.Intn(4)]
		c := rng.Intn(p.nCand)
		m1 := e.mesaDe[c]
		dest := p.prefs[c][rng.Intn(len(p.prefs[c]))]*p.nMesas + rng.Intn(p.nMesas)
		troca := rng.Intn(2) == 0 // senão, move
		c2, fora := -1, 0
		if troca {
			if dest == m1 || len(e.membros[dest]) == 0 {
				continue
			}
			c2 = e.membros[dest][rng.Intn(len(e.membros[dest]))]
			if m1 >= 0 && p.custoPref[c2*p.nSlot+m1/p.nMesas] < 0 {
				continue
			}
		} else {
			if rng.Intn(10) == 0 {
				dest = -1
			}
			if dest == m1 || (dest >= 0 && len(e.membros[dest]) >= p.maxPessoas) {
				continue
			}
			if m1 < 0 {
				fora -= custoNaoAlocado
			}
			if dest < 0 {
				fora += custoNaoAlocado
			}
		}
		antes := e.busca
		e.mover(c, m1, dest)
		if troca {
			e.mover(c2, dest, m1)
		}
		delta := custoCompleto() - antes

		semente := rng.Int63()
		e.rng = rand.New(rand.NewSource(semente))
		u := rand.New(rand.NewSource(semente)).Float64()
		esperado := u == 0 || float64(delta) <= -temp*math.Log(u)

		if got := e.aceitar(m1, dest, c, c2, fora, temp); got != esperado {
			t.Fatalf("tentativa %d: aceitar=%v, esperado %v (delta %d, T %v, u %v)", tentativa, got, esperado, delta, temp, u)
		} else if got {
			aceitos++
			if e.busca != antes+delta {
				t.Fatalf("tentativa %d: custo %d, esperado %d", tentativa, e.busca, antes+delta)
			}
		} else {
			if troca {
				e.mover(c2, m1, dest)
			}
			e.mover(c, dest, m1)
		}
	}
	t.Logf("%d movimentos aceitos", aceitos)
}

func TestOrdemHorario(t *testing.T) {
	descs := []string{"Sexta 8-10", "terça 10-12", "Segunda 14-16", "outro", "segunda 8-10", "Terca 8-10"}
	sort.Slice(descs, func(i, j int) bool { return ordemHorario(descs[i]) < ordemHorario(descs[j]) })
	want := []string{"segunda 8-10", "Segunda 14-16", "Terca 8-10", "terça 10-12", "Sexta 8-10", "outro"}
	for i := range want {
		if descs[i] != want[i] {
			t.Fatalf("ordem %q, esperado %q", descs, want)
		}
	}
}
