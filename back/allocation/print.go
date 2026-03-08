package allocation

import (
	types "candidate_alocator/back/type"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

func imprimirAlocacaoMesas(aloc map[int]int, mesas map[int]*types.Mesa, prefs map[int][]int) int {
	fmt.Println("\n---- ALOCAÇÃO FINAL ----")
	alocados := make(map[int]bool)
	for pid, mid := range aloc {
		fmt.Printf("Pessoa %d -> %s (Mesa %d)\n", pid, mesas[mid].Descricao, mid)
		alocados[pid] = true
	}

	totalSet := make(map[int]bool)
	for pid := range prefs {
		totalSet[pid] = true
	}

	var nao []int
	for pid := range totalSet {
		if !alocados[pid] {
			nao = append(nao, pid)
		}
	}

	fmt.Printf("\n---- NÃO ALOCADOS (%d) ----\n%v\n", len(nao), nao)
	return len(totalSet)
}

func imprimirMesasPreenchidas(mesas []*types.Mesa, total int) {
	fmt.Printf("\n---- MESAS PREENCHIDAS ----\nPessoas únicas com disponibilidade: %d\n\n", total)

	dias := map[string]int{
		"segunda": 1,
		"terca":   2,
		"quarta":  3,
		"quinta":  4,
		"sexta":   5,
	}

	getDiaEMesa := func(desc string) (int, int) {
		partes := strings.Split(desc, "–")
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

	sort.Slice(mesas, func(i, j int) bool {
		dia1, mesa1 := getDiaEMesa(mesas[i].Descricao)
		dia2, mesa2 := getDiaEMesa(mesas[j].Descricao)
		if dia1 == dia2 {
			return mesa1 < mesa2
		}
		return dia1 < dia2
	})

	for _, m := range mesas {
		if len(m.Candidatos) == 0 {
			continue
		}
		fmt.Printf("%s (%d candidatos) – %v | Avaliadores: %v\n",
			m.Descricao, len(m.Candidatos), m.Candidatos, m.Avaliadores)
	}
}
