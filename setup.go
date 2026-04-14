package main

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

func SetUp() {
	db, err := sql.Open("sqlite3", "./insper.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	statements := []string{

		`PRAGMA foreign_keys = OFF;`,
		`DROP TABLE IF EXISTS disponibilidade;`,
		`DROP TABLE IF EXISTS restricoes;`,
		`DROP TABLE IF EXISTS opcoes_horario;`,
		`DROP TABLE IF EXISTS pessoa;`,
		`DROP TABLE IF EXISTS avaliador;`,

		`PRAGMA foreign_keys = ON;`,

		// ------------------------------------------------------------------
		`CREATE TABLE "avaliador" (
			"id" INTEGER NOT NULL UNIQUE,
			"nome" TEXT NOT NULL UNIQUE,
			"email" TEXT NOT NULL UNIQUE,
			"sigla" TEXT NOT NULL UNIQUE,
			PRIMARY KEY("id" AUTOINCREMENT)
			);`,

		// ------------------------------------------------------------------
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
		`CREATE TABLE "opcoes_horario" (
		"id" INTEGER PRIMARY KEY AUTOINCREMENT,
		"opcao" TEXT NOT NULL
		);`,

		// ------------------------------------------------------------------
		`CREATE TABLE "disponibilidade" (
			"id" INTEGER,
			"pessoa_id" INTEGER NOT NULL,
			"horario_id" INTEGER NOT NULL,
			"preferencia" INTEGER NOT NULL,
			PRIMARY KEY("id" AUTOINCREMENT),
			FOREIGN KEY("horario_id") REFERENCES "opcoes_horario"("id"),
			FOREIGN KEY("pessoa_id") REFERENCES "pessoa"("id")
			UNIQUE("pessoa_id", "horario_id", "preferencia")

			);`,
		// ------------------------------------------------------------------
		`CREATE TABLE "restricoes" (
			"id" INTEGER NOT NULL UNIQUE PRIMARY KEY AUTOINCREMENT,
			"candidato_id" INTEGER NOT NULL,
			"naoPosso" TEXT,
			"prefiroNao" TEXT,
			FOREIGN KEY("candidato_id") REFERENCES "pessoa"("id") ON DELETE CASCADE);`,
	}

	for _, stmt := range statements {
		_, err := db.Exec(stmt)
		if err != nil {
			log.Fatalf("Erro ao executar statement:\n%s\nErro: %v", stmt, err)
		}
	}

	log.Println("Banco de dados criado com sucesso: insper.db")
}

// setupIfNeeded creates tables only if they do not yet exist.
// Safe to call on every app startup – does NOT wipe existing data.
func setupIfNeeded() {
	db, err := sql.Open("sqlite3", "./insper.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	statements := []string{
		`PRAGMA foreign_keys = ON;`,

		`CREATE TABLE IF NOT EXISTS "avaliador" (
			"id" INTEGER NOT NULL UNIQUE,
			"nome" TEXT NOT NULL UNIQUE,
			"email" TEXT NOT NULL UNIQUE,
			"sigla" TEXT NOT NULL UNIQUE,
			PRIMARY KEY("id" AUTOINCREMENT)
			);`,

		`CREATE TABLE IF NOT EXISTS "pessoa" (
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

		`CREATE TABLE IF NOT EXISTS "opcoes_horario" (
			"id" INTEGER PRIMARY KEY AUTOINCREMENT,
			"opcao" TEXT NOT NULL
			);`,

		`CREATE TABLE IF NOT EXISTS "disponibilidade" (
			"id" INTEGER,
			"pessoa_id" INTEGER NOT NULL,
			"horario_id" INTEGER NOT NULL,
			"preferencia" INTEGER NOT NULL,
			PRIMARY KEY("id" AUTOINCREMENT),
			FOREIGN KEY("horario_id") REFERENCES "opcoes_horario"("id"),
			FOREIGN KEY("pessoa_id") REFERENCES "pessoa"("id"),
			UNIQUE("pessoa_id", "horario_id", "preferencia")
			);`,

		`CREATE TABLE IF NOT EXISTS "restricoes" (
			"id" INTEGER NOT NULL UNIQUE PRIMARY KEY AUTOINCREMENT,
			"candidato_id" INTEGER NOT NULL,
			"naoPosso" TEXT,
			"prefiroNao" TEXT,
			FOREIGN KEY("candidato_id") REFERENCES "pessoa"("id") ON DELETE CASCADE);`,
	}

	for _, stmt := range statements {
		if _, err := db.Exec(stmt); err != nil {
			log.Fatalf("Erro ao executar statement:\n%s\nErro: %v", stmt, err)
		}
	}
}
