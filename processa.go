package main

import (
	"database/sql"
	"fmt"
	"log"
	"sort"

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
	sort.Slice(horarios, func(i, j int) bool {
		return len(horarios[i].Candidatos) < len(horarios[j].Candidatos)
	})
}

// -=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=

func fazerAlocacao(horarios []*Horario, pessoaPreferencias map[int][]int) map[int]int {
	alocacao := map[int]int{}
	pessoasAlocadas := map[int]bool{}

	for _, h := range horarios {
		fmt.Printf("\nAlocando horário %d (%s %s)\n", h.ID, h.Data, h.Hora)
		alocados := 0

		// Priorizar primeira opção
		for pessoaID, preferencias := range pessoaPreferencias {
			if pessoasAlocadas[pessoaID] {
				continue
			}
			if preferencias[0] == h.ID && alocados < MAX_PESSOAS_POR_HORARIO {
				alocarPessoa(pessoaID, h.ID, alocacao, pessoasAlocadas)
				alocados++
				fmt.Printf(" -> Alocado pessoa %d na PRIMEIRA opção\n", pessoaID)
			}
		}

		// Completar com outras opções se necessário
		if alocados < MAX_PESSOAS_POR_HORARIO {
			for _, pessoaID := range h.Candidatos {
				if pessoasAlocadas[pessoaID] {
					continue
				}
				if alocados < MAX_PESSOAS_POR_HORARIO {
					alocarPessoa(pessoaID, h.ID, alocacao, pessoasAlocadas)
					alocados++
					fmt.Printf(" -> Alocado pessoa %d em outra opção\n", pessoaID)
				}
			}
		}

		// Se não atingiu o mínimo, desfaz alocações deste horário
		if alocados < MIN_PESSOAS_POR_HORARIO {
			fmt.Printf(" -> Horário %d NÃO atingiu mínimo (%d/%d). Desfazendo alocações.\n", h.ID, alocados, MIN_PESSOAS_POR_HORARIO)
			for pessoaID, opcaoID := range alocacao {
				if opcaoID == h.ID {
					delete(alocacao, pessoaID)
					pessoasAlocadas[pessoaID] = false
				}
			}
		}
	}

	return alocacao
} // Retorna um dic {pessoa_id:horario_id}

func alocarPessoa(pessoaID int, opcaoID int, alocacao map[int]int, pessoasAlocadas map[int]bool) {
	alocacao[pessoaID] = opcaoID
	pessoasAlocadas[pessoaID] = true
} // Marca que uma pessoa foi alocada

func imprimirAlocacao(alocacao map[int]int, horarios map[int]*Horario) { // Imprime quem foi alocado em qual horário.
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
}

func imprimirHorariosPreenchidos(horarios map[int]*Horario, alocacao map[int]int) {
	fmt.Printf("\n---- HORÁRIOS PREENCHIDOS ----\n")
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

// -=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=

func main() {
	db, err := sql.Open("sqlite3", "./insper.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	horarios := carregarHorarios(db)
	pessoaPreferencias := carregarDisponibilidades(db, horarios)
	validHorarios := filtrarHorariosValidos(horarios)

	sortHorariosPorCandidatos(validHorarios)

	alocacao := fazerAlocacao(validHorarios, pessoaPreferencias)

	imprimirAlocacao(alocacao, horarios)

	imprimirHorariosPreenchidos(horarios, alocacao)
}
