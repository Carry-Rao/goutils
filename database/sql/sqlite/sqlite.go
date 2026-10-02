// Package sqlite provides a SQLite-backed database.Database. The driver is
// embedded, so no server is needed.
package sqlite

import (
	"database/sql"

	"github.com/Carry-Rao/goutils/database"

	_ "github.com/mattn/go-sqlite3"
)

type Database[T any] struct {
	database.Backend[T]
}

func init() { database.Register("sqlite") }

func NewDatabase[T any](cfg map[string]string) (*Database[T], error) {
	db, err := sql.Open("sqlite3", cfg["path"])
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	return &Database[T]{database.Backend[T]{DB: db, Dialect: database.SQLiteDialect}}, nil
}
