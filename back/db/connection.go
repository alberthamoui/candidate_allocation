package db

import "database/sql"

// Connection is the shared SQL surface of a database and a transaction.
type Connection interface {
	Exec(string, ...any) (sql.Result, error)
	QueryRow(string, ...any) *sql.Row
	Query(string, ...any) (*sql.Rows, error)
}
