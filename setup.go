package main

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	db, err := sql.Open("sqlite3", "./insper.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	statements := []string{

		`PRAGMA foreign_keys = ON;`,

		// ------------------------------------------------------------------
		`DROP TABLE IF EXISTS avaliador;`,
		`CREATE TABLE "avaliador" (
			"id" INTEGER NOT NULL UNIQUE,
			"nome" TEXT NOT NULL UNIQUE,
			"email" TEXT NOT NULL UNIQUE,
			PRIMARY KEY("id" AUTOINCREMENT)
			);`,

// ------------------------------------------------------------------
		`DROP TABLE IF EXISTS pessoa;`,
		`CREATE TABLE "pessoa" (
			"id" INTEGER,
			"nome" TEXT NOT NULL,
			"cpf" TEXT NOT NULL UNIQUE,
			"numero" TEXT NOT NULL,
			"email_insper" TEXT NOT NULL,
			"email_pessoal" TEXT NOT NULL,
			"semestre" INTEGER NOT NULL,
			"curso" TEXT NOT NULL,
			PRIMARY KEY("id" AUTOINCREMENT)
			);`,

		// ------------------------------------------------------------------
		`DROP TABLE IF EXISTS opcoes_horario;`,
		`CREATE TABLE "opcoes_horario" (
		"id" INTEGER PRIMARY KEY AUTOINCREMENT,
		"opcao" TEXT NOT NULL
		);`,

		// ------------------------------------------------------------------
		`DROP TABLE IF EXISTS disponibilidade;`,
		`CREATE TABLE "disponibilidade" (
			"id" INTEGER,
			"pessoa_id" INTEGER NOT NULL,
			"horario_id" INTEGER NOT NULL,
			"preferencia" INTEGER NOT NULL,
			PRIMARY KEY("id" AUTOINCREMENT),
			FOREIGN KEY("horario_id") REFERENCES "opcoes_horario"("id"),
			FOREIGN KEY("pessoa_id") REFERENCES "pessoa"("id")
			);`,
		// ------------------------------------------------------------------
		`DROP TABLE IF EXISTS restricoes;`,
		`CREATE TABLE "restricoes" (
			"id" INTEGER NOT NULL UNIQUE PRIMARY KEY AUTOINCREMENT,
			"candidato_id" INTEGER NOT NULL,
			"naoPosso" TEXT ,
			"prefiroNao" TEXT ,
			FOREIGN KEY("candidato_id") REFERENCES "pessoa"("id") ON DELETE CASCADE);`,
	}

	for _, stmt := range statements {
		_, err := db.Exec(stmt)
		if err != nil {
			log.Fatalf("Erro ao executar statement:\n%s\nErro: %v", stmt, err)
		}
	}

	log.Println("Banco de dados criado com sucesso: casoTeste.db")
}
