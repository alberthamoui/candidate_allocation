package main

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

// setupConn inicializa o schema em um banco já aberto (ex.: :memory: por sessão).
func setupConn(db *sql.DB) error {
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
			"opcao" TEXT NOT NULL UNIQUE
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

		`CREATE TABLE IF NOT EXISTS "restricoesNposso" (
			"id" INTEGER NOT NULL UNIQUE PRIMARY KEY AUTOINCREMENT,
			"avaliador_id" INTEGER NOT NULL,
			"candidato_id" INTEGER NOT NULL,
			FOREIGN KEY("avaliador_id") REFERENCES avaliador("id") ON DELETE CASCADE,
			FOREIGN KEY("candidato_id") REFERENCES "pessoa"("id") ON DELETE CASCADE);`,

		`CREATE TABLE IF NOT EXISTS "restricoesPrefiroN" (
			"id" INTEGER NOT NULL UNIQUE PRIMARY KEY AUTOINCREMENT,
			"avaliador_id" INTEGER NOT NULL,
			"candidato_id" INTEGER NOT NULL,
			FOREIGN KEY("avaliador_id") REFERENCES avaliador("id") ON DELETE CASCADE,
			FOREIGN KEY("candidato_id") REFERENCES "pessoa"("id") ON DELETE CASCADE);`,
	}
	for _, stmt := range statements {
		if _, err := db.Exec(stmt); err != nil {
			primeiraLinha, _, _ := strings.Cut(stmt, "\n")
			return fmt.Errorf("criando o schema (%s): %w", strings.TrimSpace(primeiraLinha), err)
		}
	}
	return nil
}
