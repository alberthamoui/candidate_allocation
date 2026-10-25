package db

import (
	"database/sql"
	"fmt"
	"os"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

const DefaultDatabasePath = "./insper.db"

func OpenDefault() (*sql.DB, error) {
	path := os.Getenv("CANDIDATE_ALLOCATOR_DATABASE")
	if path == "" {
		path = DefaultDatabasePath
	}
	separator := "?"
	if strings.Contains(path, "?") {
		separator = "&"
	}
	db, err := sql.Open("sqlite3", path+separator+"_foreign_keys=on&_busy_timeout=5000")
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
	defer func() { _ = db.Close() }()

	if err := EnsureAppSchema(db); err != nil {
		return fmt.Errorf("erro ao sincronizar schema do banco: %w", err)
	}

	return nil
}
