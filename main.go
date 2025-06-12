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
	MAX_PESSOAS_POR_HORARIO = 8
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


	return horarios
}

func carregarDisponibilidades(db *sql.DB, horarios map[int]*Horario) map[int][]int { // Constrói as listas de preferências das pessoas e preenche os candidatos de cada horário.
	pessoaPreferencias := map[int][]int{}

	
	return pessoaPreferencias
}

// -=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=

func filtrarHorariosValidos(horarios map[int]*Horario) []*Horario { // Tira horarios com menos gente que o minimo
	validHorarios := []*Horario{}

	

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

	return alocacao
} // Retorna um dic {pessoa_id:horario_id}

func alocarPessoa(pessoaID int, opcaoID int, alocacao map[int]int, pessoasAlocadas map[int]bool) {
	alocacao[pessoaID] = opcaoID
	pessoasAlocadas[pessoaID] = true
} // Marca que uma pessoa foi alocada

func imprimirAlocacao(alocacao map[int]int, horarios map[int]*Horario) { // Imprime quem foi alocado em qual horário.
	fmt.Printf("\n---- ALOCAÇÃO FINAL ----\n")
	for pessoaID, opcaoID := range alocacao {
		h := horarios[opcaoID]
		fmt.Printf("Pessoa %d -> %s %s (Horario ID %d)\n", pessoaID, h.Data, h.Hora, h.ID)
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
}