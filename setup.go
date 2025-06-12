package main

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	// Abre (ou cria) o arquivo de banco de dados
	db, err := sql.Open("sqlite3", "./insper.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Cria tabela de pessoas
	_, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS pessoa (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            nome TEXT NOT NULL,
            email_insper TEXT NOT NULL UNIQUE,
            email_pessoal TEXT NOT NULL UNIQUE,
            cpf TEXT NOT NULL UNIQUE,
            semestre INTEGER NOT NULL,
            curso TEXT NOT NULL
        );
    `)
	if err != nil {
		log.Fatalf("erro criando tabela pessoa: %v", err)
	}

	// Cria tabela de opções de horário
	_, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS opcoes_horario (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            data DATE NOT NULL,
            hora TIME NOT NULL,
            local TEXT NOT NULL
        );
    `)
	if err != nil {
		log.Fatalf("erro criando tabela opcoes_horario: %v", err)
	}

	// Cria tabela de disponibilidade
	_, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS disponibilidade (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            pessoa_id INTEGER NOT NULL,
            opcao_id INTEGER NOT NULL,
            FOREIGN KEY(pessoa_id) REFERENCES pessoa(id),
            FOREIGN KEY(opcao_id) REFERENCES opcoes_horario(id)
        );
    `)
	if err != nil {
		log.Fatalf("erro criando tabela disponibilidade: %v", err)
	}

	log.Println("Banco e tabelas criados com sucesso!")
}
