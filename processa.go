package main

import (
	"database/sql"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"
	"math/big"

	_ "github.com/mattn/go-sqlite3"
)

const (
	MIN_PESSOAS_POR_HORARIO = 5
	MAX_PESSOAS_POR_HORARIO = 60
)

type Horario struct {
	ID         int
	Data       string
	Hora       string
	Candidatos []int
}

type ResultadoAlocacao struct {
	Alocacao  map[int]int
	Pontuacao int
	Alocados  int
}


func fatorialBig(n int) *big.Int {
	result := big.NewInt(1)
	for i := 2; i <= n; i++ {
		result.Mul(result, big.NewInt(int64(i)))
	}
	return result
}

// -=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=

func carregarHorarios(db *sql.DB) map[int]*Horario { // Lê todos os horários disponíveis e monta a estrutura de dados.
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

func carregarDisponibilidades(db *sql.DB, horarios map[int]*Horario) map[int][]int { // Constrói as listas de preferências das pessoas e preenche os candidatos de cada horário.
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

// -=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=

func filtrarHorariosValidos(horarios map[int]*Horario) []*Horario { // Tira horarios com menos gente que o minimo
	validHorarios := []*Horario{}

	for _, h := range horarios {
		if len(h.Candidatos) >= MIN_PESSOAS_POR_HORARIO {
			validHorarios = append(validHorarios, h)
		} else {
			fmt.Printf("Cortando horário %d (%s %s) - candidatos insuficientes (%d)\n", h.ID, h.Data, h.Hora, len(h.Candidatos))
		}
	}

	return validHorarios
} // Retorna uma lista com os horários válidos

// -=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=

func sortHorariosPorCandidatos(horarios []*Horario) { // Ordena os horários para que os com menos candidatos
	sort.SliceStable(horarios, func(i, j int) bool {
		a, b := horarios[i], horarios[j]
		if len(a.Candidatos) == len(b.Candidatos) {
			return a.ID < b.ID // desempate fixo
		}
		return len(a.Candidatos) < len(b.Candidatos)
	})
}

// -=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=

func fazerAlocacaoAvaliada(horarios []*Horario, pessoaPreferencias map[int][]int) ResultadoAlocacao {
	alocacao := map[int]int{}
	pessoasAlocadas := map[int]bool{}
	pontuacao := 0
	// totalAlocados := 0

	for _, h := range horarios {
		alocados := 0

		for pessoaID, preferencias := range pessoaPreferencias {
			if pessoasAlocadas[pessoaID] || len(preferencias) == 0 {
				continue
			}
			if preferencias[0] == h.ID && alocados < MAX_PESSOAS_POR_HORARIO {
				alocarPessoa(pessoaID, h.ID, alocacao, pessoasAlocadas)
				alocados++
			}
		}

		for nivel := 1; nivel < 5 && alocados < MIN_PESSOAS_POR_HORARIO; nivel++ {
			for pessoaID, preferencias := range pessoaPreferencias {
				if pessoasAlocadas[pessoaID] || len(preferencias) <= nivel {
					continue
				}
				if preferencias[nivel] == h.ID && alocados < MAX_PESSOAS_POR_HORARIO {
					alocarPessoa(pessoaID, h.ID, alocacao, pessoasAlocadas)
					alocados++
				}
			}
		}

		if alocados < MIN_PESSOAS_POR_HORARIO {
			for pessoaID, opcaoID := range alocacao {
				if opcaoID == h.ID {
					delete(alocacao, pessoaID)
					pessoasAlocadas[pessoaID] = false
				}
			}
		}
	}

	for pessoaID, horarioID := range alocacao {
		preferencias := pessoaPreferencias[pessoaID]
		for i, pref := range preferencias {
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


func alocarPessoa(pessoaID int, opcaoID int, alocacao map[int]int, pessoasAlocadas map[int]bool) {
	alocacao[pessoaID] = opcaoID
	pessoasAlocadas[pessoaID] = true
} // Marca que uma pessoa foi alocada

func imprimirAlocacao(alocacao map[int]int, horarios map[int]*Horario) int { // Imprime quem foi alocado em qual horário.
	fmt.Printf("\n---- ALOCAÇÃO FINAL ----\n")
	alocados := map[int]bool{}
	naoAlocados := []int{}
	for pessoaID, opcaoID := range alocacao {
		h := horarios[opcaoID]
		fmt.Printf("Pessoa %d -> %s %s (Horario ID %d)\n", pessoaID, h.Data, h.Hora, h.ID)
		alocados[pessoaID] = true
	}

	// Descobrir todas as pessoas possíveis
	todasPessoas := map[int]bool{}
	for _, h := range horarios {
		for _, pessoaID := range h.Candidatos {
			todasPessoas[pessoaID] = true
		}
	}

	// Descobrir quem não foi alocado
	for pessoaID := range todasPessoas {
		if !alocados[pessoaID] {
			naoAlocados = append(naoAlocados, pessoaID)
		}
	}

	// Imprimir não alocados
	fmt.Printf("\n---- NÃO ALOCADOS (%d) ----\n", len(naoAlocados))
	fmt.Println(naoAlocados)
	qntTotal := len(alocacao) + len(naoAlocados)

	return qntTotal
}

func imprimirHorariosPreenchidos(horarios map[int]*Horario, alocacao map[int]int, qntTotal int) {
	fmt.Printf("\n---- HORÁRIOS PREENCHIDOS ----\n")
	fmt.Printf("Quantidade de candidatos total: %d\n\n", qntTotal)


	horarioToPessoas := make(map[int][]int)
	for pessoaID, horarioID := range alocacao {
		horarioToPessoas[horarioID] = append(horarioToPessoas[horarioID], pessoaID)
	}

	for _, h := range horarios {
		pessoas := horarioToPessoas[h.ID]
		fmt.Printf("Horário %d (%s %s): %d pessoas - %v\n", h.ID, h.Data, h.Hora, len(pessoas), pessoas)
		// fmt.Printf("Horário %d (%s %s): %d pessoas\n", h.ID, h.Data, h.Hora, len(pessoas))
	}
}

func gerarPermutacoesLimitadas(horarios []*Horario, pessoaPreferencias map[int][]int, maxTestes int) ResultadoAlocacao {
	var melhor ResultadoAlocacao
	melhor.Pontuacao = 1 << 30
	melhor.Alocados = -1

	var gerar func([]*Horario, int, int)
	testes := 0
	inicioTotal := time.Now()

	gerar = func(arr []*Horario, n int, nivel int) {
		if testes >= maxTestes {
			return
		}
		if n == 1 {
			testes++
			tmp := make([]*Horario, len(arr))
			copy(tmp, arr)
			inicio := time.Now()
			result := fazerAlocacaoAvaliada(tmp, pessoaPreferencias)
			duracao := time.Since(inicio)

			fmt.Printf("Permutação #%d | Alocados: %d | Pontuação: %d | Tempo: %v\n", testes, result.Alocados, result.Pontuacao, duracao)

			if result.Alocados > melhor.Alocados ||
				(result.Alocados == melhor.Alocados && result.Pontuacao < melhor.Pontuacao) {
				melhor = result
			}
			return
		}
		for i := 0; i < n; i++ {
			gerar(arr, n-1, nivel+1)
			if n%2 == 1 {
				arr[0], arr[n-1] = arr[n-1], arr[0]
			} else {
				arr[i], arr[n-1] = arr[n-1], arr[i]
			}
		}
	}

	gerar(horarios, len(horarios), 0)

	fmt.Printf("\nTotal de permutações executadas: %d\n", testes)
	fmt.Printf("Tempo total de execução: %v\n", time.Since(inicioTotal))
	return melhor
}


// -=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=

func main() {
	db, err := sql.Open("sqlite3", "./insper.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	horarios := carregarHorarios(db) // Carrega os horários disponíveis do banco de dados
	pessoaPreferencias := carregarDisponibilidades(db, horarios) // Carrega as preferências de horários de cada pessoa e preenche os candidatos de cada horário
	
	// -=-=-=-=-=-=-=-=-
	// fmt.Print("---- CORTANDO HORÁRIOS INVÁLIDOS ----\n")
	validHorarios := filtrarHorariosValidos(horarios) // Filtra os horários válidos com base no número de candidatos
	// fmt.Printf("%s\n", strings.Repeat("---", 30))
	// -=-=-=-=-=-=-=-=-
	
	// -=-=-=-=-=-=-=-=-
	sortHorariosPorCandidatos(validHorarios) // Ordena os horários por número de candidatos (do menor para o maior)
	fmt.Printf("\n---- HORÁRIOS VÁLIDOS ----\n")
	for _, h := range validHorarios {
		fmt.Printf("Horário %d (%s %s): %d candidatos\n", h.ID, h.Data, h.Hora, len(h.Candidatos))
	}
	fmt.Printf("%s\n", strings.Repeat("---", 30))
	// -=-=-=-=-=-=-=-=-
	
	// -=-=-=-=-=-=-=-=-
	fmt.Printf("\n---- INICIANDO ALOCAÇÃO ----\n")
	fmt.Printf("Total de permutações possíveis: %s\n", fatorialBig(len(validHorarios)).String())


	MAX_TESTES := 100000

	
	melhorResultado := gerarPermutacoesLimitadas(validHorarios, pessoaPreferencias, MAX_TESTES)
	qntTotal := imprimirAlocacao(melhorResultado.Alocacao, horarios)
	imprimirHorariosPreenchidos(horarios, melhorResultado.Alocacao, qntTotal)
}
