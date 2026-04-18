package main

// go run example_cli.go app.go models.go mapping.go export.go alocate.go processa.go setup.go -file ./Execelteste/base_exemplo.xlsx

// ==================================================
// ============== IMPORTS E CONSTANTES ==============
// ==================================================

import (
	// "context"
	"database/sql"
	"fmt"
	"log"
	"math"
	"math/rand"
	"runtime"
	"sort"
	"strconv"
	"strings"
	// "sync"
	// "sync/atomic"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// ~10.000 p/seg
const (
	// CONFIGURAÇÕES DE ALOCAÇÃO
	MESAS_POR_HORARIO        = 5
	MIN_PESSOAS_POR_MESA     = 5
	MAX_PESSOAS_POR_MESA     = 8
	MIN_AVALIADORES_POR_MESA = 5
	MAX_AVALIADORES_POR_MESA = 5
	INFINITY = math.MaxInt

	// CONFIGURAÇÕES DE EXECUÇÃO
	NUM_TENTATIVAS = 100_000
	NOTA_MINIMA    = 95
	SCORE_BASE     = 100
	PARALELO = false
	NOTA_TENTATIVA = "tentativa"
	// NOTA_TENTATIVA = "nota"

	// CONTROLE DE NAO ALOCADOS
	MAX_NAO_ALOCADOS = 1

	// CONTROLE DE LOGS
	PRINT_QUANTIDADE = 1000
)

var TENTATIVA = 0

// ==================================================
// =========== CRITÉRIOS DE PONTUAÇÃO ===============
// ==================================================
const (
	PONTOS_OPCAO_1         = 0     // candidato alocado na 1ª opção de horário
	PONTOS_OPCAO_2         = -1   // candidato alocado na 2ª opção de horário
	PONTOS_OPCAO_3         = -3   // candidato alocado na 3ª opção de horário
	PONTOS_OPCAO_4         = -5   // candidato alocado na 4ª opção de horário
	PONTOS_OPCAO_5         = -7   // candidato alocado na 5ª opção de horário
	PENALIDADE_SOFT        = -5   // violação de restrição "prefiro não" por avaliador
	PENALIDADE_NAO_ALOCADO = -1000 // candidato que não foi alocado
	PENALIDADE_HARD        = -1000 // violação de restrição "não posso" por avaliador
)

var MELHOR_MAP_PENALIDADES = map[string]int{
	"opcao_1":     0,
	"opcao_2":     0,
	"opcao_3":     0,
	"opcao_4":     0,
	"opcao_5":     0,
	"nao_alocado": 0,
	"prefiro_nao": 0,
	"nao_posso":   0,
}

// ==================================================
// ==================== STRUCTS =====================
// ==================================================

type Mesa struct {
	ID          int    // único (ex.: 301 = quarta-mesa1)
	DiaID       int    // 1=segunda, 2=terça, ...
	Descricao   string // "quarta - mesa 2"
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

func gerarMesas(hmap map[int]*Horario, avals []*Avaliador) ([]*Mesa, map[int][]*Mesa) {
	var todas []*Mesa
	porDia := make(map[int][]*Mesa)

	// Limita o número de mesas ao que pode ser preenchido com o mínimo de
	// avaliadores. Isso garante que nenhuma mesa fique com menos de
	// MIN_AVALIADORES_POR_MESA avaliadores.
	numMesas := len(avals) / MIN_AVALIADORES_POR_MESA
	if numMesas > MESAS_POR_HORARIO {
		numMesas = MESAS_POR_HORARIO
	}
	if numMesas == 0 {
		log.Printf("[WARN] Avaliadores insuficientes para formar qualquer mesa (necessário mínimo: %d)", MIN_AVALIADORES_POR_MESA)
		return todas, porDia
	}

	for _, h := range hmap {
		// Embaralha avaliadores para distribuição aleatória entre as mesas.
		perm := make([]*Avaliador, len(avals))
		copy(perm, avals)
		rand.Shuffle(len(perm), func(i, j int) { perm[i], perm[j] = perm[j], perm[i] })

		idx := 0
		for i := range numMesas {
			m := &Mesa{
				ID:        h.ID*100 + i,
				DiaID:     h.ID,
				Descricao: fmt.Sprintf("%s - mesa %d", h.Descricao, i+1),
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
// ============== ALOCAÇÃO DE PESSOAS ===============
// ==================================================

func conflitante(avIDs []int, pid int, hard, soft map[int]map[int]bool) (hardBlock bool, softTouch bool) {
	for _, av := range avIDs {
		if hard[av][pid] {
			return true, true
		}
		if soft[av][pid] {
			softTouch = true
		}
	}
	return false, softTouch
}

func fazerAlocacaoMesas(mesas []*Mesa, porDia map[int][]*Mesa, prefs map[int][]int, hard, soft map[int]map[int]bool) ResultadoAlocacao {
	aloc := make(map[int]int)
	ocupado := make(map[int]int)
	alocados := make(map[int]bool)
	pontos := 0

	for nivel := 0; nivel < 4; nivel++ {
		for pid, pref := range prefs {
			if alocados[pid] || len(pref) <= nivel {
				continue
			}
			dia := pref[nivel]

			var melhor *Mesa
			for _, m := range porDia[dia] {
				if ocupado[m.ID] >= MAX_PESSOAS_POR_MESA {
					continue
				}
				hardBlock, softTouch := conflitante(m.Avaliadores, pid, hard, soft)
				if hardBlock {
					continue
				}

				if melhor == nil {
					melhor = m // prioriza mesa SEM softTouch
				} else {
					_, melhorSoftTouch := conflitante(melhor.Avaliadores, pid, hard, soft)
					if !softTouch && melhorSoftTouch {
						melhor = m
					}
				}
			}
			if melhor == nil {
				continue
			}

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
// ============= MULTI-TENTATIVA (MELHOR RESULTADO) =
// ==================================================

// pontuarResultado calcula a pontuação de uma alocação com base nos
// critérios definidos nas constantes PONTOS_OPCAO_*, PENALIDADE_SOFT e PENALIDADE_HARD.
// Retorna um valor inteiro — quanto maior, melhor o resultado.
func pontuarResultado(res ResultadoAlocacao, mesas []*Mesa, prefs map[int][]int, hard, soft map[int]map[int]bool) (int, map[string]int, int) {
	pontosNivel := []int{PONTOS_OPCAO_1, PONTOS_OPCAO_2, PONTOS_OPCAO_3, PONTOS_OPCAO_4, PONTOS_OPCAO_5}
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
				if nivel < len(pontosNivel) {
					score += pontosNivel[nivel]
					PONTOS_TOMADOS += pontosNivel[nivel]
				}
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

// copiarMesas retorna uma cópia profunda do slice de mesas.
func copiarMesas(mesas []*Mesa) []*Mesa {
	copia := make([]*Mesa, len(mesas))
	for i, m := range mesas {
		mc := *m
		mc.Candidatos = append([]int{}, m.Candidatos...)
		mc.Avaliadores = append([]int{}, m.Avaliadores...)
		copia[i] = &mc
	}
	return copia
}

// fazerMelhorAlocacaoMesas delega para gerarPermutacoesParalelo.
// onProgress é chamado a cada iteração: recebe (tentativaAtual, totalTentativas, melhorScore).
// Passe nil para desabilitar (ex.: uso CLI).
func fazerMelhorAlocacaoMesas(horarios map[int]*Horario, avals []*Avaliador, prefs map[int][]int, hard, soft map[int]map[int]bool, onProgress func(int, int, int)) (ResultadoAlocacao, []*Mesa) {
	// if PARALELO{
	// 	return gerarPermutacoesParalelo(horarios, avals, prefs, hard, soft)
	// }
var melhorRes ResultadoAlocacao
	var melhorMesas []*Mesa
	melhorScore := -INFINITY
	var melhorPontosTomados int
	
	localTentativa := 0
	fmt.Printf("INICIANDO ALOCAÇÃO: MODO %s\n", strings.ToUpper(NOTA_TENTATIVA))

	for {
		// Executa a lógica de alocação
		mesas, porDia := gerarMesas(horarios, avals)
		res := fazerAlocacaoMesas(mesas, porDia, prefs, hard, soft)
		score, MAP_PENALIDADES, PONTOS_TOMADOS := pontuarResultado(res, mesas, prefs, hard, soft)

		TENTATIVA++
		localTentativa++

		// 2. Atualiza o melhor resultado
		if melhorMesas == nil || score > melhorScore {
			MELHOR_MAP_PENALIDADES = MAP_PENALIDADES
			melhorScore = score
			melhorRes = res
			melhorMesas = copiarMesas(mesas)
			melhorPontosTomados = PONTOS_TOMADOS
		}

		// 3. Feedback de progresso
		totalEsperado := -1
		if NOTA_TENTATIVA == "tentativa" {
			totalEsperado = NUM_TENTATIVAS
		}
		
		if onProgress != nil {
			onProgress(localTentativa, totalEsperado, melhorScore)
		}

		if TENTATIVA%PRINT_QUANTIDADE == 0 {
			fmt.Printf("Tentativa %d: melhor pontuação = %d - Penalidades: %v - Pontos Tomados: %d\n", TENTATIVA, melhorScore, MELHOR_MAP_PENALIDADES, melhorPontosTomados)
		}

		// 4. Condições de Parada
		if MAX_NAO_ALOCADOS > MELHOR_MAP_PENALIDADES["nao_alocado"] {
			if melhorScore >= SCORE_BASE {
				break
			}
	
			// Condição específica: Modo Tentativa
			if NOTA_TENTATIVA == "tentativa" && localTentativa >= NUM_TENTATIVAS {
				break
			}
	
			// Condição específica: Modo Nota
			if NOTA_TENTATIVA == "nota" && melhorScore >= NOTA_MINIMA {
				break
			}
		}
	}

	fmt.Printf("Finalizado na Tentativa %d: Melhor score = %d\n", TENTATIVA, melhorScore)
	return melhorRes, melhorMesas
}

// ==================================================
// ===== GERADOR DE PERMUTAÇÕES (PARALELIZADO) ======
// ==================================================
// func gerarPermutacoesParalelo(horarios map[int]*Horario, avals []*Avaliador, prefs map[int][]int, hard, soft map[int]map[int]bool) (ResultadoAlocacao, []*Mesa) {
// 	workers := runtime.NumCPU()
// 	var mu sync.Mutex
// 	var melhorRes ResultadoAlocacao
// 	var melhorMesas []*Mesa
// 	melhorScore := -(1 << 62)
// 	ctx, cancel := context.WithCancel(context.Background())
// 	defer cancel()
// 	var contador atomic.Int64
// 	var wg sync.WaitGroup
// 	for w := 0; w < workers; w++ {
// 		wg.Add(1)
// 		go func() {
// 			defer wg.Done()
// 			for {
// 				select {
// 				case <-ctx.Done():
// 					return
// 				default:
// 				}
// 				t := int(contador.Add(1))
// 				if NOTA_TENTATIVA == "nota" && t > NUM_TENTATIVAS {
// 					cancel()
// 					return
// 				}
// 				mesas, porDia := gerarMesas(horarios, avals)
// 				res := fazerAlocacaoMesas(mesas, porDia, prefs, hard, soft)
// 				score, mapPen, pontosTomados := pontuarResultado(res, mesas, prefs, hard, soft)
// 				mu.Lock()
// 				atualizado := false
// 				if melhorMesas == nil || score > melhorScore {
// 					melhorScore = score
// 					melhorRes = res
// 					melhorMesas = copiarMesas(mesas)
// 					MELHOR_MAP_PENALIDADES = mapPen
// 					TENTATIVA = t
// 					atualizado = true
// 				}
// 				localScore := melhorScore
// 				mu.Unlock()
// 				if atualizado {
// 					fmt.Printf("Tentativa %d: novo melhor = %d | Penalidades: %v | Pontos Tomados: %d\n", t, localScore, mapPen, pontosTomados)
// 				}
// 				if NOTA_TENTATIVA == "tentativa" && localScore >= NOTA_MINIMA {
// 					cancel()
// 					return
// 				}
// 			}
// 		}()
// 	}
// 	wg.Wait()
// 	return melhorRes, melhorMesas
// }

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

	if PARALELO {
		fmt.Printf("Executando em modo paralelo com %d workers\n", runtime.NumCPU())
	}else{
		fmt.Println("Executando em modo sequencial")
	}
	fmt.Println("---- DADOS CARREGADOS ----")

	// --- busca a melhor alocação em NUM_TENTATIVAS tentativas ---------------
	fmt.Printf("\n---- BUSCANDO MELHOR ALOCAÇÃO (%d tentativas) || (%d nota minima) ----\n", NUM_TENTATIVAS, NOTA_MINIMA)
	start := time.Now()
	res, mesas := fazerMelhorAlocacaoMesas(horarios, avals, prefs, hard, soft, nil)

	fmt.Println("\n---- MESAS GERADAS (melhor resultado) ----")
	for _, m := range mesas {
		fmt.Printf("Mesa %d → %s | Avaliadores: %v\n",
			m.ID, m.Descricao, m.Avaliadores)
	}
	fmt.Println(strings.Repeat("-", 60))

	// índice mesaID -> *Mesa  (facilita buscas na impressão)
	mapMesa := make(map[int]*Mesa, len(mesas))
	for _, m := range mesas {
		mapMesa[m.ID] = m
	}

	// --- impressão de resultados -------------------------------------------
	total := imprimirAlocacaoMesas(res.Alocacao, mapMesa, prefs)
	imprimirMesasPreenchidas(mesas, res.Alocacao, total)

	pontuacao, MAP_PENALIDADES, _ := pontuarResultado(res, mesas, prefs, hard, soft)
	fmt.Printf("\nPontuação da melhor alocação: %d\n", pontuacao)
	fmt.Printf("Tempo total de execução: %v\n", time.Since(start))
	fmt.Printf("Penalidades detalhadas: %v\n", MAP_PENALIDADES)
}
