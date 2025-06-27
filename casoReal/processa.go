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
	"os"
	"sort"
	"strings"
	"strconv"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const (
    MESAS_POR_HORARIO          = 5   
    MIN_PESSOAS_POR_MESA       = 5
    MAX_PESSOAS_POR_MESA       = 8
    MIN_AVALIADORES_POR_MESA   = 5   
    MAX_AVALIADORES_POR_MESA   = 5
    MAX_TESTES                 = 100_000
    MELHOR_CASO                = 50
)

// ==================================================
// ==================== STRUCTS =====================
// ==================================================

type Mesa struct {
    ID          int            // único (ex.: 301 = quarta-mesa1)
    DiaID       int            // 1=segunda, 2=terça, ...
    Descricao   string         // "quarta – mesa 2"
    Candidatos  []int          // já alocados
    Avaliadores []int
}

type ResultadoAlocacao struct {
	Alocacao  map[int]int
	Pontuacao int
	Alocados  int
}

type Avaliador struct {
	ID    int
	Nome  string
	Email string
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
		h := horarios[hid]
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

func gerarMesas(horarios map[int]*Horario, avals []*Avaliador) ([]*Mesa, map[int][]*Mesa) {
    var todas []*Mesa
    porDia := make(map[int][]*Mesa)

    rand.Seed(time.Now().UnixNano())

    for _, h := range horarios {
        for i := 0; i < MESAS_POR_HORARIO; i++ {
            idMesa := h.ID*100 + i // 101,102… 201,202…
            m := &Mesa{
                ID:        idMesa,
                DiaID:     h.ID,
                Descricao: fmt.Sprintf("%s – mesa %d", h.Descricao, i+1),
            }

            // Sorteia avaliadores p/ mesa
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

// ==================================================
// ========= PRÉ-PROCESSAMENTO DE HORÁRIOS =========
// ==================================================

func filtrarHorariosValidos(horarios map[int]*Horario) []*Horario {
	var valid []*Horario
	for _, h := range horarios {
		if len(h.Candidatos) >= MIN_PESSOAS_POR_MESA  {
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

func fazerAlocacaoMesas(mesas []*Mesa, porDia map[int][]*Mesa, prefs map[int][]int, restr map[int]map[int]bool) ResultadoAlocacao {
    aloc := make(map[int]int)      // pessoa -> mesaID
    alocados := make(map[int]bool) // set
    pontuacao := 0

    // Cada mesa começa com 0 pessoas
    ocupado := make(map[int]int)

    // percorre nível de preferência 0..3
    for nivel := 0; nivel < 4; nivel++ {
        for pid, pref := range prefs {
            if alocados[pid] || len(pref) <= nivel {
                continue
            }
            dia := pref[nivel]

            // procura 1ª mesa desse dia com vaga e sem restrição
            for _, m := range porDia[dia] {
                if ocupado[m.ID] >= MAX_PESSOAS_POR_MESA {
                    continue
                }
                if !podeAlocarNoHorario(&Horario{Avaliadores: m.Avaliadores}, pid, restr) {
                    continue
                }

                // aloca
                aloc[pid] = m.ID
                ocupado[m.ID]++
                m.Candidatos = append(m.Candidatos, pid)
                alocados[pid] = true
                pontuacao += nivel
                break
            }
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
    return ResultadoAlocacao{Alocacao: aloc, Pontuacao: pontuacao, Alocados: len(aloc)}
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

	// ordena mesas pela quantidade de candidatos
	sort.Slice(mesas, func(i, j int) bool {
		return len(mesas[i].Candidatos) > len(mesas[j].Candidatos)
	})

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

func main() {
    if len(os.Args) < 2 {
        log.Fatalf("uso: %s <caminho_banco.db>", os.Args[0])
    }
    dbPath := os.Args[1]

    // --- abre SQLite --------------------------------------------------------
    db, err := sql.Open("sqlite3", dbPath)
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    // --- carrega dados do banco --------------------------------------------
    avals    := carregarAvaliadores(db)
    restr    := carregarRestricoes(db)
    horarios := carregarHorarios(db)
    prefs    := carregarDisponibilidades(db, horarios)

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
    res   := fazerAlocacaoMesas(mesas, porDia, prefs, restr)

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
