package db

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

const DefaultDatabasePath = "./insper.db"

func OpenDefault() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", DefaultDatabasePath)
	if err != nil {
		return nil, fmt.Errorf("erro ao abrir banco default %s: %w", DefaultDatabasePath, err)
	}

	return db, nil
}

func EnsureDefaultDatabase() error {
	db, err := OpenDefault()
	if err != nil {
		return err
	}
	defer db.Close()

	if err := EnsureAppSchema(db); err != nil {
		return fmt.Errorf("erro ao sincronizar schema do banco: %w", err)
	}

	return nil
}
