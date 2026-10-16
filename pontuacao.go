package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// ==================================================
// ================== PONTUAÇÃO =====================
// ==================================================

// pontuarResultado calcula a pontuação de uma alocação com base nos
// critérios definidos nas constantes PONTOS_OPCAO_*, PENALIDADE_SOFT e PENALIDADE_HARD.
// Retorna um valor inteiro — quanto maior, melhor o resultado. Com
// semPreferencia, o horário não dá nem tira pontos (preferencia.go).
func pontuarResultado(res ResultadoAlocacao, mesas []*Mesa, prefs map[int][]int, hard, soft map[int]map[int]bool, semPreferencia bool) (int, map[string]int, int) {
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
			if hid == m.DiaID && !semPreferencia {
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
