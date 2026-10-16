package main

import (
	"math"
	"slices"
	"sort"
)

// ==================================================
// ============ REPARO DEPOIS DA BUSCA ==============
// ==================================================
//
// O simulated annealing move um candidato (ou troca dois) por vez. Algumas
// melhorias precisam de vários movimentos juntos, e cada um sozinho piora o
// custo antes de a mesa fechar:
//
//   - abrir uma mesa nova exige minPessoas candidatos de uma vez; quem ficou
//     sem mesa continua sem mesa se os outros que escolheram o horário estão
//     em mesas no mínimo (sair sozinho desfaria a mesa de origem);
//   - esvaziar uma mesa que está no mínimo exige tirar todos de uma vez (ex.:
//     todos preferem um horário que tem vagas, mas ninguém sai sozinho);
//   - levar alguém para um horário melhor pode exigir uma cadeia: ele sai de
//     uma mesa no mínimo, outro entra no lugar dele, e assim por diante.
//
// Nesses casos a busca para num ótimo local. O reparo tenta esses movimentos
// compostos sobre a melhor solução e fica com cada um que baixa o custo real,
// até não haver mais melhora; nunca devolve algo pior do que recebeu.

const reparoMaxRodadas = 50

// reparar aplica os movimentos compostos à solução enquanto o custo cair.
func (p *problema) reparar(sol solucao) solucao {
	e := p.estadoDe(sol)
	for range reparoMaxRodadas {
		melhorou := false
		for c := range p.nCand {
			if !e.temMesa(c) && e.abrirMesa(c) {
				melhorou = true
			}
		}
		for s := range p.nSlot {
			if e.abrirMesaEm(s, -1) {
				melhorou = true
			}
		}
		for m := range e.membros {
			if len(e.membros[m]) > 0 && e.fecharMesa(m) {
				melhorou = true
			}
		}
		for e.cadeia() {
			melhorou = true
		}
		if !melhorou {
			break
		}
	}
	if e.real >= sol.custo {
		return sol
	}
	var r solucao
	e.salvar(&r)
	return r
}

// estadoDe monta o estado de uma solução, com as penalidades finais.
func (p *problema) estadoDe(sol solucao) *estado {
	e := p.estadoVazio(0)
	for c, m := range sol.mesaDe {
		if m >= 0 {
			e.mover(c, -1, m)
		}
	}
	e.definirPesos(1)
	return e
}

// temMesa diz se c está numa mesa que vai ser formada (com o mínimo).
func (e *estado) temMesa(c int) bool {
	m := e.mesaDe[c]
	return m >= 0 && len(e.membros[m]) >= e.p.minPessoas
}

// abrirMesa tenta formar, em cada horário que x escolheu, uma mesa com x.
func (e *estado) abrirMesa(x int) bool {
	for _, s := range e.p.prefs[x] {
		if e.abrirMesaEm(s, x) {
			return true
		}
	}
	return false
}

// abrirMesaEm tenta formar uma mesa numa vaga de mesa vazia do horário s,
// com x (se x >= 0) e mais candidatos que escolheram o horário: primeiro os
// que estão sem mesa, depois os que podem sair da mesa deles sem deixá-la
// abaixo do mínimo, dos que mais ganham para os que menos perdem. Completa o
// mínimo e segue enquanto quem entra ganha com a troca, até o máximo.
func (e *estado) abrirMesaEm(s, x int) bool {
	p := e.p
	vazia := -1
	for k := 0; k < p.nMesas; k++ {
		if m := s*p.nMesas + k; len(e.membros[m]) == 0 {
			vazia = m
			break
		}
	}
	if vazia < 0 {
		return false
	}

	type candidato struct{ c, semMesa, perda int }
	var podem []candidato
	for c := range p.nCand {
		if c == x || p.custoPref[c*p.nSlot+s] < 0 {
			continue
		}
		perda, semMesa := p.custoPref[c*p.nSlot+s]-custoNaoAlocado, 0
		if e.temMesa(c) {
			semMesa = 1
			perda = p.custoPref[c*p.nSlot+s] - p.custoPref[c*p.nSlot+e.mesaDe[c]/p.nMesas]
		}
		podem = append(podem, candidato{c, semMesa, perda})
	}
	sort.SliceStable(podem, func(i, j int) bool {
		if podem[i].semMesa != podem[j].semMesa {
			return podem[i].semMesa < podem[j].semMesa
		}
		return podem[i].perda < podem[j].perda
	})

	var movs [][2]int
	if x >= 0 {
		movs = append(movs, [2]int{x, vazia})
	}
	saem := map[int]int{} // mesa de origem → quantos já saem dela
	for _, cd := range podem {
		if len(movs) == p.maxPessoas || (len(movs) >= p.minPessoas && cd.perda >= 0) {
			break
		}
		if m := e.mesaDe[cd.c]; e.temMesa(cd.c) {
			if len(e.membros[m])-saem[m] <= p.minPessoas {
				continue
			}
			saem[m]++
		}
		movs = append(movs, [2]int{cd.c, vazia})
	}
	return len(movs) >= p.minPessoas && e.tentar(movs)
}

// fecharMesa tenta desfazer a mesa m pondo cada membro na mesa formada com
// vaga mais barata entre os horários que ele escolheu.
func (e *estado) fecharMesa(m int) bool {
	p := e.p
	entram := map[int]int{} // mesa de destino → quantos já entram nela
	var movs [][2]int
	for _, c := range e.membros[m] {
		melhor, custoMelhor := -1, 0
		for _, s := range p.prefs[c] {
			for k := 0; k < p.nMesas; k++ {
				d := s*p.nMesas + k
				n := len(e.membros[d]) + entram[d]
				if d == m || n < p.minPessoas || n >= p.maxPessoas {
					continue
				}
				if custo := p.custoPref[c*p.nSlot+s]; melhor < 0 || custo < custoMelhor {
					melhor, custoMelhor = d, custo
				}
			}
		}
		if melhor < 0 {
			return false
		}
		entram[melhor]++
		movs = append(movs, [2]int{c, melhor})
	}
	return e.tentar(movs)
}

// cadeia procura a melhor sequência de movimentos em que uma vaga "anda" de
// mesa em mesa: alguém entra numa mesa formada com vaga, outro ocupa o lugar
// dele, e assim por diante, até a cadeia terminar numa mesa com gente acima
// do mínimo (que pode perder um) ou em alguém sem mesa. As mesas do meio
// ficam com o mesmo tamanho. É um caminho mínimo (Bellman-Ford) entre as
// mesas em que o peso de cada passo é a mudança no custo de preferência de
// quem se move; os avaliadores são conferidos depois, em tentar.
func (e *estado) cadeia() bool {
	p := e.p
	// nós: mesas formadas, mais um nó para "sem mesa" (pool)
	no := make([]int, len(e.membros)) // mesa → nó (-1 se não formada)
	var mesas []int
	for m := range e.membros {
		no[m] = -1
		if len(e.membros[m]) >= p.minPessoas {
			no[m] = len(mesas)
			mesas = append(mesas, m)
		}
	}
	pool := len(mesas)
	V := pool + 1
	noDe := func(c int) int {
		if e.temMesa(c) {
			return no[e.mesaDe[c]]
		}
		return pool
	}
	custoAtual := func(c int) int {
		if e.temMesa(c) {
			return p.custoPref[c*p.nSlot+e.mesaDe[c]/p.nMesas]
		}
		return custoNaoAlocado
	}

	// passo[x][y]: o candidato de y que mais ganha (ou menos perde) entrando em x
	const sem = math.MaxInt / 4
	peso := make([]int, V*V)
	quem := make([]int, V*V)
	for i := range peso {
		peso[i] = sem
	}
	for c := range p.nCand {
		y := noDe(c)
		for _, s := range p.prefs[c] {
			for k := 0; k < p.nMesas; k++ {
				x := no[s*p.nMesas+k]
				if x < 0 || x == y {
					continue
				}
				if w := p.custoPref[c*p.nSlot+s] - custoAtual(c); w < peso[x*V+y] {
					peso[x*V+y], quem[x*V+y] = w, c
				}
			}
		}
	}

	// começa nas mesas com vaga; cada passo leva a vaga para a mesa de quem entrou
	dist, pred := make([]int, V), make([]int, V)
	for x := range V {
		dist[x], pred[x] = sem, -1
		if x < pool && len(e.membros[mesas[x]]) < p.maxPessoas {
			dist[x] = 0
		}
	}
	for range V {
		mudou := false
		for x := range V {
			if dist[x] == sem || x == pool {
				continue
			}
			for y := range V {
				if w := peso[x*V+y]; w != sem && dist[x]+w < dist[y] {
					dist[y], pred[y] = dist[x]+w, x
					mudou = true
				}
			}
		}
		if !mudou {
			break
		}
	}

	// termina onde dá para tirar alguém: mesa acima do mínimo ou "sem mesa"
	fim, melhor := -1, 0
	for y := range V {
		pode := y == pool || len(e.membros[mesas[y]]) > p.minPessoas
		if pode && pred[y] >= 0 && dist[y] < melhor {
			fim, melhor = y, dist[y]
		}
	}
	if fim < 0 {
		return false
	}
	var movs [][2]int
	visto := map[int]bool{}
	for y := fim; pred[y] >= 0; y = pred[y] {
		if visto[y] {
			return false // ciclo: o caminho não é simples
		}
		visto[y] = true
		x := pred[y]
		movs = append(movs, [2]int{quem[x*V+y], mesas[x]})
	}
	return e.tentar(movs)
}

// tentar aplica os movimentos (candidato → mesa, -1 = sem mesa), recalcula os
// horários afetados (com os melhores avaliadores) e fica com eles só se o
// custo real cair; senão desfaz tudo.
func (e *estado) tentar(movs [][2]int) bool {
	p := e.p
	type volta struct{ c, de int }
	var desfazer []volta
	var slots []int
	fora := 0
	for _, mv := range movs {
		c, para := mv[0], mv[1]
		de := e.mesaDe[c]
		if de == para {
			continue
		}
		for _, m := range [2]int{de, para} {
			if s := m / p.nMesas; m >= 0 && !slices.Contains(slots, s) {
				slots = append(slots, s)
			}
		}
		if de < 0 {
			fora -= custoNaoAlocado
		}
		if para < 0 {
			fora += custoNaoAlocado
		}
		e.mover(c, de, para)
		desfazer = append(desfazer, volta{c, de})
	}

	type novoSlot struct {
		s, base, conf, real int
		avs                 [][]int
	}
	novos := make([]novoSlot, 0, len(slots))
	delta := fora
	for _, s := range slots {
		avs := make([][]int, p.nMesas)
		conf, real, _ := e.calcSlot(s, avs, math.MaxInt)
		novos = append(novos, novoSlot{s, e.baseDoSlot(s), conf, real, avs})
		delta += real - e.realSlot[s]
	}

	if delta >= 0 {
		for i := len(desfazer) - 1; i >= 0; i-- {
			e.mover(desfazer[i].c, e.mesaDe[desfazer[i].c], desfazer[i].de)
		}
		return false
	}
	e.busca += fora
	for _, n := range novos {
		e.busca += n.base - e.baseSlot[n.s] + n.conf - e.confSlot[n.s]
		e.baseSlot[n.s], e.confSlot[n.s], e.realSlot[n.s] = n.base, n.conf, n.real
		e.guardarAvs(n.s, n.avs)
	}
	e.real += delta
	return true
}
