package main

import (
	dbpkg "candidate_alocator/back/db"
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

	if err := dbpkg.EnsureAppSchema(db); err != nil {
		log.Fatalf("Erro ao sincronizar schema do banco: %v", err)
	}

	log.Println("Schema do banco sincronizado com sucesso: insper.db")
}
