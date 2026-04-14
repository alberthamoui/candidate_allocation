package main

// ==================================================
// ============== IMPORTS E CONSTANTES ==============
// ==================================================

import (
	"database/sql"
	"fmt"
	"log"
	"math/big"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
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

// ==================================================
// ==================== STRUCTS =====================
// ==================================================

type Mesa struct {
	ID          int    // único (ex.: 301 = quarta-mesa1)
	DiaID       int    // 1=segunda, 2=terça, ...
	Descricao   string // "quarta – mesa 2"
	Candidatos  []int  // já alocados
	Avaliadores []int
}

type ResultadoAlocacao struct {
	Alocacao  map[int]int
	Pontuacao int
	Alocados  int
}

type Avaliador struct {
	ID    int    `json:"id"`
	Nome  string `json:"nome"`
	Email string `json:"email"`
}

type Horario struct {
	ID          int
	Descricao   string
	Candidatos  []int
	Avaliadores []int
}

type horarioInfo struct {
	H       *Horario
	Pessoas []int
}

func fatorialBig(n int) *big.Int {
	result := big.NewInt(1)
	for i := 2; i <= n; i++ {
		result.Mul(result, big.NewInt(int64(i)))
	}
	return result
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

func carregarRestricoes(db *sql.DB) map[int]map[int]bool {
	rows, err := db.Query(`SELECT candidato_id, naoPosso FROM restricoes`)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	restr := make(map[int]map[int]bool)
	for rows.Next() {
		var cid int
		var raw sql.NullString
		if err := rows.Scan(&cid, &raw); err != nil {
			log.Fatal(err)
		}
		if !raw.Valid || strings.TrimSpace(raw.String) == "" {
			continue
		}
		for _, sig := range strings.Split(raw.String, ",") {
			sig = strings.TrimSpace(strings.TrimPrefix(sig, "A"))
			aid, err := strconv.Atoi(sig)
			if err != nil {
				continue
			}
			if restr[aid] == nil {
				restr[aid] = make(map[int]bool)
			}
			restr[aid][cid] = true
		}
	}
	return restr
}

func gerarMesas(hmap map[int]*Horario, avals []*Avaliador) ([]*Mesa, map[int][]*Mesa) {
	var todas []*Mesa
	porDia := make(map[int][]*Mesa)

	for _, h := range hmap {
		// Embaralha avaliadores uma vez por horário
		perm := make([]*Avaliador, len(avals))
		copy(perm, avals)
		rand.Shuffle(len(perm), func(i, j int) { perm[i], perm[j] = perm[j], perm[i] })

		// Distribui avaliadores sequencialmente (sem repetição por dia)
		// Cada avaliador aparece em no máximo uma mesa por horário.
		idx := 0

		for i := 0; i < MESAS_POR_HORARIO; i++ {
			m := &Mesa{
				ID:        h.ID*100 + i,
				DiaID:     h.ID,
				Descricao: fmt.Sprintf("%s – mesa %d", h.Descricao, i+1),
			}

			for k := 0; k < MIN_AVALIADORES_POR_MESA && idx < len(perm); k++ {
				m.Avaliadores = append(m.Avaliadores, perm[idx].ID)
				idx++
			}

			todas = append(todas, m)
			porDia[h.ID] = append(porDia[h.ID], m)
		}
	}
	return todas, porDia
}

// ==================================================
// ========= PRÉ-PROCESSAMENTO DE HORÁRIOS =========
// ==================================================

func filtrarHorariosValidos(horarios map[int]*Horario) []*Horario {
	var valid []*Horario
	for _, h := range horarios {
		if len(h.Candidatos) >= MIN_PESSOAS_POR_MESA {
			valid = append(valid, h)
		}
	}
	return valid
}

func sortHorariosPorCandidatos(hs []*Horario) {
	sort.SliceStable(hs, func(i, j int) bool {
		return len(hs[i].Candidatos) < len(hs[j].Candidatos)
	})
}

// ==================================================
// ============== ALOCAÇÃO DE PESSOAS ===============
// ==================================================

func conflitante(avIDs []int, pid int, hard, soft map[int]map[int]bool) (hardBlock bool, softTouch bool) {
	for _, av := range avIDs {
		if hard[av][pid] { return true, true }
		if soft[av][pid] { softTouch = true }
	}
	return false, softTouch
}

func podeAvaliar(avID, pid int, restr map[int]map[int]bool) bool {
	return !restr[avID][pid]
}

func podeAlocarNoHorario(h *Horario, pid int, restr map[int]map[int]bool) bool {
	for _, av := range h.Avaliadores {
		if !podeAvaliar(av, pid, restr) {
			return false
		}
	}
	return true
}

func fazerAlocacaoMesas(
	mesas []*Mesa, porDia map[int][]*Mesa,
	prefs map[int][]int,
	hard, soft map[int]map[int]bool) ResultadoAlocacao {

	aloc := make(map[int]int)
	ocupado := make(map[int]int)
	alocados := make(map[int]bool)
	pontos := 0

	for nivel := 0; nivel < 4; nivel++ {
		for pid, pref := range prefs {
			if alocados[pid] || len(pref) <= nivel { continue }
			dia := pref[nivel]

			var melhor *Mesa
			for _, m := range porDia[dia] {
				if ocupado[m.ID] >= MAX_PESSOAS_POR_MESA { continue }
				hardBlock, softTouch := conflitante(m.Avaliadores, pid, hard, soft)
				if hardBlock { continue }

				if melhor == nil {
					melhor = m // prioriza mesa SEM softTouch
				} else {
					_, melhorSoftTouch := conflitante(melhor.Avaliadores, pid, hard, soft)
					if !softTouch && melhorSoftTouch {
						melhor = m
					}
				}
			}
			if melhor == nil { continue }

			// aloca na melhor mesa encontrada
			ocupado[melhor.ID]++
			melhor.Candidatos = append(melhor.Candidatos, pid)
			alocados[pid] = true
			aloc[pid] = melhor.ID
			pontos += nivel
		}
	}


	// remove mesas que ficaram abaixo do mínimo
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
	return ResultadoAlocacao{Alocacao: aloc, Pontuacao: pontos, Alocados: len(aloc)}
}

// ==================================================
// ===== GERADOR DE PERMUTAÇÕES (PARALELIZADO) ======
// ==================================================

// func gerarPermutacoesParalelo(ctx context.Context, horarios []*Horario, prefs map[int][]int, restr map[int]map[int]bool, maxTestes, workers int) ResultadoAlocacao {
// 	best := ResultadoAlocacao{Pontuacao: 1 << 30, Alocados: -1}
// 	permCh := make(chan []*Horario, workers)
// 	var mu sync.Mutex
// 	ctx, cancel := context.WithCancel(ctx)
// 	defer cancel()

// 	g, ctx := errgroup.WithContext(ctx)

// 	for w := 0; w < workers; w++ {
// 		g.Go(func() error {
// 			for perm := range permCh {
// 				res := fazerAlocacaoAvaliada(perm, prefs, restr)
// 				mu.Lock()
// 				if res.Alocados > best.Alocados || (res.Alocados == best.Alocados && res.Pontuacao < best.Pontuacao) {
// 					best = res
// 					if best.Alocados >= MELHOR_CASO {
// 						cancel()
// 					}
// 				}
// 				mu.Unlock()
// 			}
// 			return nil
// 		})
// 	}

// 	g.Go(func() error {
// 		defer close(permCh)
// 		count := 0
// 		var heap func([]*Horario, int)
// 		heap = func(a []*Horario, n int) {
// 			if ctx.Err() != nil || count >= maxTestes {
// 				return
// 			}
// 			if n == 1 {
// 				tmp := make([]*Horario, len(a))
// 				copy(tmp, a)
// 				permCh <- tmp
// 				count++
// 				return
// 			}
// 			for i := 0; i < n; i++ {
// 				heap(a, n-1)
// 				if n%2 == 1 {
// 					a[0], a[n-1] = a[n-1], a[0]
// 				} else {
// 					a[i], a[n-1] = a[n-1], a[i]
// 				}
// 			}
// 		}
// 		heap(horarios, len(horarios))
// 		return nil
// 	})

// 	_ = g.Wait()
// 	return best
// }

// ==================================================
// ============= IMPRESSÃO DOS RESULTADOS ===========
// ==================================================

func imprimirAlocacao(aloc map[int]int, horarios map[int]*Horario) int {
	fmt.Println("\n---- ALOCAÇÃO FINAL ----")
	alSet := make(map[int]bool) // quem foi alocado

	for pid, hid := range aloc {
		h := horarios[hid]
		fmt.Printf("Pessoa %d -> %s (horário ID %d)\n", pid, h.Descricao, h.ID)
		alSet[pid] = true
	}

	// --- calcula totais únicos --------------------------
	totalSet := make(map[int]bool)
	for _, h := range horarios {
		for _, pid := range h.Candidatos {
			totalSet[pid] = true
		}
	}

	var nao []int
	for pid := range totalSet {
		if !alSet[pid] {
			nao = append(nao, pid)
		}
	}
	// ----------------------------------------------------

	fmt.Printf("\n---- NÃO ALOCADOS (%d) ----\n%v\n", len(nao), nao)
	return len(totalSet)
}

func imprimirHorariosPreenchidos(horarios map[int]*Horario, aloc map[int]int, total int) {
	fmt.Printf("\n---- HORÁRIOS PREENCHIDOS ----\nCandidatos totais: %d\n\n", total)

	m := make(map[int][]int)
	for pid, hid := range aloc {
		m[hid] = append(m[hid], pid)
	}

	var preenchidos []horarioInfo
	for _, h := range horarios {
		if ps := m[h.ID]; len(ps) > 0 {
			preenchidos = append(preenchidos, horarioInfo{h, ps})
		}
	}

	sort.Slice(preenchidos, func(i, j int) bool {
		return len(preenchidos[i].Pessoas) > len(preenchidos[j].Pessoas)
	})

	for _, inf := range preenchidos {
		fmt.Printf(
			"Horário %d (%s): %d pessoas – %v | Avaliadores: %v\n",
			inf.H.ID, inf.H.Descricao,
			len(inf.Pessoas), inf.Pessoas,
			inf.H.Avaliadores,
		)
	}
}

func imprimirAlocacaoMesas(aloc map[int]int, mesas map[int]*Mesa, prefs map[int][]int) int {
	fmt.Println("\n---- ALOCAÇÃO FINAL ----")
	alocados := make(map[int]bool)
	for pid, mid := range aloc {
		fmt.Printf("Pessoa %d -> %s (Mesa %d)\n", pid, mesas[mid].Descricao, mid)
		alocados[pid] = true
	}

	// total único de pessoas que possuem preferência registrada
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
		fmt.Printf("%s (%d candidatos) – %v | Avaliadores: %v\n",
			m.Descricao, len(m.Candidatos), m.Candidatos, m.Avaliadores)
	}
}

// ==================================================
// ===================== MAIN =======================
// ==================================================

func Alocar(db *sql.DB) {
	fmt.Println("---- INICIANDO ALOCAÇÃO ----")
	// --- carrega dados do banco --------------------------------------------
	avals := carregarAvaliadores(db)
	hard, soft := carregarRestricoes(db)
	horarios := carregarHorarios(db)
	prefs := carregarDisponibilidades(db, horarios)

	fmt.Println("---- DADOS CARREGADOS ----")

	// --- gera MESAS (painéis) p/ cada dia ----------------------------------
	mesas, porDia := gerarMesas(horarios, avals)
	fmt.Println("\n---- MESAS GERADAS ----")
	for _, m := range mesas {
		fmt.Printf("Mesa %d → %s | Avaliadores: %v\n",
			m.ID, m.Descricao, m.Avaliadores)
	}
	fmt.Println(strings.Repeat("-", 60))

	// --- alocação -----------------------------------------------------------
	start := time.Now()
	res := fazerAlocacaoMesas(mesas, porDia, prefs, hard, soft)

	// índice mesaID -> *Mesa  (facilita buscas na impressão)
	mapMesa := make(map[int]*Mesa, len(mesas))
	for _, m := range mesas {
		mapMesa[m.ID] = m
	}

	// --- impressão de resultados -------------------------------------------
	total := imprimirAlocacaoMesas(res.Alocacao, mapMesa, prefs)
	imprimirMesasPreenchidas(mesas, res.Alocacao, total)

	fmt.Printf("\nTempo total de execução: %v\n", time.Since(start))
}
