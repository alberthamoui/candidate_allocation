package allocation

import (
	types "candidate_alocator/back/type"
	"database/sql"
	"errors"
	"fmt"
	"math/big"
	"math/rand"
	"sort"
	"strings"
	"time"
)

const (
	MESAS_POR_HORARIO        = 5
	MIN_PESSOAS_POR_MESA     = 5
	MAX_PESSOAS_POR_MESA     = 8
	MIN_AVALIADORES_POR_MESA = 5
	MAX_AVALIADORES_POR_MESA = 5
	MAX_TESTES               = 100_000
	MELHOR_CASO              = 50
)

func fatorialBig(n int) *big.Int {
	result := big.NewInt(1)
	for i := 2; i <= n; i++ {
		result.Mul(result, big.NewInt(int64(i)))
	}
	return result
}

func gerarMesas(horarios map[int]*types.Horario, avals []*types.Avaliador) ([]*types.Mesa, map[int][]*types.Mesa) {
	var todas []*types.Mesa
	porDia := make(map[int][]*types.Mesa)

	rand.Seed(time.Now().UnixNano())

	for _, h := range horarios {
		for i := 0; i < MESAS_POR_HORARIO; i++ {
			idMesa := h.ID*100 + i
			m := &types.Mesa{
				ID:        idMesa,
				DiaID:     h.ID,
				Descricao: fmt.Sprintf("%s – mesa %d", h.Descricao, i+1),
			}

			n := rand.Intn(MAX_AVALIADORES_POR_MESA-MIN_AVALIADORES_POR_MESA+1) + MIN_AVALIADORES_POR_MESA
			rand.Shuffle(len(avals), func(i, j int) { avals[i], avals[j] = avals[j], avals[i] })
			for k := 0; k < n; k++ {
				m.Avaliadores = append(m.Avaliadores, avals[k].ID)
			}

			todas = append(todas, m)
			porDia[h.ID] = append(porDia[h.ID], m)
		}
	}

	return todas, porDia
}

func filtrarHorariosValidos(horarios map[int]*types.Horario) []*types.Horario {
	var valid []*types.Horario
	for _, h := range horarios {
		if len(h.Candidatos) >= MIN_PESSOAS_POR_MESA {
			valid = append(valid, h)
		}
	}
	return valid
}

func sortHorariosPorCandidatos(hs []*types.Horario) {
	sort.SliceStable(hs, func(i, j int) bool {
		return len(hs[i].Candidatos) < len(hs[j].Candidatos)
	})
}

func podeAvaliar(avID, pid int, restr map[int]map[int]bool) bool {
	return !restr[avID][pid]
}

func podeAlocarNoHorario(h *types.Horario, pid int, restr map[int]map[int]bool) bool {
	for _, av := range h.Avaliadores {
		if !podeAvaliar(av, pid, restr) {
			return false
		}
	}
	return true
}

func fazerAlocacaoMesas(mesas []*types.Mesa, porDia map[int][]*types.Mesa, prefs map[int][]int, restr map[int]map[int]bool) types.ResultadoAlocacao {
	aloc := make(map[int]int)
	alocados := make(map[int]bool)
	pontuacao := 0
	ocupado := make(map[int]int)

	for nivel := 0; nivel < 4; nivel++ {
		for pid, pref := range prefs {
			if alocados[pid] || len(pref) <= nivel {
				continue
			}
			dia := pref[nivel]

			for _, m := range porDia[dia] {
				if ocupado[m.ID] >= MAX_PESSOAS_POR_MESA {
					continue
				}
				if !podeAlocarNoHorario(&types.Horario{Avaliadores: m.Avaliadores}, pid, restr) {
					continue
				}

				aloc[pid] = m.ID
				ocupado[m.ID]++
				m.Candidatos = append(m.Candidatos, pid)
				alocados[pid] = true
				pontuacao += nivel
				break
			}
		}
	}

	for _, m := range mesas {
		if ocupado[m.ID] > 0 && ocupado[m.ID] < MIN_PESSOAS_POR_MESA {
			for _, pid := range m.Candidatos {
				delete(aloc, pid)
				delete(alocados, pid)
			}
			m.Candidatos = nil
			ocupado[m.ID] = 0
		}
	}

	return types.ResultadoAlocacao{Alocacao: aloc, Pontuacao: pontuacao, Alocados: len(aloc)}
}

func Run(db *sql.DB) error {
	if db == nil {
		return errors.New("conexao do banco nao pode ser nil")
	}

	fmt.Println("---- INICIANDO ALOCAÇÃO ----")

	avals, err := carregarAvaliadores(db)
	if err != nil {
		return err
	}
	restr, err := carregarRestricoes(db)
	if err != nil {
		return err
	}
	horarios, err := carregarHorarios(db)
	if err != nil {
		return err
	}
	prefs, err := carregarDisponibilidades(db, horarios)
	if err != nil {
		return err
	}

	fmt.Println("---- DADOS CARREGADOS ----")

	mesas, porDia := gerarMesas(horarios, avals)
	fmt.Println("\n---- MESAS GERADAS ----")
	for _, m := range mesas {
		fmt.Printf("Mesa %d → %s | Avaliadores: %v\n", m.ID, m.Descricao, m.Avaliadores)
	}
	fmt.Println(strings.Repeat("-", 60))

	start := time.Now()
	res := fazerAlocacaoMesas(mesas, porDia, prefs, restr)

	mapMesa := make(map[int]*types.Mesa, len(mesas))
	for _, m := range mesas {
		mapMesa[m.ID] = m
	}

	total := imprimirAlocacaoMesas(res.Alocacao, mapMesa, prefs)
	imprimirMesasPreenchidas(mesas, total)

	fmt.Printf("\nTempo total de execução: %v\n", time.Since(start))
	return nil
}
