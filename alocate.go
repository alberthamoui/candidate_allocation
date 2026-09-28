package main

// ==================================================
// ============== IMPORTS E CONSTANTES ==============
// ==================================================

import (
	"database/sql"
	"fmt"
	"log"
	"math"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const (
	// CONFIGURAÇÕES DE ALOCAÇÃO: valores padrão, editáveis na tela
	// (ParametrosAlocacao)
	MESAS_POR_HORARIO    = 5
	MIN_PESSOAS_POR_MESA = 5
	MAX_PESSOAS_POR_MESA = 8
	AVALIADORES_POR_MESA = 5
	SCORE_BASE           = 100

	// Limites aceitos para os parâmetros editáveis
	LIMITE_MESAS_POR_HORARIO    = 10
	LIMITE_PESSOAS_POR_MESA     = 20
	LIMITE_AVALIADORES_POR_MESA = 10

	// CONFIGURAÇÕES DO OTIMIZADOR (simulated annealing)
	SA_EXECUCOES    = 8       // execuções independentes em paralelo; fica a melhor
	SA_ITERACOES    = 200_000 // movimentos testados por execução
	SA_TEMP_INICIAL = 10.0    // temperatura inicial (na escala dos pontos)
	SA_TEMP_FINAL   = 0.05    // temperatura final
	SA_SEMENTE      = 42      // semente fixa: mesma entrada → mesmo resultado

	// Penalidades da busca: começam brandas, para a busca conseguir montar
	// mesas aos poucos e reorganizar avaliadores, e endurecem até o valor final
	// ao chegar em SA_FRACAO_RAMPA das iterações.
	SA_PESO_FALTA_INICIAL    = 100       // por vaga faltando numa mesa incompleta
	SA_PESO_FALTA_FINAL      = 4000      // mesa de 4 fica pior que 4 sem mesa
	SA_PESO_PROIBIDO_INICIAL = 100       // por conflito "não posso"
	SA_PESO_PROIBIDO_FINAL   = 1_000_000 // na prática, proibido
	SA_FRACAO_RAMPA          = 0.7

	SA_PCT_MOVER = 60 // % dos movimentos que movem um candidato; o resto troca dois
)

// ==================================================
// =========== CRITÉRIOS DE PONTUAÇÃO ===============
// ==================================================
const (
	PONTOS_OPCAO_1         = 0     // candidato alocado na 1ª opção de horário
	PONTOS_OPCAO_2         = -1    // candidato alocado na 2ª opção de horário
	PONTOS_OPCAO_3         = -3    // candidato alocado na 3ª opção de horário
	PONTOS_OPCAO_4         = -5    // candidato alocado na 4ª opção de horário
	PONTOS_OPCAO_5         = -7    // candidato alocado na 5ª opção de horário
	PENALIDADE_SOFT        = -5    // violação de restrição "prefiro não" por avaliador
	PENALIDADE_NAO_ALOCADO = -1000 // candidato que não foi alocado
	PENALIDADE_HARD        = -1000 // violação de restrição "não posso" por avaliador
)

// pontosOpcao devolve os pontos de alocar um candidato na opção de índice
// nivel (0 = 1ª opção). Da 6ª opção em diante, cada nível custa mais 2 pontos.
func pontosOpcao(nivel int) int {
	pontos := [...]int{PONTOS_OPCAO_1, PONTOS_OPCAO_2, PONTOS_OPCAO_3, PONTOS_OPCAO_4, PONTOS_OPCAO_5}
	if nivel < len(pontos) {
		return pontos[nivel]
	}
	return PONTOS_OPCAO_5 - 2*(nivel-len(pontos)+1)
}

// ==================================================
// ==================== STRUCTS =====================
// ==================================================

type Mesa struct {
	ID          int    // único (ex.: 301 = horário 3, mesa 2)
	DiaID       int    // id do horário
	Descricao   string // "quarta 14-16 - mesa 2"
	Candidatos  []int
	Avaliadores []int
}

type ResultadoAlocacao struct {
	Alocacao  map[int]int // pessoa_id → mesa.ID
	Pontuacao int
	Alocados  int
}

type Avaliador struct {
	ID    int    `json:"id"`
	Nome  string `json:"nome"`
	Email string `json:"email"`
}

type Horario struct {
	ID         int
	Descricao  string
	Candidatos []int
}

// ==================================================
// =========== CARREGAMENTO DE DADOS DB ============
// ==================================================

func carregarHorarios(db *sql.DB) map[int]*Horario {
	horarios := make(map[int]*Horario)
	rows, err := db.Query(`SELECT id, opcao FROM opcoes_horario`)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var h Horario
		if err := rows.Scan(&h.ID, &h.Descricao); err != nil {
			log.Fatal(err)
		}
		h.Candidatos = []int{}
		horarios[h.ID] = &h
	}
	return horarios
}

func carregarDisponibilidades(db *sql.DB, horarios map[int]*Horario) map[int][]int {
	prefs := make(map[int][]int)
	rows, err := db.Query(`SELECT pessoa_id, horario_id, preferencia FROM disponibilidade ORDER BY pessoa_id, preferencia ASC`)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var pid, hid, pref int
		if err := rows.Scan(&pid, &hid, &pref); err != nil {
			log.Fatal(err)
		}

		h, ok := horarios[hid]
		if !ok {
			log.Printf("[WARN] horario_id %d não encontrado na tabela de horários. Ignorando.", hid)
			continue
		}

		h.Candidatos = append(h.Candidatos, pid)
		prefs[pid] = append(prefs[pid], hid)
	}
	return prefs
}

func carregarAvaliadores(db *sql.DB) []*Avaliador {
	rows, err := db.Query(`SELECT id, nome, email FROM avaliador`)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	var avals []*Avaliador
	for rows.Next() {
		var a Avaliador
		if err := rows.Scan(&a.ID, &a.Nome, &a.Email); err != nil {
			log.Fatal(err)
		}
		avals = append(avals, &a)
	}
	return avals
}

func carregarRestricoes(db *sql.DB) (hard map[int]map[int]bool, soft map[int]map[int]bool) {
	hard = make(map[int]map[int]bool)
	soft = make(map[int]map[int]bool)

	loadInto := func(m map[int]map[int]bool, query string) {
		rows, err := db.Query(query)
		if err != nil {
			log.Fatal(err)
		}
		defer rows.Close()
		for rows.Next() {
			var aid, cid int
			if err := rows.Scan(&aid, &cid); err != nil {
				log.Fatal(err)
			}
			if m[aid] == nil {
				m[aid] = make(map[int]bool)
			}
			m[aid][cid] = true
		}
	}

	loadInto(hard, `SELECT avaliador_id, candidato_id FROM restricoesNposso`)
	loadInto(soft, `SELECT avaliador_id, candidato_id FROM restricoesPrefiroN`)
	return
}

// ==================================================
// ============ OTIMIZAÇÃO DA ALOCAÇÃO ==============
// ==================================================
//
// Cada candidato vai para uma mesa de um dos horários que escolheu, ou fica
// sem mesa. Cada horário tem até nMesas mesas, cada uma com avPorMesa
// avaliadores distintos, e uma mesa só é formada com minPessoas..maxPessoas
// candidatos (valores de ParametrosAlocacao). O custo minimizado é
// exatamente -(pontuação) de pontuarResultado; "não posso" é proibido.
//
// A busca é um simulated annealing sobre a divisão dos candidatos em mesas,
// com dois movimentos: mover um candidato para outra mesa (ou tirá-lo de
// mesa) e trocar dois candidatos de mesa.
//
// Os avaliadores não entram na busca: dado quem está em cada mesa, a melhor
// escolha de avaliadores de um horário é um problema de atribuição (o custo
// de pôr o avaliador a na mesa k é a soma dos conflitos dele com os membros),
// que o algoritmo húngaro resolve de forma exata. Cada movimento reotimiza só
// os horários afetados.
//
// Mesas incompletas e conflitos "não posso" têm, na busca, penalidades que
// crescem ao longo da execução (SA_PESO_*), para a busca conseguir montar
// mesas aos poucos; a melhor solução é escolhida sempre pelo custo real.
// Um limite inferior exato da escolha de horários (fluxo de custo mínimo)
// permite parar cedo quando a solução é comprovadamente ótima.

const (
	custoNaoAlocado = -PENALIDADE_NAO_ALOCADO
	custoSoft       = -PENALIDADE_SOFT
	custoProibido   = 1_000_000 // "não posso" no custo real: nunca é a melhor solução
)

// ParametrosAlocacao são as regras de formação das mesas, editáveis na tela.
type ParametrosAlocacao struct {
	MesasPorHorario    int `json:"mesas_por_horario"`
	MinPessoasPorMesa  int `json:"min_pessoas_por_mesa"`
	MaxPessoasPorMesa  int `json:"max_pessoas_por_mesa"`
	AvaliadoresPorMesa int `json:"avaliadores_por_mesa"`
}

func parametrosAlocacaoPadrao() ParametrosAlocacao {
	return ParametrosAlocacao{
		MesasPorHorario:    MESAS_POR_HORARIO,
		MinPessoasPorMesa:  MIN_PESSOAS_POR_MESA,
		MaxPessoasPorMesa:  MAX_PESSOAS_POR_MESA,
		AvaliadoresPorMesa: AVALIADORES_POR_MESA,
	}
}

// validar confere se os parâmetros fazem sentido e estão dentro dos limites.
func (pa ParametrosAlocacao) validar() error {
	switch {
	case pa.MesasPorHorario < 1 || pa.MesasPorHorario > LIMITE_MESAS_POR_HORARIO:
		return fmt.Errorf("mesas por horário deve estar entre 1 e %d", LIMITE_MESAS_POR_HORARIO)
	case pa.MinPessoasPorMesa < 1 || pa.MinPessoasPorMesa > LIMITE_PESSOAS_POR_MESA:
		return fmt.Errorf("mínimo de candidatos por mesa deve estar entre 1 e %d", LIMITE_PESSOAS_POR_MESA)
	case pa.MaxPessoasPorMesa < 1 || pa.MaxPessoasPorMesa > LIMITE_PESSOAS_POR_MESA:
		return fmt.Errorf("máximo de candidatos por mesa deve estar entre 1 e %d", LIMITE_PESSOAS_POR_MESA)
	case pa.MinPessoasPorMesa > pa.MaxPessoasPorMesa:
		return fmt.Errorf("o mínimo de candidatos por mesa (%d) não pode ser maior que o máximo (%d)", pa.MinPessoasPorMesa, pa.MaxPessoasPorMesa)
	case pa.AvaliadoresPorMesa < 1 || pa.AvaliadoresPorMesa > LIMITE_AVALIADORES_POR_MESA:
		return fmt.Errorf("avaliadores por mesa deve estar entre 1 e %d", LIMITE_AVALIADORES_POR_MESA)
	}
	return nil
}

// problema é a instância com tudo indexado de 0..n-1, para a busca ser rápida.
type problema struct {
	nCand, nAval, nSlot, nMesas int // nMesas = mesas por horário que dá para formar
	minPessoas, maxPessoas      int // candidatos por mesa
	avPorMesa                   int
	candID, avalID, slotID      []int
	slotDesc                    []string
	prefs                       [][]int // candidato → horários em ordem de preferência
	custoPref                   []int   // [cand*nSlot+slot] custo da opção; -1 se não escolhido
	softDe                      []int   // [cand*nAval+aval] custo "prefiro não" (0 se não há)
	proibDe                     []int   // [cand*nAval+aval] 1 se "não posso"
	par                         parametrosSA
}

// parametrosSA reúne os parâmetros da busca (padrão: constantes SA_*).
type parametrosSA struct {
	tempIni, tempFim   float64
	faltaIni, faltaFim float64
	proibIni, proibFim float64
	fracaoRampa        float64
	pctMover           int // % dos movimentos que movem um candidato; o resto troca dois
}

var parametrosPadrao = parametrosSA{
	tempIni: SA_TEMP_INICIAL, tempFim: SA_TEMP_FINAL,
	faltaIni: SA_PESO_FALTA_INICIAL, faltaFim: SA_PESO_FALTA_FINAL,
	proibIni: SA_PESO_PROIBIDO_INICIAL, proibFim: SA_PESO_PROIBIDO_FINAL,
	fracaoRampa: SA_FRACAO_RAMPA,
	pctMover:    SA_PCT_MOVER,
}

// montarProblema indexa a instância. Os parâmetros já devem ter sido validados.
func montarProblema(param ParametrosAlocacao, horarios map[int]*Horario, avals []*Avaliador, prefs map[int][]int, hard, soft map[int]map[int]bool) *problema {
	p := &problema{
		par:        parametrosPadrao,
		minPessoas: param.MinPessoasPorMesa,
		maxPessoas: param.MaxPessoasPorMesa,
		avPorMesa:  param.AvaliadoresPorMesa,
	}

	// ids ordenados: a ordem de iteração de maps em Go é aleatória
	for id := range horarios {
		p.slotID = append(p.slotID, id)
	}
	sort.Ints(p.slotID)
	slotIdx := make(map[int]int, len(p.slotID))
	for i, id := range p.slotID {
		slotIdx[id] = i
		p.slotDesc = append(p.slotDesc, horarios[id].Descricao)
	}
	for pid := range prefs {
		p.candID = append(p.candID, pid)
	}
	sort.Ints(p.candID)
	for _, a := range avals {
		p.avalID = append(p.avalID, a.ID)
	}
	sort.Ints(p.avalID)

	p.nCand, p.nAval, p.nSlot = len(p.candID), len(p.avalID), len(p.slotID)
	// cada avaliador só pode estar em uma mesa por horário
	p.nMesas = min(param.MesasPorHorario, p.nAval/p.avPorMesa)

	p.prefs = make([][]int, p.nCand)
	p.custoPref = make([]int, p.nCand*p.nSlot)
	for i := range p.custoPref {
		p.custoPref[i] = -1
	}
	for c, pid := range p.candID {
		for nivel, hid := range prefs[pid] {
			s := slotIdx[hid]
			if p.custoPref[c*p.nSlot+s] >= 0 {
				continue // horário repetido: vale a melhor opção
			}
			p.custoPref[c*p.nSlot+s] = -pontosOpcao(nivel)
			p.prefs[c] = append(p.prefs[c], s)
		}
	}

	p.softDe = make([]int, p.nCand*p.nAval)
	p.proibDe = make([]int, p.nCand*p.nAval)
	for c, pid := range p.candID {
		for a, aid := range p.avalID {
			if hard[aid][pid] {
				p.proibDe[c*p.nAval+a] = 1
			} else if soft[aid][pid] {
				p.softDe[c*p.nAval+a] = custoSoft
			}
		}
	}
	return p
}

// hungaro resolve a atribuição de custo mínimo de n linhas a colunas
// distintas de m ≥ n (algoritmo húngaro com potenciais, O(n²·m)).
// Os buffers são reaproveitados entre chamadas.
type hungaro struct {
	custo         []int // [linha*m+coluna], a partir de 0
	u, v, minv, p []int
	way           []int
	usado         []bool
}

func novoHungaro(maxLinhas, maxColunas int) hungaro {
	return hungaro{
		custo: make([]int, maxLinhas*maxColunas),
		u:     make([]int, maxLinhas+1),
		v:     make([]int, maxColunas+1),
		minv:  make([]int, maxColunas+1),
		p:     make([]int, maxColunas+1),
		way:   make([]int, maxColunas+1),
		usado: make([]bool, maxColunas+1),
	}
}

// resolver devolve o custo mínimo; depois, h.p[j] (j = 1..m) é a linha
// (a partir de 1) atribuída à coluna j-1, ou 0 se a coluna ficou livre.
//
// Depois de processar i linhas, -v[0] é o custo ótimo só dessas linhas, que
// nunca diminui. Se passar de corte, para e devolve completo = false (o
// custo devolvido é então só um limite inferior).
func (h *hungaro) resolver(n, m, corte int) (custo int, completo bool) {
	const inf = math.MaxInt / 4
	u, v, p, way, minv, usado := h.u[:n+1], h.v[:m+1], h.p[:m+1], h.way[:m+1], h.minv[:m+1], h.usado[:m+1]
	clear(u)
	clear(v)
	clear(p)
	for i := 1; i <= n; i++ {
		p[0] = i
		j0 := 0
		for j := range minv {
			minv[j], usado[j] = inf, false
		}
		for {
			usado[j0] = true
			i0, delta, j1 := p[j0], inf, 0
			linha := h.custo[(i0-1)*m : i0*m]
			for j := 1; j <= m; j++ {
				if usado[j] {
					continue
				}
				if cur := linha[j-1] - u[i0] - v[j]; cur < minv[j] {
					minv[j], way[j] = cur, j0
				}
				if minv[j] < delta {
					delta, j1 = minv[j], j
				}
			}
			for j := 0; j <= m; j++ {
				if usado[j] {
					u[p[j]] += delta
					v[j] -= delta
				} else {
					minv[j] -= delta
				}
			}
			j0 = j1
			if p[j0] == 0 {
				break
			}
		}
		for j0 != 0 {
			j1 := way[j0]
			p[j0] = p[j1]
			j0 = j1
		}
		if -v[0] > corte && i < n {
			return -v[0], false
		}
	}
	return -v[0], true
}

// estado é uma divisão dos candidatos em mesas (mesa m = horário*nMesas + k)
// com os melhores avaliadores para ela. Guarda dois custos: o de busca (com as
// penalidades da fase atual), que decide os movimentos, e o real
// (= -pontuação), que decide a melhor solução.
type estado struct {
	p                    *problema
	rng                  *rand.Rand
	mesaDe               []int   // candidato → mesa (-1 = sem mesa)
	membros              [][]int // mesa → candidatos
	somaSoft, somaProib  []int   // [mesa*nAval+aval] conflitos do avaliador com os membros
	avs                  [][]int // mesa → avaliadores (só para mesas não vazias)
	baseSlot, confSlot   []int   // custo de busca do horário = base + conflitos
	realSlot             []int
	busca, real          int
	pesoFalta, pesoProib int
	topConf              []int // candidato → soma dos avPorMesa maiores conflitos dele (pesos atuais)
	hung                 hungaro
	tmp                  [2][][]int // avaliadores calculados, ainda não aceitos
	mesaDaLinha          []int      // buffer de calcSlot: linha do húngaro → mesa
	maiores              []int      // buffer de definirPesos
}

type solucao struct {
	mesaDe []int
	avs    [][]int
	custo  int
}

// baseDoSlot é a parte do custo de busca do horário s que não depende dos
// avaliadores: preferências e vagas faltando em mesas incompletas.
func (e *estado) baseDoSlot(s int) int {
	p := e.p
	base := 0
	for k := 0; k < p.nMesas; k++ {
		membros := e.membros[s*p.nMesas+k]
		for _, c := range membros {
			base += p.custoPref[c*p.nSlot+s]
		}
		if n := len(membros); n > 0 && n < p.minPessoas {
			base += (p.minPessoas - n) * e.pesoFalta
		}
	}
	return base
}

// calcSlot escolhe os melhores avaliadores para as mesas do horário s
// (gravando-os em dst[k]) e devolve o custo de conflitos da busca e o custo
// real do horário. No custo real, mesa com menos de minPessoas será
// desfeita, então seus membros contam como não alocados. Se o custo de
// conflitos passar de corte, desiste no meio e devolve ok = false.
func (e *estado) calcSlot(s int, dst [][]int, corte int) (conf, real int, ok bool) {
	p := e.p
	A := p.nAval
	mesaDaLinha := e.mesaDaLinha
	linhas := 0
	for k := 0; k < p.nMesas; k++ {
		m := s*p.nMesas + k
		dst[k] = dst[k][:0]
		if len(e.membros[m]) == 0 {
			continue
		}
		// uma linha por vaga de avaliador da mesa
		linha := e.hung.custo[linhas*A : (linhas+1)*A]
		for a := range linha {
			linha[a] = e.somaSoft[m*A+a] + e.pesoProib*e.somaProib[m*A+a]
		}
		mesaDaLinha[linhas] = k
		for r := 1; r < p.avPorMesa; r++ {
			copy(e.hung.custo[(linhas+r)*A:(linhas+r+1)*A], linha)
			mesaDaLinha[linhas+r] = k
		}
		linhas += p.avPorMesa
	}
	if linhas == 0 {
		return 0, 0, true
	}
	conf, ok = e.hung.resolver(linhas, A, corte)
	if !ok {
		return conf, 0, false
	}
	for j := 1; j <= A; j++ {
		if i := e.hung.p[j]; i > 0 {
			k := mesaDaLinha[i-1]
			dst[k] = append(dst[k], j-1)
		}
	}

	for k := 0; k < p.nMesas; k++ {
		m := s*p.nMesas + k
		n := len(e.membros[m])
		switch {
		case n == 0:
		case n < p.minPessoas:
			real += n * custoNaoAlocado
		default:
			for _, c := range e.membros[m] {
				real += p.custoPref[c*p.nSlot+s]
			}
			for _, a := range dst[k] {
				real += e.somaSoft[m*A+a] + custoProibido*e.somaProib[m*A+a]
			}
		}
	}
	return conf, real, true
}

func (e *estado) guardarAvs(s int, src [][]int) {
	for k := 0; k < e.p.nMesas; k++ {
		m := s*e.p.nMesas + k
		e.avs[m] = append(e.avs[m][:0], src[k]...)
	}
}

// definirPesos ajusta as penalidades da busca para a fração progresso (0..1)
// da execução e recalcula tudo.
func (e *estado) definirPesos(progresso float64) {
	par := &e.p.par
	f := min(progresso/par.fracaoRampa, 1)
	rampa := func(ini, fim float64) int { return int(ini * math.Pow(fim/ini, f)) }
	e.pesoFalta = rampa(par.faltaIni, par.faltaFim)
	e.pesoProib = rampa(par.proibIni, par.proibFim)

	A := e.p.nAval
	maiores := e.maiores
	for c := range e.topConf {
		clear(maiores)
		for a := 0; a < A; a++ {
			w := e.p.softDe[c*A+a] + e.pesoProib*e.p.proibDe[c*A+a]
			for i := range maiores { // insere mantendo em ordem decrescente
				if w > maiores[i] {
					copy(maiores[i+1:], maiores[i:len(maiores)-1])
					maiores[i] = w
					break
				}
			}
		}
		e.topConf[c] = 0
		for _, w := range maiores {
			e.topConf[c] += w
		}
	}

	e.busca, e.real = 0, 0
	for s := 0; s < e.p.nSlot; s++ {
		e.baseSlot[s] = e.baseDoSlot(s)
		e.confSlot[s], e.realSlot[s], _ = e.calcSlot(s, e.tmp[0], math.MaxInt)
		e.guardarAvs(s, e.tmp[0])
		e.busca += e.baseSlot[s] + e.confSlot[s]
		e.real += e.realSlot[s]
	}
	for _, m := range e.mesaDe {
		if m < 0 {
			e.busca += custoNaoAlocado
			e.real += custoNaoAlocado
		}
	}
}

// mover tira c da mesa de (se >= 0) e o põe na mesa para (se >= 0).
func (e *estado) mover(c, de, para int) {
	A := e.p.nAval
	soft, proib := e.p.softDe[c*A:(c+1)*A], e.p.proibDe[c*A:(c+1)*A]
	if de >= 0 {
		lst := e.membros[de]
		for i, x := range lst {
			if x == c {
				lst[i] = lst[len(lst)-1]
				e.membros[de] = lst[:len(lst)-1]
				break
			}
		}
		ws, wp := e.somaSoft[de*A:(de+1)*A], e.somaProib[de*A:(de+1)*A]
		for a := range soft {
			ws[a] -= soft[a]
			wp[a] -= proib[a]
		}
	}
	if para >= 0 {
		e.membros[para] = append(e.membros[para], c)
		ws, wp := e.somaSoft[para*A:(para+1)*A], e.somaProib[para*A:(para+1)*A]
		for a := range soft {
			ws[a] += soft[a]
			wp[a] += proib[a]
		}
	}
	e.mesaDe[c] = para
}

// aceitar avalia um movimento já aplicado entre as mesas m1 e m2 (-1 = sem
// mesa), em que sai1 saiu de m1 e sai2 saiu de m2 (-1 = ninguém) e o custo
// fora das mesas mudou em deltaFora. Pelo critério de Metropolis decide se o
// movimento fica; se devolver false, o chamador desfaz.
//
// O sorteio vem antes do cálculo, e o húngaro só roda enquanto o movimento
// ainda pode passar. Para isso cada horário tem um limite inferior do novo
// custo de conflitos: pôr alguém numa mesa nunca o reduz, e tirar alguém
// reduz no máximo a soma dos avPorMesa maiores conflitos dele (topConf). Se uma mesa
// ficou vazia, ela perde suas linhas no húngaro e o limite cai para 0.
func (e *estado) aceitar(m1, m2, sai1, sai2, deltaFora int, temp float64) bool {
	limiar := math.MaxInt / 4
	if u := e.rng.Float64(); u > 0 {
		limiar = int(math.Floor(min(-temp*math.Log(u), float64(limiar))))
	}

	var slots, base, confLB [2]int
	n := 0
	for _, m := range [2]int{m1, m2} {
		if m >= 0 && (n == 0 || slots[0] != m/e.p.nMesas) {
			slots[n] = m / e.p.nMesas
			n++
		}
	}
	deltaLB := deltaFora
	for i := 0; i < n; i++ {
		s := slots[i]
		base[i] = e.baseDoSlot(s)
		confLB[i] = e.confSlot[s]
		for _, x := range [2][2]int{{m1, sai1}, {m2, sai2}} {
			if m, c := x[0], x[1]; m >= 0 && c >= 0 && m/e.p.nMesas == s {
				confLB[i] -= e.topConf[c]
				if len(e.membros[m]) == 0 {
					confLB[i] = math.MinInt / 4
				}
			}
		}
		confLB[i] = max(confLB[i], 0)
		deltaLB += base[i] - e.baseSlot[s] + confLB[i] - e.confSlot[s]
	}
	if deltaLB > limiar {
		return false
	}

	var conf, real [2]int
	for i := 0; i < n; i++ {
		// folga que ainda sobra para o custo de conflitos deste horário
		corte := confLB[i] + limiar - deltaLB
		c, r, ok := e.calcSlot(slots[i], e.tmp[i], corte)
		if !ok || c > corte {
			return false
		}
		conf[i], real[i] = c, r
		deltaLB += c - confLB[i]
	}
	// aqui deltaLB já é o delta exato
	deltaReal := deltaFora
	for i := 0; i < n; i++ {
		s := slots[i]
		deltaReal += real[i] - e.realSlot[s]
		e.baseSlot[s], e.confSlot[s], e.realSlot[s] = base[i], conf[i], real[i]
		e.guardarAvs(s, e.tmp[i])
	}
	e.busca += deltaLB
	e.real += deltaReal
	return true
}

// passo sorteia e testa um movimento.
func (e *estado) passo(temp float64) {
	p, rng := e.p, e.rng
	c1 := rng.Intn(p.nCand)
	if len(p.prefs[c1]) == 0 {
		return
	}
	m1 := e.mesaDe[c1]

	if rng.Intn(100) < p.par.pctMover { // mover candidato
		dest := -1
		if rng.Intn(20) != 0 {
			s := p.prefs[c1][rng.Intn(len(p.prefs[c1]))]
			dest = s*p.nMesas + rng.Intn(p.nMesas)
		}
		if dest == m1 || (dest >= 0 && len(e.membros[dest]) >= p.maxPessoas) {
			return
		}
		fora := 0
		if m1 < 0 {
			fora -= custoNaoAlocado
		}
		if dest < 0 {
			fora += custoNaoAlocado
		}
		e.mover(c1, m1, dest)
		if !e.aceitar(m1, dest, c1, -1, fora, temp) {
			e.mover(c1, dest, m1)
		}
		return
	}

	// trocar c1 com um candidato de uma mesa de um horário que c1 escolheu
	s := p.prefs[c1][rng.Intn(len(p.prefs[c1]))]
	m2 := s*p.nMesas + rng.Intn(p.nMesas)
	if m2 == m1 || len(e.membros[m2]) == 0 {
		return
	}
	c2 := e.membros[m2][rng.Intn(len(e.membros[m2]))]
	if m1 >= 0 && p.custoPref[c2*p.nSlot+m1/p.nMesas] < 0 {
		return // c2 não escolheu o horário da mesa de c1
	}
	e.mover(c1, m1, m2)
	e.mover(c2, m2, m1)
	if !e.aceitar(m1, m2, c1, c2, 0, temp) {
		e.mover(c2, m1, m2)
		e.mover(c1, m2, m1)
	}
}

// novoEstado aloca os candidatos de forma gulosa (quem tem menos opções
// primeiro, na primeira mesa com vaga) e escolhe os melhores avaliadores.
func (p *problema) novoEstado(semente int64) *estado {
	nM := p.nSlot * p.nMesas
	e := &estado{
		p:         p,
		rng:       rand.New(rand.NewSource(semente)),
		mesaDe:    make([]int, p.nCand),
		membros:   make([][]int, nM),
		somaSoft:  make([]int, nM*p.nAval),
		somaProib: make([]int, nM*p.nAval),
		avs:       make([][]int, nM),
		baseSlot:  make([]int, p.nSlot),
		confSlot:  make([]int, p.nSlot),
		realSlot:  make([]int, p.nSlot),
		topConf:   make([]int, p.nCand),
		hung:      novoHungaro(p.nMesas*p.avPorMesa, p.nAval),

		mesaDaLinha: make([]int, p.nMesas*p.avPorMesa),
		maiores:     make([]int, p.avPorMesa),
	}
	for m := range e.membros {
		e.membros[m] = make([]int, 0, p.maxPessoas+1)
		e.avs[m] = make([]int, 0, p.avPorMesa)
	}
	for i := range e.tmp {
		e.tmp[i] = make([][]int, p.nMesas)
		for k := range e.tmp[i] {
			e.tmp[i][k] = make([]int, 0, p.avPorMesa)
		}
	}

	for c := range e.mesaDe {
		e.mesaDe[c] = -1
	}
	ordem := e.rng.Perm(p.nCand)
	sort.SliceStable(ordem, func(i, j int) bool { return len(p.prefs[ordem[i]]) < len(p.prefs[ordem[j]]) })
	for _, c := range ordem {
	busca:
		for _, s := range p.prefs[c] {
			for k := 0; k < p.nMesas; k++ {
				if m := s*p.nMesas + k; len(e.membros[m]) < p.maxPessoas {
					e.mover(c, -1, m)
					break busca
				}
			}
		}
	}
	e.definirPesos(0)
	return e
}

func (e *estado) salvar(dst *solucao) {
	dst.mesaDe = append(dst.mesaDe[:0], e.mesaDe...)
	if dst.avs == nil {
		dst.avs = make([][]int, len(e.avs))
	}
	for m := range e.avs {
		dst.avs[m] = append(dst.avs[m][:0], e.avs[m]...)
	}
	dst.custo = e.real
}

// otimizar executa um simulated annealing completo e devolve a melhor solução
// vista. Para antes do fim se alguma execução atingir o custo alvo.
func (p *problema) otimizar(semente int64, iteracoes int, alvo int, feitas, melhorGlobal *atomic.Int64, parar *atomic.Bool) solucao {
	const bloco = 1_000
	e := p.novoEstado(semente)
	var melhor solucao
	e.salvar(&melhor)
	par := &p.par
	temp := par.tempIni
	razao := par.tempFim / par.tempIni

	for it := 0; it < iteracoes && melhor.custo > alvo; it++ {
		if it%bloco == 0 {
			if it > 0 {
				feitas.Add(bloco)
				atualizarMinimo(melhorGlobal, int64(melhor.custo))
			}
			if parar.Load() {
				break
			}
			progresso := float64(it) / float64(iteracoes)
			temp = par.tempIni * math.Pow(razao, progresso)
			e.definirPesos(progresso)
		}
		e.passo(temp)
		if e.real < melhor.custo {
			e.salvar(&melhor)
		}
	}
	atualizarMinimo(melhorGlobal, int64(melhor.custo))
	if melhor.custo <= alvo {
		parar.Store(true)
	}
	return melhor
}

func atualizarMinimo(v *atomic.Int64, x int64) {
	for {
		atual := v.Load()
		if x >= atual || v.CompareAndSwap(atual, x) {
			return
		}
	}
}

// limiteInferior resolve exatamente só a escolha de horários, ignorando
// avaliadores e o mínimo por mesa. Nenhuma alocação tem custo menor que isso;
// se a busca chega nesse valor, a solução é comprovadamente ótima.
func (p *problema) limiteInferior() int {
	return p.fluxoHorarios(func(c, s int) int { return p.custoPref[c*p.nSlot+s] }, custoNaoAlocado)
}

// maxAlocaveis é quantos candidatos, no máximo, cabem nos horários que
// escolheram (mesmas simplificações de limiteInferior).
func (p *problema) maxAlocaveis() int {
	return p.nCand - p.fluxoHorarios(func(c, s int) int { return 0 }, 1)
}

// fluxoHorarios resolve por fluxo de custo mínimo a escolha de horários: cada
// candidato vai para um horário que escolheu (custo custoOpcao) ou fica sem
// mesa (custo custoSem), com até nMesas*maxPessoas candidatos por horário.
// Devolve o custo mínimo.
func (p *problema) fluxoHorarios(custoOpcao func(c, s int) int, custoSem int) int {
	type aresta struct{ para, cap, custo, rev int }
	origem, destino := 0, p.nCand+p.nSlot+1
	g := make([][]aresta, destino+1)
	ligar := func(u, v, cap, custo int) {
		g[u] = append(g[u], aresta{v, cap, custo, len(g[v])})
		g[v] = append(g[v], aresta{u, 0, -custo, len(g[u]) - 1})
	}
	for c := 0; c < p.nCand; c++ {
		ligar(origem, 1+c, 1, 0)
		ligar(1+c, destino, 1, custoSem)
		for _, s := range p.prefs[c] {
			ligar(1+c, 1+p.nCand+s, 1, custoOpcao(c, s))
		}
	}
	for s := 0; s < p.nSlot; s++ {
		ligar(1+p.nCand+s, destino, p.nMesas*p.maxPessoas, 0)
	}

	// caminhos mínimos sucessivos (Bellman-Ford), uma unidade por vez
	total := 0
	dist := make([]int, len(g))
	paiNo, paiAresta := make([]int, len(g)), make([]int, len(g))
	for f := 0; f < p.nCand; f++ {
		for i := range dist {
			dist[i] = math.MaxInt
		}
		dist[origem] = 0
		for mudou := true; mudou; {
			mudou = false
			for u := range g {
				if dist[u] == math.MaxInt {
					continue
				}
				for i, a := range g[u] {
					if a.cap > 0 && dist[u]+a.custo < dist[a.para] {
						dist[a.para] = dist[u] + a.custo
						paiNo[a.para], paiAresta[a.para] = u, i
						mudou = true
					}
				}
			}
		}
		for v := destino; v != origem; v = paiNo[v] {
			a := &g[paiNo[v]][paiAresta[v]]
			a.cap--
			g[v][a.rev].cap++
		}
		total += dist[destino]
	}
	return total
}

// montarMesas converte a solução em mesas com ids do banco. Só entram mesas
// com o mínimo de candidatos, renumeradas por horário (mesa 1, mesa 2...).
func (p *problema) montarMesas(sol solucao) (map[int]int, []*Mesa) {
	membros := make([][]int, len(sol.avs))
	for c, m := range sol.mesaDe {
		if m >= 0 {
			membros[m] = append(membros[m], c)
		}
	}
	aloc := make(map[int]int)
	var mesas []*Mesa
	for s := 0; s < p.nSlot; s++ {
		num := 0
		for k := 0; k < p.nMesas; k++ {
			m := s*p.nMesas + k
			if len(membros[m]) < p.minPessoas {
				continue
			}
			mesa := &Mesa{
				ID:        p.slotID[s]*100 + num,
				DiaID:     p.slotID[s],
				Descricao: fmt.Sprintf("%s - mesa %d", p.slotDesc[s], num+1),
			}
			num++
			for _, c := range membros[m] {
				mesa.Candidatos = append(mesa.Candidatos, p.candID[c])
				aloc[p.candID[c]] = mesa.ID
			}
			for _, a := range sol.avs[m] {
				mesa.Avaliadores = append(mesa.Avaliadores, p.avalID[a])
			}
			mesas = append(mesas, mesa)
		}
	}
	return aloc, mesas
}

// fazerMelhorAlocacaoMesas roda SA_EXECUCOES buscas em paralelo e devolve a
// melhor alocação. Os parâmetros já devem ter sido validados. onProgress
// recebe (iteraçõesFeitas, totalIterações, melhorScore) algumas vezes por
// segundo.
func fazerMelhorAlocacaoMesas(param ParametrosAlocacao, horarios map[int]*Horario, avals []*Avaliador, prefs map[int][]int, hard, soft map[int]map[int]bool, onProgress func(int, int, int)) (ResultadoAlocacao, []*Mesa) {
	p := montarProblema(param, horarios, avals, prefs, hard, soft)
	if p.nMesas == 0 {
		log.Printf("[WARN] Avaliadores insuficientes para formar qualquer mesa (necessário mínimo: %d)", p.avPorMesa)
		return ResultadoAlocacao{Alocacao: map[int]int{}, Pontuacao: SCORE_BASE - p.nCand*custoNaoAlocado}, nil
	}
	if p.nCand == 0 {
		return ResultadoAlocacao{Alocacao: map[int]int{}, Pontuacao: SCORE_BASE}, nil
	}

	inicio := time.Now()
	alvo := p.limiteInferior()
	fmt.Printf("INICIANDO ALOCAÇÃO: %d candidatos, %d avaliadores, %d horários × %d mesas (%d a %d candidatos, %d avaliadores cada) | %d execuções × %d iterações | limite inferior do custo: %d\n",
		p.nCand, p.nAval, p.nSlot, p.nMesas, p.minPessoas, p.maxPessoas, p.avPorMesa, SA_EXECUCOES, SA_ITERACOES, alvo)

	var feitas, melhorGlobal atomic.Int64
	var parar atomic.Bool
	melhorGlobal.Store(math.MaxInt64)
	total := SA_EXECUCOES * SA_ITERACOES

	resultados := make([]solucao, SA_EXECUCOES)
	var wg sync.WaitGroup
	for r := range SA_EXECUCOES {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resultados[r] = p.otimizar(SA_SEMENTE+int64(r), SA_ITERACOES, alvo, &feitas, &melhorGlobal, &parar)
		}()
	}
	fim := make(chan struct{})
	go func() { wg.Wait(); close(fim) }()

	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
esperar:
	for {
		select {
		case <-fim:
			break esperar
		case <-ticker.C:
			if onProgress != nil {
				onProgress(int(feitas.Load()), total, SCORE_BASE-int(melhorGlobal.Load()))
			}
		}
	}

	melhor := resultados[0]
	for _, r := range resultados[1:] {
		if r.custo < melhor.custo {
			melhor = r
		}
	}

	aloc, mesas := p.montarMesas(melhor)
	res := ResultadoAlocacao{Alocacao: aloc, Alocados: len(aloc)}
	score, penalidades, _ := pontuarResultado(res, mesas, prefs, hard, soft)
	res.Pontuacao = score

	status := "melhor encontrada"
	if melhor.custo <= alvo {
		status = "ÓTIMA (atingiu o limite inferior)"
	}
	fmt.Printf("Finalizado em %v | score %d | solução %s | penalidades: %v\n",
		time.Since(inicio).Round(time.Millisecond), score, status, penalidades)
	return res, mesas
}

// ==================================================
// ================== PONTUAÇÃO =====================
// ==================================================

// pontuarResultado calcula a pontuação de uma alocação com base nos
// critérios definidos nas constantes PONTOS_OPCAO_*, PENALIDADE_SOFT e PENALIDADE_HARD.
// Retorna um valor inteiro — quanto maior, melhor o resultado.
func pontuarResultado(res ResultadoAlocacao, mesas []*Mesa, prefs map[int][]int, hard, soft map[int]map[int]bool) (int, map[string]int, int) {
	var MAP_PENALIDADES = map[string]int{
		"opcao_1":     0,
		"opcao_2":     0,
		"opcao_3":     0,
		"opcao_4":     0,
		"opcao_5":     0,
		"nao_alocado": 0,
		"prefiro_nao": 0,
		"nao_posso":   0,
	}
	PONTOS_TOMADOS := 0

	// índice mesaID -> Mesa para lookup rápido
	mesaIdx := make(map[int]*Mesa, len(mesas))
	for _, m := range mesas {
		mesaIdx[m.ID] = m
	}

	score := SCORE_BASE

	// Penalidade por candidatos não alocados (ausentes de res.Alocacao)
	for pid := range prefs {
		if _, alocado := res.Alocacao[pid]; !alocado {
			score += PENALIDADE_NAO_ALOCADO
			PONTOS_TOMADOS += PENALIDADE_NAO_ALOCADO
			MAP_PENALIDADES["nao_alocado"] += 1
		}
	}

	for pid, mid := range res.Alocacao {
		m := mesaIdx[mid]
		if m == nil {
			continue
		}

		// Pontos pela preferência de horário do candidato
		for nivel, hid := range prefs[pid] {
			if hid == m.DiaID {
				score += pontosOpcao(nivel)
				PONTOS_TOMADOS += pontosOpcao(nivel)
				MAP_PENALIDADES[fmt.Sprintf("opcao_%d", nivel+1)] += 1
				break
			}
		}

		// Penalidade por violação de restrições por avaliador
		for _, avID := range m.Avaliadores {
			if hard[avID][pid] {
				score += PENALIDADE_HARD
				MAP_PENALIDADES["nao_posso"] += 1
				PONTOS_TOMADOS += PENALIDADE_HARD
			} else if soft[avID][pid] {
				score += PENALIDADE_SOFT
				MAP_PENALIDADES["prefiro_nao"] += 1
				PONTOS_TOMADOS += PENALIDADE_SOFT
			}
		}
	}
	return score, MAP_PENALIDADES, PONTOS_TOMADOS
}

// ==================================================
// ============= IMPRESSÃO DOS RESULTADOS ===========
// ==================================================

func imprimirAlocacaoMesas(aloc map[int]int, mesas map[int]*Mesa, prefs map[int][]int) int {
	fmt.Println("\n---- ALOCAÇÃO FINAL ----")
	// alocados := make(map[int]bool)

	// total único de pessoas que possuem preferência registrada
	totalSet := make(map[int]bool)
	for pid := range prefs {
		totalSet[pid] = true
	}

	var nao []int
	for pid := range totalSet {
		if _, exists := aloc[pid]; !exists {
			nao = append(nao, pid)
		}
	}

	fmt.Printf("\n---- NÃO ALOCADOS (%d) ----\n%v\n", len(nao), nao)
	return len(totalSet)
}

func imprimirMesasPreenchidas(mesas []*Mesa, aloc map[int]int, total int) {
	fmt.Printf("\n---- MESAS PREENCHIDAS ----\nPessoas únicas com disponibilidade: %d\n\n", total)

	// Mapa de prioridade dos dias
	dias := map[string]int{
		"segunda": 1,
		"terca":   2,
		"quarta":  3,
		"quinta":  4,
		"sexta":   5,
	}

	// Função para extrair o dia e número da mesa
	getDiaEMesa := func(desc string) (int, int) {
		partes := strings.Split(desc, "-")
		if len(partes) < 2 {
			return 999, 999
		}
		dia := strings.TrimSpace(partes[0])
		numMesa := 999

		if strings.Contains(partes[1], "mesa") {
			p := strings.Split(strings.TrimSpace(partes[1]), " ")
			if len(p) >= 2 {
				n, err := strconv.Atoi(p[1])
				if err == nil {
					numMesa = n
				}
			}
		}
		prioridade, ok := dias[strings.ToLower(dia)]
		if !ok {
			prioridade = 999
		}
		return prioridade, numMesa
	}

	// Ordena as mesas por dia e número
	sort.Slice(mesas, func(i, j int) bool {
		dia1, mesa1 := getDiaEMesa(mesas[i].Descricao)
		dia2, mesa2 := getDiaEMesa(mesas[j].Descricao)
		if dia1 == dia2 {
			return mesa1 < mesa2
		}
		return dia1 < dia2
	})

	// Imprime as mesas
	for _, m := range mesas {
		if len(m.Candidatos) == 0 {
			continue
		}
		fmt.Printf("%s (%d candidatos) - %v | Avaliadores: %v\n",
			m.Descricao, len(m.Candidatos), m.Candidatos, m.Avaliadores)
	}
}
