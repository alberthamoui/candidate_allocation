package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/alberthamoui/candidate_allocation/db"
	_ "github.com/mattn/go-sqlite3"
)

func main() {

	// abre (ou cria) o banco
	conn, err := sql.Open("sqlite3", "./insper.db")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	// 1) adiciona 10 pessoas
	var personIDs []int64
	var numero string = "119898274312"
	for i := 1; i <= 10; i++ {
		nome := fmt.Sprintf("Pessoa %02d", i)
		emailInsper := fmt.Sprintf("p%02d@insper.edu.br", i)
		emailPessoal := fmt.Sprintf("p%02d@gmail.com", i)
		cpf := fmt.Sprintf("000.000.000-%02d", i)
		semestre := i
		curso := "Engenharia"
		id, err := db.AddPessoa(conn, nome, cpf, numero, emailInsper, emailPessoal, semestre, curso)
		if err != nil {
			log.Fatalf("erro AddPessoa %d: %v", i, err)
		}
		personIDs = append(personIDs, id)
	}

	// 2) adiciona 5 horários
	var horarioIDs []int64
	baseDate := time.Now()
	for i := 1; i <= 5; i++ {
		data := baseDate.AddDate(0, 0, i).Format("2006-01-02")
		hora := fmt.Sprintf("%02d:00:00", 8+i)
		local := fmt.Sprintf("Sala %d", i)
		id, err := db.AddHorario(conn, data, hora, local)
		if err != nil {
			log.Fatalf("erro AddHorario %d: %v", i, err)
		}
		horarioIDs = append(horarioIDs, id)
	}

	// 3) para cada pessoa, adiciona duas opções de disponibilidade
	for idx, pid := range personIDs {
		// escolhe dois índices de horário de forma circular
		j1 := idx % len(horarioIDs)
		j2 := (idx + 1) % len(horarioIDs)

		if _, err := db.AddDisponibilidade(conn, pid, horarioIDs[j1], 1); err != nil {
			log.Fatalf("erro AddDisp pessoa %d opção %d: %v", pid, horarioIDs[j1], err)
		}
		if _, err := db.AddDisponibilidade(conn, pid, horarioIDs[j2], 2); err != nil {
			log.Fatalf("erro AddDisp pessoa %d opção %d: %v", pid, horarioIDs[j2], err)
		}
	}

	log.Println("Inserções concluídas com sucesso")
}
