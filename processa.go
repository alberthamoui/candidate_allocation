package main

// ==================================================
// ============== IMPORTS E CONSTANTES ==============
// ==================================================

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"math/big"
	"runtime"
	"sort"
	"strings"
	"sync"
	// "sync/atomic"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/sync/errgroup"
)

const (
    MIN_PESSOAS_POR_HORARIO = 5
    MAX_PESSOAS_POR_HORARIO = 8
    MAX_TESTES              = 1_000_000
    MELHOR_CASO             = 50
)

// -=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-
// ==================================================
// ==================== STRUCTS =====================
// ==================================================

type Horario struct {
    ID         int
    Data       string
    Hora       string
    Candidatos []int
}

type ResultadoAlocacao struct {
    Alocacao  map[int]int // pessoaID -> horarioID
    Pontuacao int
    Alocados  int
}

type horarioComAlocados struct {
    H       *Horario
    Pessoas []int
}

// -=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-
// ==================================================
// =============== FUNÇÕES UTILITÁRIAS ==============
// ==================================================

func fatorialBig(n int) *big.Int {
    result := big.NewInt(1)
    for i := 2; i <= n; i++ {
        result.Mul(result, big.NewInt(int64(i)))
    }
    return result
}

// -=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-
// ==================================================
// =========== CARREGAMENTO DE DADOS DB =============
// ==================================================

func carregarHorarios(db *sql.DB) map[int]*Horario {
    horarios := map[int]*Horario{}

    rows, err := db.Query(`SELECT id, data, hora FROM opcoes_horario`)
    if err != nil {
        log.Fatal(err)
    }
    defer rows.Close()

    for rows.Next() {
        var h Horario
        if err := rows.Scan(&h.ID, &h.Data, &h.Hora); err != nil {
            log.Fatal(err)
        }
        h.Candidatos = []int{}
        horarios[h.ID] = &h
    }
    return horarios
}

func carregarDisponibilidades(db *sql.DB, horarios map[int]*Horario) map[int][]int {
    pessoaPreferencias := map[int][]int{}

    rows, err := db.Query(`SELECT pessoa_id, horario_id FROM disponibilidade`)
    if err != nil {
        log.Fatal(err)
    }
    defer rows.Close()

    for rows.Next() {
        var pessoaID, opcaoID int
        if err := rows.Scan(&pessoaID, &opcaoID); err != nil {
            log.Fatal(err)
        }
        horarios[opcaoID].Candidatos = append(horarios[opcaoID].Candidatos, pessoaID)
        pessoaPreferencias[pessoaID] = append(pessoaPreferencias[pessoaID], opcaoID)
    }
    return pessoaPreferencias
}

// -=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-
// ==================================================
// ========= PRÉ-PROCESSAMENTO DE HORÁRIOS ==========
// ==================================================

func filtrarHorariosValidos(horarios map[int]*Horario) []*Horario {
    valid := []*Horario{}
    for _, h := range horarios {
        if len(h.Candidatos) >= MIN_PESSOAS_POR_HORARIO {
            valid = append(valid, h)
        } else {
            fmt.Printf("Cortando horário %d (%s %s) - candidatos insuficientes (%d)\n", h.ID, h.Data, h.Hora, len(h.Candidatos))
        }
    }
    return valid
}

func sortHorariosPorCandidatos(horarios []*Horario) {
    sort.SliceStable(horarios, func(i, j int) bool {
        a, b := horarios[i], horarios[j]
        if len(a.Candidatos) == len(b.Candidatos) {
            return a.ID < b.ID
        }
        return len(a.Candidatos) < len(b.Candidatos)
    })
}

// -=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-
// ==================================================
// ============== ALOCAÇÃO DE PESSOAS ===============
// ==================================================

func alocarPessoa(pessoaID, horarioID int, alocacao map[int]int, pessoasAlocadas map[int]bool) {
    alocacao[pessoaID] = horarioID
    pessoasAlocadas[pessoaID] = true
}

func fazerAlocacaoAvaliada(horarios []*Horario, pessoaPreferencias map[int][]int) ResultadoAlocacao {
    alocacao := map[int]int{}
    pessoasAlocadas := map[int]bool{}
    pontuacao := 0

    for _, h := range horarios {
        alocados := 0

        // 1ª preferência
        for pessoaID, prefs := range pessoaPreferencias {
            if pessoasAlocadas[pessoaID] || len(prefs) == 0 {
                continue
            }
            if prefs[0] == h.ID && alocados < MAX_PESSOAS_POR_HORARIO {
                alocarPessoa(pessoaID, h.ID, alocacao, pessoasAlocadas)
                alocados++
            }
        }

        // preferências 2..4 enquanto não bate mínimo
        for nivel := 1; nivel < 5 && alocados < MIN_PESSOAS_POR_HORARIO; nivel++ {
            for pessoaID, prefs := range pessoaPreferencias {
                if pessoasAlocadas[pessoaID] || len(prefs) <= nivel {
                    continue
                }
                if prefs[nivel] == h.ID && alocados < MAX_PESSOAS_POR_HORARIO {
                    alocarPessoa(pessoaID, h.ID, alocacao, pessoasAlocadas)
                    alocados++
                }
            }
        }

        // se ainda insuficiente, desfaz alocações deste horário
        if alocados < MIN_PESSOAS_POR_HORARIO {
            for pessoaID, horarioEscolhido := range alocacao {
                if horarioEscolhido == h.ID {
                    delete(alocacao, pessoaID)
                    pessoasAlocadas[pessoaID] = false
                }
            }
        }
    }

    // calcula pontuação
    for pessoaID, horarioID := range alocacao {
        prefs := pessoaPreferencias[pessoaID]
        for i, pref := range prefs {
            if pref == horarioID {
                pontuacao += i
                break
            }
        }
    }

    return ResultadoAlocacao{
        Alocacao:  alocacao,
        Pontuacao: pontuacao,
        Alocados:  len(alocacao),
    }
}

// -=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-
// ==================================================
// ============= IMPRESSÃO DOS RESULTADOS ===========
// ==================================================

func imprimirAlocacao(alocacao map[int]int, horarios map[int]*Horario) int {
    fmt.Printf("\n---- ALOCAÇÃO FINAL ----\n")
    alocadosSet := map[int]bool{}
    naoAlocados := []int{}

    for pessoaID, horarioID := range alocacao {
        h := horarios[horarioID]
        fmt.Printf("Pessoa %d -> %s %s (Horario ID %d)\n", pessoaID, h.Data, h.Hora, h.ID)
        alocadosSet[pessoaID] = true
    }

    // descobrir todas as pessoas possíveis
    todasPessoas := map[int]bool{}
    for _, h := range horarios {
        for _, p := range h.Candidatos {
            todasPessoas[p] = true
        }
    }
    for pid := range todasPessoas {
        if !alocadosSet[pid] {
            naoAlocados = append(naoAlocados, pid)
        }
    }

    fmt.Printf("\n---- NÃO ALOCADOS (%d) ----\n%v\n", len(naoAlocados), naoAlocados)
    return len(alocacao) + len(naoAlocados)
}

func imprimirHorariosPreenchidos(horarios map[int]*Horario, alocacao map[int]int, total int) {
    fmt.Printf("\n---- HORÁRIOS PREENCHIDOS ----\nCandidatos totais: %d\n\n", total)

    horarioToPessoas := map[int][]int{}
    for pessoaID, horarioID := range alocacao {
        horarioToPessoas[horarioID] = append(horarioToPessoas[horarioID], pessoaID)
    }

    // cria slice apenas com horários com pessoas
    type horarioInfo struct {
        H       *Horario
        Pessoas []int
    }
    var preenchidos []horarioInfo
    for _, h := range horarios {
        pessoas := horarioToPessoas[h.ID]
        if len(pessoas) > 0 {
            preenchidos = append(preenchidos, horarioInfo{h, pessoas})
        }
    }

    // ordena por quantidade de pessoas (decrescente)
    sort.Slice(preenchidos, func(i, j int) bool {
        return len(preenchidos[i].Pessoas) > len(preenchidos[j].Pessoas)
    })

    // imprime
    for _, info := range preenchidos {
        fmt.Printf("Horário %d (%s %s): %d pessoas - %v\n",
            info.H.ID, info.H.Data, info.H.Hora, len(info.Pessoas), info.Pessoas)
    }
}

// -=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-
// ==================================================
// ===== GERADOR DE PERMUTAÇÕES (PARALELIZADO) ======
// ==================================================

func gerarPermutacoesParalelo(parentCtx context.Context, horarios []*Horario, prefs map[int][]int, maxTestes, numWorkers int) (ResultadoAlocacao, error) {
    // contexto cancelável
    ctx, cancel := context.WithCancel(parentCtx)
    defer cancel()

    g, ctx := errgroup.WithContext(ctx)

    permCh := make(chan []*Horario, numWorkers) // back-pressure = numWorkers

    // estado compartilhado
    var bestMu sync.Mutex
    best := ResultadoAlocacao{Pontuacao: 1 << 30, Alocados: -1}
    // var counter uint64

    // === workers ===
    for w := 0; w < numWorkers; w++ {
        g.Go(func() error {
            for perm := range permCh {
                if ctx.Err() != nil {
                    return ctx.Err()
                }

                // start := time.Now()
                resultado := fazerAlocacaoAvaliada(perm, prefs)
                // n := atomic.AddUint64(&counter, 1)
                // fmt.Printf("Test #%d - Alocados:%d Pontuação:%d Tempo:%v\n", n, resultado.Alocados, resultado.Pontuacao, time.Since(start))

                bestMu.Lock()
                if resultado.Alocados > best.Alocados || (resultado.Alocados == best.Alocados && resultado.Pontuacao < best.Pontuacao) {
                    best = resultado
                    if best.Alocados >= MELHOR_CASO {
                        cancel() // atingiu alvo ótimo
                    }
                }
                bestMu.Unlock()
            }
            return nil
        })
    }

    // === gerador de permutações (Heap) ===
    g.Go(func() error {
        defer close(permCh)
        testes := 0
        var heap func([]*Horario, int)
        heap = func(arr []*Horario, n int) {
            if ctx.Err() != nil || testes >= maxTestes {
                return
            }
            if n == 1 {
                tmp := make([]*Horario, len(arr))
                copy(tmp, arr)
                permCh <- tmp
                testes++
                return
            }
            for i := 0; i < n; i++ {
                heap(arr, n-1)
                if n%2 == 1 {
                    arr[0], arr[n-1] = arr[n-1], arr[0]
                } else {
                    arr[i], arr[n-1] = arr[n-1], arr[i]
                }
                if ctx.Err() != nil || testes >= maxTestes {
                    return
                }
            }
        }
        heap(horarios, len(horarios))
        return nil
    })

    // === espera tudo terminar ===
    if err := g.Wait(); err != nil && err != context.Canceled {
        return best, err
    }
    return best, nil
}

// -=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-
// ==================================================
// =================== FUNÇÃO MAIN ==================
// ==================================================

func main() {
    db, err := sql.Open("sqlite3", "./insper.db")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    // 1. carrega dados
    horarios := carregarHorarios(db)
    pessoaPreferencias := carregarDisponibilidades(db, horarios)

    // 2. pré-processa horários
    validHorarios := filtrarHorariosValidos(horarios)
    sortHorariosPorCandidatos(validHorarios)

    fmt.Printf("\n---- HORÁRIOS VÁLIDOS ----\n")
    for _, h := range validHorarios {
        fmt.Printf("Horário %d (%s %s): %d candidatos\n", h.ID, h.Data, h.Hora, len(h.Candidatos))
    }
    fmt.Println(strings.Repeat("-", 60))

    // 3. informa espaço de busca
    fmt.Printf("\n---- INICIANDO ALOCAÇÃO ----\n")
    fmt.Printf("Total de permutações possíveis: %s\n", fatorialBig(len(validHorarios)).String())

    // 4. paraleliza busca
    start := time.Now()
    melhorResultado, err := gerarPermutacoesParalelo(context.Background(), validHorarios, pessoaPreferencias, MAX_TESTES, runtime.NumCPU())
    if err != nil && err != context.Canceled {
        log.Fatalf("erro ao gerar permutações: %v", err)
    }

    // 5. imprime saída
    total := imprimirAlocacao(melhorResultado.Alocacao, horarios)
    imprimirHorariosPreenchidos(horarios, melhorResultado.Alocacao, total)
    fmt.Printf("\nTempo total de execução: %v\n", time.Since(start))
}
